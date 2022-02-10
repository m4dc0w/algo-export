package main

import (
	"encoding/base64"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/client/v2/common"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/m4dc0w/algo-export/exporter"
)

type accountList []types.Address

func (al *accountList) String() string {
	return fmt.Sprint(*al)
}

func (al *accountList) Set(value string) error {
	*al = accountList{}
	for _, val := range strings.Split(value, ",") {
		address, err := types.DecodeAddress(val)
		if err != nil {
			return fmt.Errorf("address:%v not valid: %w", address, err)
		}
		*al = append(*al, address)
	}
	return nil
}

func main() {
	var (
		accounts         accountList
		formatFlag       = flag.String("f", exporter.Formats()[0], fmt.Sprintf("Format to export: [%s]", strings.Join(exporter.Formats(), ", ")))
		hostAddrFlag     = flag.String("s", "localhost:8980", "Index server to connect to")
		apiKey           = flag.String("api", "", "Optional API Key for local indexer, or for PureStake")
		pureStakeApiFlag = flag.Bool("p", false, "Use PureStake API - ignoring -s argument")
		outDirFlag       = flag.String("o", "", "output directory path for exported files")
	)
	flag.Var(&accounts, "a", "Account or list of comma delimited accounts to export")
	flag.Parse()

	if len(accounts) == 0 {
		fmt.Println("One or more account addresses to export must be specified.")
		flag.Usage()
		os.Exit(1)
	}
	var export = exporter.GetFormatter(*formatFlag)
	if export == nil {
		fmt.Println("Unable to find formatter for:", *formatFlag)
		fmt.Println("Valid formats are:\n", strings.Join(exporter.Formats(), "\n "))
		os.Exit(1)
	}

	client, err := getClient(*hostAddrFlag, *apiKey, *pureStakeApiFlag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !fileExist(*outDirFlag) {
		if err = os.MkdirAll(*outDirFlag, 0755); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if err := exportAccounts(client, export, accounts, *outDirFlag); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getClient(serverFlag string, apiKey string, usePureStake bool) (*indexer.Client, error) {
	var (
		client     *indexer.Client
		serverAddr *url.URL
		err        error
	)
	if !usePureStake {
		serverAddr, err = url.Parse(fmt.Sprintf("http://%s", serverFlag))
		if err != nil {
			return nil, fmt.Errorf("error in server address: %w", err)
		}
		client, err = indexer.MakeClient(serverAddr.String(), apiKey)
		if err != nil {
			return nil, fmt.Errorf("error creating indexer client: %w", err)
		}
	} else {
		commonClient, err := common.MakeClientWithHeaders("https://mainnet-algorand.api.purestake.io/idx2", "X-API-Key", apiKey, []*common.Header{})
		if err != nil {
			return nil, fmt.Errorf("error creating indexer client to purestake: %w", err)
		}
		client = (*indexer.Client)(commonClient)
	}
	return client, err
}

func exportTransactions(client *indexer.Client, export exporter.Interface, account string, outCsv io.Writer, assetMap map[uint64]models.Asset, topTxID string, txns []models.Transaction) error {
	var records []exporter.ExportRecord

	for index, tx := range txns {
		if topTxID != "" {
			topTxID = fmt.Sprintf("%d-%s", index, topTxID)  // Keep an unique id each inner transaction.
		}
		// Recursive export of inner transactions.
		if len(tx.InnerTxns) > 0 {
			var uniqueTxID string
			if topTxID == "" {
				uniqueTxID = "inner-" + tx.Id  // Initialize to top level transaction id.
			}
			fmt.Printf("    processing %d inner transaction(s) for transaction id: %s\n", len(tx.InnerTxns), tx.Id)
			if err := exportTransactions(client, export, account, outCsv, assetMap, uniqueTxID, tx.InnerTxns); err != nil {
				return err
			}
		}

		// Populate assetMap if entry does not exist.
		if tx.AssetTransferTransaction.AssetId != 0 {
			if _, ok := assetMap[tx.AssetTransferTransaction.AssetId]; !ok {
				// Rate limited to <1 request per second.
				time.Sleep(2 * time.Second)

				lookupASA := client.LookupAssetByID(tx.AssetTransferTransaction.AssetId)
				_, asset, err := lookupASA.Do(context.TODO())
				if err != nil {
					return fmt.Errorf("error looking up asset id: %w", err)
				}
				fmt.Printf("    looked up | Asset ID: %d | UnitName: %s | Name: %s | Decimals: %d |\n", asset.Index, asset.Params.UnitName, asset.Params.Name, asset.Params.Decimals)
				assetMap[tx.AssetTransferTransaction.AssetId] = asset
			}
		}
		records = append(records, exporter.FilterTransaction(tx, topTxID, account, assetMap)...)
	}

	// Applications (e.g. DeFi, Liquidity Pool) are usually part of a Group transaction.
	if len(txns) > 1 && len(txns[0].Group) > 0 {
		// Group ID.
		groupID := base64.StdEncoding.EncodeToString(txns[0].Group)
		appl, err := exporter.ExtractApplication(txns)
		if err != nil {
			fmt.Printf("error finding application: %w", err)
		}
		err = nil

		switch {
			// https://docs.tinyman.org/contracts
			// Version 1.1 - Mainnet Validator App ID: 552635992
			// Version 1.0 - Mainnet Validator App ID: 350338509
			case appl.ApplicationId == 552635992 || appl.ApplicationId == 350338509:
				fmt.Printf("Processing tinyman transaction %d for group id: %s\n", appl.ApplicationId, groupID)
				records, err = exporter.ApplTinyman(assetMap, records)
			default:
				fmt.Printf("Skipping application ID %d for group id: %s\n", appl.ApplicationId, groupID)
		}

		if err != nil {
			fmt.Printf("error exporting application ID %d: %w", appl.ApplicationId, err)
			return err
		}
	}
	// ASA Airdrops are usually done in 1 ASA deposit transaction.
	if len(records) == 1 && records[0].IsASADeposit() {
		var err error
		records, err = exporter.AirdropASA(records)
		if err != nil {
			return err
		}
	}
	// Assume Mining transactions are usually done in 1 ASA deposit transaction.
	if len(records) == 1 && records[0].IsASADeposit() {
		var err error
		r := records[0]
		switch {
		case r.IsASAMining(27165954):
			records, err = exporter.MiningPlanets(records)
		}
		if err != nil {
			return err
		}
	}

	for _, record := range records {
		export.WriteRecord(outCsv, assetMap, record)
	}
	return nil
}

func exportAccounts(client *indexer.Client, export exporter.Interface, accounts accountList, outDir string) error {
	state := LoadConfig()
	assetMap := make(map[uint64]models.Asset)

	fmt.Println("Exporting accounts:")
	for _, accountAddress := range accounts {
		// accountAddress contains the non-checksummed internal version - String() provides the
		// version users know - the base32 pubkey w/ checksum
		account := accountAddress.String()

		startRound := state.ForAccount(export.Name(), account).LastRound + 1
		fmt.Println(account, "starting at:", startRound)

		var outCsv *os.File
		var txnsGroup []models.Transaction

		nextToken := ""
		numPages := 1
		for {
			lookupTx := client.LookupAccountTransactions(account)
			lookupTx.MinRound(startRound)
			lookupTx.NextToken(nextToken)
			transactions, err := lookupTx.Do(context.TODO())
			if err != nil {
				return fmt.Errorf("error looking up transactions: %w", err)
			}
			endRound := transactions.CurrentRound
			if numPages == 1 {
				state.ForAccount(export.Name(), account).LastRound = endRound
				outCsv, err = os.Create(filepath.Join(outDir, fmt.Sprintf("%s-%s-%d-%d.csv", export.Name(), account, startRound, endRound)))
				if err != nil {
					return fmt.Errorf("unable to create file: %w", err)
				}
				export.WriteHeader(outCsv)
			}

			numTx := len(transactions.Transactions)
			fmt.Printf("  %v transactions\n", numTx)
			if numTx == 0 {
				break
			}

			for _, tx := range transactions.Transactions {
				if len(tx.Group) == 0 {
					// Export previous group transactions.
					if err := exportTransactions(client, export, account, outCsv, assetMap, "", txnsGroup); err != nil {
						return err
					}
					txnsGroup = nil // Reset group.
					// Export current single transaction.
					if err := exportTransactions(client, export, account, outCsv, assetMap, "", []models.Transaction{tx}); err != nil {
						return err
					}
					continue
				}

				// Transaction is in same group.
				if len(txnsGroup) > 0 && bytes.Equal(tx.Group, txnsGroup[0].Group) {
					txnsGroup = append(txnsGroup, tx)
					continue
				}

				// Current transaction is in different group, so export previous transaction group.
				if err := exportTransactions(client, export, account, outCsv, assetMap, "", txnsGroup); err != nil {
					return err
				}
				txnsGroup = nil // Reset group.
				txnsGroup = append(txnsGroup, tx)
			}

			fmt.Printf("  %v NextToken at Page %d\n", transactions.NextToken, numPages)
			nextToken = transactions.NextToken
			numPages++

			// Rate limited to <1 request per second.
			time.Sleep(2 * time.Second)
		}
	}
	state.SaveConfig()
	return nil
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatalln(err)
	}
	return true
}

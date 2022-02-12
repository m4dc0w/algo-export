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

func toExportRecords(client *indexer.Client, export exporter.Interface, account string, assetMap map[uint64]models.Asset, topTxID string, txns []models.Transaction) ([]exporter.ExportRecord, error) {
	var records []exporter.ExportRecord
	for index, tx := range txns {
		fmt.Printf("  Converting Tx Type: %s | TxID: %s | Sender: %s\n", tx.Type, tx.Id, tx.Sender)
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
			innerRecords, err := toExportRecords(client, export, account, assetMap, uniqueTxID, tx.InnerTxns)
			if err != nil {
				return records, err
			}
			records = append(records, innerRecords...)
		}

		// Populate assetMap if entry does not exist.
		if tx.AssetTransferTransaction.AssetId != 0 {
			if _, ok := assetMap[tx.AssetTransferTransaction.AssetId]; !ok {
				// Rate limited to <1 request per second.
				time.Sleep(2 * time.Second)

				lookupASA := client.LookupAssetByID(tx.AssetTransferTransaction.AssetId)
				_, asset, err := lookupASA.Do(context.TODO())
				if err != nil {
					return records, fmt.Errorf("error looking up asset id: %w", err)
				}
				fmt.Printf("    looked up | Asset ID: %d | UnitName: %s | Name: %s | Decimals: %d |\n", asset.Index, asset.Params.UnitName, asset.Params.Name, asset.Params.Decimals)
				assetMap[tx.AssetTransferTransaction.AssetId] = asset
			}
		}

		records = append(records, exporter.FilterTransaction(tx, topTxID, account, assetMap)...)
	}
	return records, nil
}

func exportTransactions(client *indexer.Client, export exporter.Interface, account string, outCsv io.Writer, assetMap map[uint64]models.Asset, topTxID string, txns []models.Transaction) error {
	fmt.Printf("\nExport %d Transactions\n", len(txns))

	records, err := toExportRecords(client, export, account, assetMap, topTxID, txns)
	if err != nil {
		return err
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

		fmt.Printf("  Processing Application ID: %d | group id: %s\n", appl.ApplicationId, groupID)

		switch appl.ApplicationId {
			// https://docs.tinyman.org/contracts
			// Version 1.1 - Mainnet Validator App ID: 552635992
			// Version 1.0 - Mainnet Validator App ID: 350338509
			case 552635992, 350338509:
				records, err = exporter.ApplTinyman(assetMap, records, txns)

			// Yieldly No-Loss Lottery.
			// https://app.yieldly.finance/algo-prize-game
			case 233725844:
				records, err = exporter.ApplYieldlyAlgoPrizeGame(records, txns)

			// Yieldly Staking Pool one to two.
			case 233725850:	// YLDY -> YLDY/ALGO
				records, err = exporter.ApplYieldlyStakingPoolsYLDYALGO(records, txns)

			// Yieldly Staking Pools one to one.
			// https://app.yieldly.finance/pools			
			case 348079765, // YLDY -> OPUL
				352116819,		// YLDY -> SMILE
				367431051,		// OPUL -> OPUL
				373819681,		// SMILE -> SMILE
				385089192,		// YLDY -> ARCC
				393388133,		// YLDY -> GEMS
				419301793,		// GEMS -> GEMS
				424101057,		// YLDY -> XET
				447336112,		// YLDY -> CHOICE
				464365150,		// CHOICE -> CHOICE
				498747685,		// ARCC -> ARCC
				511597182,		// YLDY -> AKITA
				583357499,		// YLDY -> ARCC
				593126242,		// YLDY -> KTNC
				591414576,		// YLDY -> DEFLY
				593270704,		// YLDY -> TINY
				593289960,		// YLDY -> TREES
				593324268,		// YLDY -> BLOCK
				596950925:		// YLDY -> HDL
				records, err = exporter.ApplYieldlyStakingPools(records, txns)
			
			// Yieldly Liquidity Pools.
			// https://app.yieldly.finance/liquidity-pools
			case 511593477, // AKITA/ALGO LP -> YLDY
				556355279,		// AKTA/ALGO LP -> YLDY
				568949192,		// XET/YLDY LP -> YLDY
				583355704,		// ARCC/YLDY LP -> YLDY
				591416743,		// DEFLY/YLDY LP -> YLDY
				593133882,		// KTNC/YLDY LP -> YLDY
				593278929,		// TINY/YLDY LP -> YLDY
				593294372,		// TREES/YLDY LP -> YLDY
				593337625,		// BLOCK/YLDY LP -> YLDY
				596954871:		// HDL/YLDY LP -> YLDY
				records, err = exporter.ApplYieldlyLiquidityPools(records, txns)

			// AKITA -> AKTA swap
			// https://swap.akita.community/
			// https://algoexplorer.io/application/537279393
			case 537279393:
				records, err = exporter.ApplAkitaTokenSwap(assetMap, records)
			default:
				fmt.Printf("    Noop for Application ID: %d | group id: %s\n", appl.ApplicationId, groupID)
		}

		if err != nil {
			fmt.Printf("error exporting application ID %d: %w", appl.ApplicationId, err)
			return err
		}
	}

	// ASA Airdrops are usually done in 1 ASA deposit transaction.
	if exporter.IsLengthExcludeReward(records, 1) && records[0].IsASADeposit() {
		var err error
		records, err = exporter.AirdropASA(records)
		if err != nil {
			return err
		}
	}

	// Assume Mining transactions are usually done in 1 ASA deposit transaction.
	if exporter.IsLengthExcludeReward(records, 1) && records[0].IsASADeposit() {
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

	// Other Rewards.
	if exporter.IsLengthExcludeReward(records, 1) && records[0].IsDeposit() {
		var err error
		r := records[0]
		switch {
		case r.IsAlgorandGovernance():
			records, err = exporter.RewardsAlgorandGovernance(records)
		}
		if err != nil {
			return err
		}
	}

	for _, record := range records {
		fmt.Printf("Writing %s\n", record.String())
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
				// Transaction is in same group.
				if len(txnsGroup) > 0 && len(tx.Group) > 0 && bytes.Equal(tx.Group, txnsGroup[0].Group) {
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

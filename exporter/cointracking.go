package exporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

func init() {
	registerFormat("cointracking", NewcointrackingExporter)
}

type cointrackingExporter struct {
}

func NewcointrackingExporter() Interface {
	return &cointrackingExporter{}
}

func (k cointrackingExporter) Name() string {
	return "cointracking"
}

func (k *cointrackingExporter) WriteHeader(writer io.Writer) {
	// https://cointracking.info/import/import_csv/
	// If you want to create your own CSV file, please ensure the format is like this:
	// "Type", "Buy Amount", "Buy Currency", "Sell Amount", "Sell Currency", "Fee", "Fee Currency", "Exchange", "Trade-Group", "Comment", "Date"
	// Optionally you can add those 3 columns at the end (after the "Date" column):
	// "Tx-ID", "Buy Value in your Account Currency", "Sell Value in your Account Currency"
	fmt.Fprintln(writer, "Type,Buy Amount,Buy Currency,Sell Amount,Sell Currency,Fee,Fee Currency,Exchange,Trade-Group,Comment,Date,Tx-ID")
}

func (k *cointrackingExporter) WriteRecord(writer io.Writer, assetMap map[uint64]models.Asset, record ExportRecord) {
	// Type,Buy Amount,Buy Currency,Sell Amount,Sell Currency,Fee,Fee Currency,Exchange,Trade-Group,Comment,Date,Tx-ID

	// Type,
	// https://cointracking.freshdesk.com/en/support/solutions/articles/29000034379-expanded-transaction-types-may-2020-
	switch {
	case record.airdrop:
		fmt.Fprintf(writer, "Airdrop,")
	case record.feeTx || record.otherFee:
		fmt.Fprintf(writer, "Other Fee,")
	case record.mining:
		fmt.Fprintf(writer, "Mining,")
	case record.reward:
		fmt.Fprintf(writer, "Reward / Bonus,")
	case record.spend:
		fmt.Fprintf(writer, "Spend,")
	case record.staking:
		fmt.Fprintf(writer, "Staking,")
	case record.IsTrade():
		fmt.Fprintf(writer, "Trade,")
	case record.IsDeposit():
		fmt.Fprintf(writer, "Deposit,")
	default:
		fmt.Fprintf(writer, "Withdrawal,")
	}
	// Buy Amount,Buy Currency,
	switch {
	case record.recvCustomQty != "" && record.recvCustomCurrency != "":
		fmt.Fprintf(writer, "%s,%s,", record.recvCustomQty, record.recvCustomCurrency)
	case record.recvQty != 0:
		fmt.Fprintf(writer, "%s,%s,", assetIDFmt(record.recvQty, record.recvASA, assetMap), asaFmt(record.recvASA, assetMap))
	default:
		fmt.Fprintf(writer, ",,")
	}
	// Sell Amount,Sell Currency,
	switch {
	case record.sentCustomQty != "" && record.sentCustomCurrency != "":
		fmt.Fprintf(writer, "%s,%s,", record.sentCustomQty, record.sentCustomCurrency)
	case record.sentQty != 0:
		fmt.Fprintf(writer, "%s,%s,", assetIDFmt(record.sentQty, record.sentASA, assetMap), asaFmt(record.sentASA, assetMap))
	default:
		fmt.Fprintf(writer, ",,")
	}
	// Fee,Fee Currency,
	switch {
	case record.feeCustom != "" && record.feeCustomCurrency != "":
		fmt.Fprintf(writer, "%s,%s,", record.feeCustom, record.feeCustomCurrency)
	case record.fee != 0:
		fmt.Fprintf(writer, "%s,ALGO,", algoFmt(record.fee))
	default:
		fmt.Fprintf(writer, ",,")
	}

	// Exchange,
	fmt.Fprintf(writer, "ALGO Wallet,")

	// Trade-Group,
	fmt.Fprintf(writer, "%s,", record.account)

	// Comment,
	var comments []string
	if record.recvASA != 0 {
		comments = append(comments, asaComment(record.recvASA, assetMap))
	}
	if record.sentASA != 0 && record.recvASA != record.sentASA {
		comments = append(comments, asaComment(record.sentASA, assetMap))
	}
	if record.comment != "" {
		comments = append(comments, record.comment)
	}
	fmt.Fprintf(writer, "%q,", strings.Join(comments, " | "))

	// Date,
	fmt.Fprint(writer, record.blockTime.UTC().Format("2006-01-02T15:04:05Z,"))

	// Tx-ID,
	switch {
	case record.topTxID != "":
		fmt.Fprintf(writer, "%s_%s", record.topTxID, record.account[:10])
	default:
		fmt.Fprintf(writer, "%s_%s", record.txid, record.account[:10])
	}
	switch {
	case record.airdrop:
		fmt.Fprintf(writer, "_airdrop")
	case record.appl:
		fmt.Fprintf(writer, "_appl")
	case record.feeTx:
		fmt.Fprintf(writer, "_fee")
	case record.mining:
		fmt.Fprintf(writer, "_mining")
	case record.reward:
		fmt.Fprintf(writer, "_reward")
	}
	fmt.Fprint(writer, "\n")
}

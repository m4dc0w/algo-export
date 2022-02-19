package exporter

import (
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

// https://docs.algofi.org/protocol/mainnet-contracts
// https://cointracking.freshdesk.com/en/support/solutions/articles/29000033408-loans-and-their-repayments

// https://app.algofi.org/
func ApplAlgoFiMarket(records []ExportRecord, txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := ExtractFirstArg(txns)

	// Supply
	if action == "mt" {
		records[0].comment = "AlgoFi - Supply"
		return records, nil
	}

	// Withdraw
	if action == "rcu" {
		records[0].comment = "AlgoFi - Withdraw"
		return records, nil
	}

	// Borrow
	if action == "b" {
		records[0].incomeNoTax = true
		records[0].comment = "AlgoFi - Borrow"
		return records, nil
	}

	// Repay
	if action == "rb" {
		records[0].expenseNoTax = true
		records[0].comment = "AlgoFi - Repay"
		return records, nil
	}

	return records, fmt.Errorf("invalid ApplAlgoFiMarket() record | onCompletion: %s | action: %s | records length: %d | txns length: %d\n", onCompletion, action, len(records), len(txns))
}

package exporter

import (
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

// https://docs.algofi.org/protocol/mainnet-contracts
// https://cointracking.freshdesk.com/en/support/solutions/articles/29000033408-loans-and-their-repayments
func ApplAlgoFi(records []ExportRecord,  txns []models.Transaction) ([]ExportRecord, error) {
	return records, nil
}

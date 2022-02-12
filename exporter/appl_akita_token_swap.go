package exporter

import (
	"fmt"
)

// ApplAkitaTokenSwap exports Akita Token Swap.
// AKITA -> AKTA swap
// https://swap.akita.community/
// https://algoexplorer.io/application/537279393
func ApplAkitaTokenSwap(records []ExportRecord) ([]ExportRecord, error) {
	var processed []ExportRecord
	if !IsLengthExcludeReward(records, 5) {
		return records, fmt.Errorf("invalid ApplAkitaTokenSwap() record")
	}

	r := records[0]
	r.appl = true
	r.trade = true
	r.sentQty = records[2].sentQty
	r.sentASA = records[2].sentASA
	r.comment = "Akita Token Swap"
	processed = append(processed, r)
	processed = append(processed, records[1])
	processed = append(processed, records[3:]...)
	return processed, nil
}

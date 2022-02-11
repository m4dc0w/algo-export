package exporter

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// AirdropASA exports transactions as an airdrop based on certain criteria.
func AirdropASA(records []ExportRecord) ([]ExportRecord, error) {
	// Assume Airdrops are usually done in 1 ASA deposit transaction.
	if !IsLengthExcludeReward(records, 1) || !records[0].IsASADeposit() {
		return records, fmt.Errorf("invalid ASAAirdrop() record")
	}

	r := records[0]

	// General YLDY team airdrop address.
	// Example 4587 YLDY Airdrop on 2021-07-04.
	//   https://twitter.com/YieldlyFinance/status/1411550660391096322
	//   https://yieldly.medium.com/yldy-rewards-review-and-thank-you-yieldly-89bf871b591f
	if r.sender == "LWWSLXSOC2J3HMNXYPWSMGIJ4A2BRVO65LLL5IU374R24IWV6NIKCT2ZGA" && r.sender != r.account {
		records[0].airdrop = true
		records[0].comment = "YLDY Team Airdrop"
		return records, nil
	}

	if len(r.txRaw.Note) == 0 {
		return records, nil
	}

	// Use tx note contents to determine if tx is an airdrop.
	note := base64.StdEncoding.EncodeToString(r.txRaw.Note)
	decoded, err := base64.StdEncoding.DecodeString(note)
	if err != nil {
		fmt.Println("decode error:", err)
		return records, err
	}

	comment := string(decoded)
	// https://www.freckletoken.com/tools/airdrop
	// Example note:
	//   "ASA Drop" - powered by Freckle Token airdrop tool
	if strings.Contains(comment, "- powered by Freckle Token airdrop tool") {
		records[0].airdrop = true
		records[0].comment = comment
	}
	return records, nil
}

package exporter

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func (r ExportRecord) IsAlgorandGovernance() bool {
	if !r.IsALGODeposit() {
		return false
	}

	if r.sender == r.account {
		return false
	}

	// Check if sender is an Algorand governance address.
	switch r.sender {
	case "GULDQIEZ2CUPBSHKXRWUW7X3LCYL44AI5GGSHHOQDGKJAZ2OANZJ43S72U",  // Governance Period 1
		"57QZ4S7YHTWPRAM3DQ2MLNSVLAQB7DTK4D7SUNRIEFMRGOU7DMYFGF55BY":			// Governance Period 2
		return true
	}
	return false
}

// Algorand Governance.
// https://governance.algorand.foundation/
func RewardsAlgorandGovernance(records []ExportRecord) ([]ExportRecord, error) {	
	// Assume Algorand Governance transactions are done in 1 deposit transaction.
	if !IsLengthExcludeReward(records, 1) || !records[0].IsAlgorandGovernance() {
		return records, fmt.Errorf("invalid RewardsAlgoGovernance() record")
	}

	r := records[0]

	if len(r.txRaw.Note) == 0 {
		return records, nil
	}

	// Use tx note contents to determine if tx is a Governance reward.
	note := base64.StdEncoding.EncodeToString(r.txRaw.Note)
	decoded, err := base64.StdEncoding.DecodeString(note)
	if err != nil {
		fmt.Println("decode error:", err)
		return records, err
	}

	comment := string(decoded)
	// Example note:
	//   af/gov1:j{"rewardsPrd":1,"idx":12345}
	if strings.HasPrefix(comment, `af/gov1:j{"rewardsPrd":`) {
		records[0].reward = true
		records[0].comment = strings.Join([]string{"Algorand Governance Rewards", comment}, " | ")
	}
	return records, nil
}

func (r ExportRecord) IsAlgoStake() bool {
	if !r.IsASADeposit() {
		return false
	}

	if r.sender == r.account {
		return false
	}

	// AlgoStake Wallet
	if r.sender == "4ZK3UPFRJ643ETWSWZ4YJXH3LQTL2FUEI6CIT7HEOVZL6JOECVRMPP34CY" {
		return true
	}
	return false
}

func RewardsAlgoStake(records []ExportRecord) ([]ExportRecord, error) {	
	if !IsLengthExcludeReward(records, 1) || !records[0].IsAlgoStake() {
		return records, fmt.Errorf("invalid RewardsAlgoStake() record")
	}
	records[0].staking = true
	records[0].comment = "AlgoStake"
	return records, nil
}

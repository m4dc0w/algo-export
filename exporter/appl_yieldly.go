package exporter

import (
	"fmt"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

// https://app.yieldly.finance/algo-prize-game
func ApplYieldlyAlgoPrizeGame(records []ExportRecord,  txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := ExtractFirstArg(txns)

	// Claim
	if action == "CA" && IsLengthExcludeReward(records, 4) && records[1].IsDeposit() {
		records[0].otherFee = true
		records[1].reward = true
		records[1].comment = "Claim - Yieldly - ALGO Weekly Prize Game"
		return records, nil
	}

	// Deposit
	if action == "D" && IsLengthExcludeReward(records, 3) && records[0].IsWithdrawal() {
		records[0].comment = "Deposit - Yieldly - ALGO Weekly Prize Game"
		return records, nil
	}
	
	// Withdrawal
	if action == "W" && IsLengthExcludeReward(records, 4) && records[1].IsDeposit() {
		records[0].otherFee = true
		records[1].comment = "Withdraw - Yieldly - ALGO Weekly Prize Game"
		return records, nil
	}

	return records, fmt.Errorf("invalid ApplYieldlyAlgoPrizeGame() record | onCompletion: %s | action: %s | records length: %d | txns length: %d", onCompletion, action, len(records), len(txns))
}

// https://app.yieldly.finance/distribution
func ApplYieldlyDistributionPools(records []ExportRecord, txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := ExtractFirstArg(txns)

	// Claim
	if action == "CA" && IsLengthExcludeReward(records, 3) && records[0].IsDeposit() {
		records[0].reward = true
		records[0].comment = "Claim - Yieldly - Distribution Pools"
		return records, nil
	}

	// Stake
	if action == "S" && IsLengthExcludeReward(records, 4) && records[0].IsWithdrawal() {
		records[0].comment = "Stake - Yieldly - Distribution Pools"
		return records, nil
	}

	// Withdraw
	if action == "W" && IsLengthExcludeReward(records, 3) && records[0].IsDeposit() {
		records[0].comment = "Withdraw - Yieldly - Distribution Pools"
		return records, nil
	}
	
	// Opt-out is not implemented.

	return records, fmt.Errorf("invalid ApplYieldlyDistributionPools() record | onCompletion: %s | action: %s | records length: %d | txns length: %d", onCompletion, action, len(records), len(txns))
}

func applYieldyStakingPoolArg(txns []models.Transaction) (string, string) {
	// Yieldly staking pool has the "claim", "stake", and "withdraw" action in the 2nd argument.
	// If arg is empty and OnCompletion is Opt-Out.
	for _, txn := range txns {
		if txn.Type == "appl" {
			appl := txn.ApplicationTransaction
			if appl.OnCompletion != "noop" {
				return appl.OnCompletion, ""  // e.g. closeout
			}

			// ApplicationArgs (apaa) transaction specific arguments.
			for _, appa := range appl.ApplicationArgs {
				// Skip the first arg.
				if string(appa) == "bail" {
					break
				}
				return appl.OnCompletion, string(appa)
			}
		}
	}
	return "", ""
}

func ApplYieldlyStakingPoolsYLDYALGO(records []ExportRecord, txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := applYieldyStakingPoolArg(txns)
	
	// Claim on TEAL4 contracts.
	if action == "CAL" && IsLengthExcludeReward(records, 6) && records[1].IsDeposit() && records[2].IsDeposit() {
		records[1].reward = true
		records[1].comment = "Claim - Yieldly - Staking Pools"
		records[2].reward = true
		records[2].comment = "Claim - Yieldly - Staking Pools"
		return records, nil
	}

	// Stake on TEAL4 contracts.
	if action == "S" && IsLengthExcludeReward(records, 4) && records[0].IsWithdrawal() {
		records[0].comment = "Stake - Yieldly - Staking Pools"
		return records, nil
	}


	// Withdraw on TEAL4 contracts.
	if action == "W" && IsLengthExcludeReward(records, 4) && records[1].IsDeposit() {
		records[0].otherFee = true
		records[1].comment = "Withdraw - Yieldly - Staking Pools"
		return records, nil
	}
	
	// Opt-out is not implemented.

	return records, fmt.Errorf("invalid ApplYieldlyStakingPoolsYLDYALGO() record | onCompletion: %s | action: %s | records length: %d | txns length: %d", onCompletion, action, len(records), len(txns))
}

// https://app.yieldly.finance/liquidity-pools
func ApplYieldlyLiquidityPools(records []ExportRecord, txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := applYieldyStakingPoolArg(txns)

	// Claim.
	if action == "claim" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].reward = true
		records[1].comment = "Claim - Yieldly - Liquidity Pools"
		return records, nil
	}

	// Stake.
	if action == "stake" && IsLengthExcludeReward(records, 4) && records[1].IsWithdrawal() {
		records[1].comment = "Stake - Yieldly - Liquidity Pools"
		return records, nil
	}

	// Withdrawal.
	if action == "withdraw" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].comment = "Withdraw - Yieldly - Liquidity Pools"
		return records, nil
	}
	
	if onCompletion == "closeout" && (IsLengthExcludeReward(records, 3) ||  IsLengthExcludeReward(records, 4)) {
		processed := records
		for i, r := range records {
			if strings.HasPrefix(r.topTxID, "0-0-inner-") {
				processed[i].comment = "Opt-out Withdraw - Yieldly - Liquidity Pools"
			}
			if strings.HasPrefix(r.topTxID, "1-0-inner-") {
				processed[i].reward = true
				processed[i].comment = "Opt-out Claim - Yieldly - Liquidity Pools"
			}
		}
		return records, nil
	}	
	return records, fmt.Errorf("invalid ApplYieldlyLiquidityPools() record")
}

// https://app.yieldly.finance/nft
func ApplYieldlyNFTPrizeGames(assetMap map[uint64]models.Asset, records []ExportRecord) ([]ExportRecord, error) {
	return records, nil
}

// https://app.yieldly.finance/pools
func ApplYieldlyStakingPools(records []ExportRecord, txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := applYieldyStakingPoolArg(txns)

	// Claim on TEAL5 contracts.
	if action == "claim" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].reward = true
		records[1].comment = "Claim - Yieldly - Staking Pools"
		return records, nil
	}

	// Stake on TEAL5 contracts.
	if action == "stake" && IsLengthExcludeReward(records, 4) && records[1].IsWithdrawal() {
		records[1].comment = "Stake - Yieldly - Staking Pools"
		return records, nil
	}

	// Withdrawal on TEAL5 contracts.
	if action == "withdraw" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].comment = "Withdraw - Yieldly - Staking Pools"
		return records, nil
	}

	// Opt-out on TEAL4 & TEAL5 contracts.
	if onCompletion == "closeout" && IsLengthExcludeReward(records, 4) {
		records[0].comment = "Opt-out Withdraw - Yieldly - Staking Pools"
		records[1].reward = true
		records[1].comment = "Opt-out Claim - Yieldly - Staking Pools"
		return records, nil
	}
	if onCompletion == "closeout" && IsLengthExcludeReward(records, 3) {
		// Use txns to identify which transaction is withdraw vs claim.
		switch {
		case txns[0].Id == records[0].txid:	// 1st txn is withdraw.
			records[0].comment = "Opt-out Withdraw - Yieldly - Staking Pools"
		case txns[1].Id == records[0].txid: // 2nd txn is claim rewards.
			records[0].reward = true
			records[0].comment = "Opt-out Claim - Yieldly - Staking Pools"
		}
		return records, nil
	}

	// Claim on TEAL4 contracts.
	if action == "CA" && IsLengthExcludeReward(records, 3) && records[0].IsDeposit() {
		records[0].reward = true
		records[0].comment = "Claim - Yieldly - Staking Pools"
		return records, nil
	}

	// Stake on TEAL4 contracts.
	if action == "S" && IsLengthExcludeReward(records, 4) && records[0].IsWithdrawal() {
		records[0].comment = "Stake - Yieldly - Staking Pools"
		return records, nil
	}

	// Withdraw on TEAL4 contracts.
	if action == "W" && IsLengthExcludeReward(records, 3) && records[0].IsDeposit() {
		records[1].comment = "Withdraw - Yieldly - Staking Pools"
		return records, nil
	}

	return records, fmt.Errorf("invalid ApplYieldlyStakingPools() record | onCompletion: %s | action: %s | records length: %d | txns length: %d", onCompletion, action, len(records), len(txns))
}

package exporter

import (
	"fmt"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

// ApplTinyman exports Tinyman Liquidity Pool transactions.
// https://docs.tinyman.org/contracts
// Version 1.1 - Mainnet Validator App ID: 552635992
//   https://algoexplorer.io/application/552635992
// Version 1.0 - Mainnet Validator App ID: 350338509
//   https://algoexplorer.io/application/350338509
// Treat Tinyman LP as a Split Trade.
// https://cointracking.freshdesk.com/en/support/solutions/articles/29000038185-how-are-liquidity-pool-transactions-imported-
// https://cointracking.freshdesk.com/en/support/solutions/articles/29000037542
func ApplTinyman(assetMap map[uint64]models.Asset, records []ExportRecord) ([]ExportRecord, error) {
	var processed []ExportRecord
	// Redeem Excess Amounts.
	if IsLengthExcludeReward(records, 2) && records[0].receiver == records[0].account {
		processed = records
		processed[0].reward = true
		processed[0].comment = "Tinyman Redeem Excess Amounts"
		processed[1].spend = true
		return processed, nil
	}

	// Swap ALGO -> ASA
	if IsLengthExcludeReward(records, 3) && records[0].receiver == records[0].account {
		r := records[0]
		r.appl = true
		r.trade = true
		r.sentQty = records[1].sentQty
		r.sentASA = records[1].sentASA
		if records[1].sentASA == 0 {
			r.fee = records[1].fee  // Put fees in same record when trading ALGO -> ASA.
		}
		r.comment = "Tinyman Swap"
		processed = append(processed, r)
		processed = append(processed, records[2:]...)
		processed[1].spend = true
		return processed, nil
	}

	// Swap ASA -> ASA or ASA -> ALGO
	if IsLengthExcludeReward(records, 4) && records[0].receiver == records[0].account {
		r := records[0]
		r.appl = true
		r.trade = true
		r.sentQty = records[1].sentQty
		r.sentASA = records[1].sentASA

		r.comment = "Tinyman Swap"
		processed = append(processed, r)
		processed = append(processed, records[2:]...)
		processed[2].spend = true
		return processed, nil
	}

	// Deposit ASA-ALGO Liquidity Pool.
	if IsLengthExcludeReward(records, 5) && records[0].receiver == records[0].account {
		r1 := records[0]
		r1.txid = records[1].txid
		r1.appl = true
		r1.trade = true
		r1.recvQty = r1.recvQty / 2
		r1.sentQty = records[1].sentQty
		r1.sentASA = records[1].sentASA
		if records[1].sentASA == 0 {
			r1.fee = records[1].fee  // Put fees in same record when trading ALGO -> ASA.
		}
		r1.comment = "Tinyman Liquidity Pool Deposit"
		processed = append(processed, r1)
		r2 := records[0]
		r2.txid = records[2].txid
		r2.appl = true
		r2.trade = true
		r2.recvQty = records[0].recvQty - r1.recvQty
		r2.sentQty = records[2].sentQty
		r2.sentASA = records[2].sentASA
		r2.comment = "Tinyman Liquidity Pool Deposit"
		processed = append(processed, r2)
		processed = append(processed, records[3:]...)
		processed[3].spend = true
		return processed, nil
	}
	// Deposit ASA-ASA Liquidity Pool.
	if IsLengthExcludeReward(records, 6) && records[0].receiver == records[0].account {
		r1 := records[0]
		r1.txid = records[1].txid
		r1.appl = true
		r1.trade = true
		r1.recvQty = r1.recvQty / 2
		r1.sentQty = records[1].sentQty
		r1.sentASA = records[1].sentASA
		r1.comment = "Tinyman Liquidity Pool Deposit"
		processed = append(processed, r1)
		processed = append(processed, records[2])

		r2 := records[0]
		r2.txid = records[3].txid
		r2.appl = true
		r2.trade = true
		r2.recvQty = records[0].recvQty - r1.recvQty
		r2.sentQty = records[3].sentQty
		r2.sentASA = records[3].sentASA
		r2.comment = "Tinyman Liquidity Pool Deposit"
		processed = append(processed, r2)
		processed = append(processed, records[4:]...)
		processed[4].spend = true
		return processed, nil
	}
	// Withdrawal ASA-ASA or ASA-ALGO Liquidity Pool.
	if IsLengthExcludeReward(records, 5) && records[0].sender == records[0].account {
		r1 := records[0]
		r1.txid = records[2].txid
		r1.appl = true
		r1.trade = true
		r1.sentQty = r1.sentQty / 2
		r1.recvQty = records[2].recvQty
		r1.recvASA = records[2].recvASA
		r1.comment = "Tinyman Liquidity Pool Withdrawal"
		processed = append(processed, r1)
		processed = append(processed, records[1])  // Tx Fee.
		r2 := records[0]
		r2.txid = records[3].txid
		r2.appl = true
		r2.trade = true
		r2.sentQty = records[0].sentQty - r1.sentQty
		r2.recvQty = records[3].recvQty
		r2.recvASA = records[3].recvASA
		r2.comment = "Tinyman Liquidity Pool Withdrawal"
		processed = append(processed, r2)
		processed = append(processed, records[4:]...)
		processed[3].spend = true
		return processed, nil
	}
	return processed, fmt.Errorf("error exporting Tinyman application")
}

// ApplAkitaTokenSwap exports Akita Token Swap.
// AKITA -> AKTA swap
// https://swap.akita.community/
// https://algoexplorer.io/application/537279393
func ApplAkitaTokenSwap(assetMap map[uint64]models.Asset, records []ExportRecord) ([]ExportRecord, error) {
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

// https://app.yieldly.finance/algo-prize-game
func ApplYieldlyAlgoPrizeGame(assetMap map[uint64]models.Asset, records []ExportRecord) ([]ExportRecord, error) {
	return records, nil
}

// https://app.yieldly.finance/distribution
func ApplYieldlyDistributionPools(assetMap map[uint64]models.Asset, records []ExportRecord) ([]ExportRecord, error) {
	return records, nil
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

// https://app.yieldly.finance/liquidity-pools
func ApplYieldlyLiquidityPools(records []ExportRecord, txns []models.Transaction) ([]ExportRecord, error) {
	onCompletion, action := applYieldyStakingPoolArg(txns)

	// Claim.
	if action == "claim" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].reward = true
		records[1].comment = "Claim - Yieldly Liquidity Staking Pools"
		return records, nil
	}

	// Stake.
	if action == "stake" && IsLengthExcludeReward(records, 4) && records[1].IsWithdrawal() {
		records[1].comment = "Stake - Yieldly Liquidity Staking Pools"
		return records, nil
	}

	// Withdrawal.
	if action == "withdraw" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].comment = "Withdraw - Yieldly Liquidity Staking Pools"
		return records, nil
	}
	
	if onCompletion == "closeout" && (IsLengthExcludeReward(records, 3) ||  IsLengthExcludeReward(records, 4)) {
		processed := records
		for i, r := range records {
			if strings.HasPrefix(r.topTxID, "0-0-inner-") {
				processed[i].comment = "Opt-out Withdraw - Yieldly Liquidity Staking Pools"
			}
			if strings.HasPrefix(r.topTxID, "1-0-inner-") {
				processed[i].reward = true
				processed[i].comment = "Opt-out Claim - Yieldly Liquidity Staking Pools"
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
		records[1].comment = "Claim - Yieldly Staking Pools"
		return records, nil
	}

	// Stake on TEAL5 contracts.
	if action == "stake" && IsLengthExcludeReward(records, 4) && records[1].IsWithdrawal() {
		records[1].comment = "Stake - Yieldly Staking Pools"
		return records, nil
	}

	// Withdrawal on TEAL5 contracts.
	if action == "withdraw" && IsLengthExcludeReward(records, 3) && records[1].IsDeposit() {
		records[1].comment = "Withdraw - Yieldly Staking Pools"
		return records, nil
	}

	// Opt-out on TEAL4 & TEAL5 contracts.
	if onCompletion == "closeout" && IsLengthExcludeReward(records, 4) {
		records[0].comment = "Opt-out Withdraw - Yieldly Staking Pools"
		records[1].reward = true
		records[1].comment = "Opt-out Claim - Yieldly Staking Pools"
		return records, nil
	}
	if onCompletion == "closeout" && IsLengthExcludeReward(records, 3) {
		// Use txns to identify which transaction is withdraw vs claim.
		switch {
		case txns[0].Id == records[0].txid:	// 1st txn is withdraw.
			records[0].comment = "Opt-out Withdraw - Yieldly Staking Pools"
		case txns[1].Id == records[0].txid: // 2nd txn is claim rewards.
			records[0].reward = true
			records[0].comment = "Opt-out Claim - Yieldly Staking Pools"
		}
		return records, nil
	}

	// Claim on TEAL4 contracts.
	if action == "CA" && IsLengthExcludeReward(records, 3) && records[0].IsDeposit() {
		records[0].comment = "Claim - Yieldly Staking Pools"
		return records, nil
	}

	// Stake on TEAL4 contracts.
	if action == "S" && IsLengthExcludeReward(records, 4) && records[0].IsWithdrawal() {
		records[0].comment = "Stake - Yieldly Staking Pools"
		return records, nil
	}

	// Withdraw on TEAL4 contracts.
	if action == "W" && IsLengthExcludeReward(records, 3) && records[0].IsDeposit() {
		records[1].comment = "Withdraw - Yieldly Staking Pools"
		return records, nil
	}

	return records, fmt.Errorf("invalid ApplYieldlyStakingPools() record | onCompletion: %s | action: %s | records length: %d | txns length: %d", onCompletion, action, len(records), len(txns))
}

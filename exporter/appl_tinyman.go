package exporter

import (
	"fmt"

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
func ApplTinyman(records []ExportRecord,  txns []models.Transaction) ([]ExportRecord, error) {
	var processed []ExportRecord
	onCompletion, action := ExtractFirstArg(txns)

	// Redeem Excess Amounts.
	if action == "redeem" && IsLengthExcludeReward(records, 2) && records[0].receiver == records[0].account {
		processed = records
		processed[0].reward = true
		processed[0].comment = "Tinyman Redeem Excess Amounts"
		processed[1].otherFee = true
		return processed, nil
	}

	// Swap ALGO -> ASA.
	if action == "swap" && IsLengthExcludeReward(records, 3) && records[0].receiver == records[0].account {
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
		processed[1].otherFee = true
		return processed, nil
	}

	// Swap ASA -> ASA or ASA -> ALGO
	if action == "swap" && IsLengthExcludeReward(records, 4) && records[0].IsDeposit() {
		r := records[0]
		r.appl = true
		r.trade = true
		r.sentQty = records[1].sentQty
		r.sentASA = records[1].sentASA

		r.comment = "Tinyman Swap"
		processed = append(processed, r)
		processed = append(processed, records[2:]...)
		processed[2].otherFee = true
		return processed, nil
	}

	// Deposit ASA-ALGO Liquidity Pool.
	if action == "mint" && IsLengthExcludeReward(records, 5) && records[0].IsDeposit() {
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
		r2.recvQty = records[0].recvQty - r1.recvQty  // Original qty could be an odd number, so use subtraction instead of dividing by 2.
		r2.sentQty = records[2].sentQty
		r2.sentASA = records[2].sentASA
		r2.comment = "Tinyman Liquidity Pool Deposit"
		processed = append(processed, r2)
		processed = append(processed, records[3:]...)
		processed[3].otherFee = true
		return processed, nil
	}
	// Deposit ASA-ASA Liquidity Pool.
	if action == "mint" && IsLengthExcludeReward(records, 6) && records[0].IsDeposit() {
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
		r2.recvQty = records[0].recvQty - r1.recvQty  // Original qty could be an odd number, so use subtraction instead of dividing by 2.
		r2.sentQty = records[3].sentQty
		r2.sentASA = records[3].sentASA
		r2.comment = "Tinyman Liquidity Pool Deposit"
		processed = append(processed, r2)
		processed = append(processed, records[4:]...)
		processed[4].otherFee = true
		return processed, nil
	}
	// Withdrawal ASA-ASA or ASA-ALGO Liquidity Pool.
	if action == "burn" && IsLengthExcludeReward(records, 5) && records[0].IsWithdrawal() {
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
		processed[3].otherFee = true
		return processed, nil
	}
	return processed, fmt.Errorf("invalid ApplTinyman() record | onCompletion: %s | action: %s | records length: %d | txns length: %d", onCompletion, action, len(records), len(txns))
}

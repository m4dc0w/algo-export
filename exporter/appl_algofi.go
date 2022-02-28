package exporter

import (
	"fmt"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

// https://docs.algofi.org/protocol/mainnet-contracts
// https://cointracking.freshdesk.com/en/support/solutions/articles/29000033408-loans-and-their-repayments
type AlgoFiState struct {
	SupplyALGO  uint64
	SupplySTBL  uint64
	SupplyUSDC  uint64
	SupplygoBTC uint64
	SupplygoETH uint64

	BorrowALGO  uint64
	BorrowSTBL  uint64
	BorrowUSDC  uint64
	BorrowgoBTC uint64
	BorrowgoETH uint64
}

// https://app.algofi.org/
func ApplAlgoFiLend(records []ExportRecord, txns []models.Transaction, assetMap map[uint64]models.Asset, state AlgoFiState) ([]ExportRecord, AlgoFiState, error) {
	onCompletion, action := ExtractFirstArg(txns)

	var processed []ExportRecord

	// Supply
	if action == "mt" {
		r := records[0]
		processed = append(processed, records...)
		processed[0].comment = "AlgoFi - Supply"
		switch asaUnitName(r.sentASA, assetMap) {
		case "ALGO":
			state.SupplyALGO = state.SupplyALGO + r.sentQty - r.fee
		case "STBL":
			state.SupplySTBL = state.SupplySTBL + r.sentQty
		case "USDC":
			state.SupplyUSDC = state.SupplyUSDC + r.sentQty
		case "goBTC":
			state.SupplygoBTC = state.SupplygoBTC + r.sentQty
		case "goETH":
			state.SupplygoETH = state.SupplygoETH + r.sentQty
		default:
			return records, state, fmt.Errorf("invalid ApplAlgoFiMarket() Supply record | onCompletion: %s | action: %s | records length: %d | txns length: %d\n", onCompletion, action, len(records), len(txns))
		}
		return processed, state, nil
	}

	// Withdraw
	if action == "rcu" {
		r := records[0]
		processed = append(processed, records...)
		processed[0].comment = "AlgoFi - Withdraw"
		switch asaUnitName(r.recvASA, assetMap) {
		case "ALGO":
			if r.recvQty > state.SupplyALGO {
				processed[0].recvQty = state.SupplyALGO
				r.recvQty = r.recvQty - state.SupplyALGO
				r.lending = true
				r.comment = "AlgoFi - Withdraw - Lending Income"
				processed = append(processed, r)
				state.SupplyALGO = 0
			} else {
				state.SupplyALGO = state.SupplyALGO - r.recvQty
			}
		case "STBL":
			if r.recvQty > state.SupplySTBL {
				processed[0].recvQty = state.SupplySTBL
				r.recvQty = r.recvQty - state.SupplySTBL
				r.lending = true
				r.comment = "AlgoFi - Withdraw - Lending Income"
				processed = append(processed, r)
				state.SupplySTBL = 0
			} else {
				state.SupplySTBL = state.SupplySTBL - r.recvQty
			}
		case "USDC":
			if r.recvQty > state.SupplyUSDC {
				processed[0].recvQty = state.SupplyUSDC
				r.recvQty = r.recvQty - state.SupplyUSDC
				r.lending = true
				r.comment = "AlgoFi - Withdraw - Lending Income"
				processed = append(processed, r)
				state.SupplyUSDC = 0
			} else {
				state.SupplyUSDC = state.SupplyUSDC - r.recvQty
			}
		case "goBTC":
			if r.recvQty > state.SupplygoBTC {
				processed[0].recvQty = state.SupplygoBTC
				r.recvQty = r.recvQty - state.SupplygoBTC
				r.lending = true
				r.comment = "AlgoFi - Withdraw - Lending Income"
				processed = append(processed, r)
				state.SupplygoBTC = 0
			} else {
				state.SupplygoBTC = state.SupplygoBTC - r.recvQty
			}
		case "goETH":
			if r.recvQty > state.SupplygoETH {
				processed[0].recvQty = state.SupplygoETH
				r.recvQty = r.recvQty - state.SupplygoETH
				r.lending = true
				r.comment = "AlgoFi - Withdraw - Lending Income"
				processed = append(processed, r)
				state.SupplygoETH = 0
			} else {
				state.SupplygoETH = state.SupplygoETH - r.recvQty
			}
		default:
			return records, state, fmt.Errorf("invalid ApplAlgoFiMarket() Withdraw record | onCompletion: %s | action: %s | records length: %d | txns length: %d\n", onCompletion, action, len(records), len(txns))
		}
		return processed, state, nil
	}

	// Borrow
	if action == "b" {
		r := records[0]
		processed = append(processed, records...)
		processed[0].incomeNoTax = true
		processed[0].comment = "AlgoFi - Borrow"
		switch asaUnitName(r.recvASA, assetMap) {
		case "ALGO":
			state.BorrowALGO = state.BorrowALGO + r.recvQty
		case "STBL":
			state.BorrowSTBL = state.BorrowSTBL + r.recvQty
		case "USDC":
			state.BorrowUSDC = state.BorrowUSDC + r.recvQty
		case "goBTC":
			state.BorrowgoBTC = state.BorrowgoBTC + r.recvQty
		case "goETH":
			state.BorrowgoETH = state.BorrowgoETH + r.recvQty
		default:
			return records, state, fmt.Errorf("invalid ApplAlgoFiMarket() Borrow record | onCompletion: %s | action: %s | records length: %d | txns length: %d\n", onCompletion, action, len(records), len(txns))
		}
		return processed, state, nil
	}

	// Repay
	if action == "rb" {
		r := records[0]
		processed = append(processed, records...)
		processed[0].expenseNoTax = true
		processed[0].comment = "AlgoFi - Repay"
		switch asaUnitName(r.sentASA, assetMap) {
		case "ALGO":
			if (r.sentQty - r.fee) > state.BorrowALGO {
				processed[0].sentQty = state.BorrowALGO + r.fee
				r.sentQty = (r.sentQty - r.fee) - state.BorrowALGO
				r.fee = 0
				r.borrow = true
				r.comment = "AlgoFi - Repay - Borrowing Fee"
				processed = append(processed, r)
				state.BorrowALGO = 0
			} else {
				state.BorrowALGO = state.BorrowALGO - (r.sentQty - r.fee)
			}
		case "STBL":
			if r.sentQty > state.BorrowSTBL {
				processed[0].sentQty = state.BorrowSTBL
				r.sentQty = r.sentQty - state.BorrowSTBL
				r.borrow = true
				r.comment = "AlgoFi - Repay - Borrowing Fee"
				processed = append(processed, r)
				state.BorrowSTBL = 0
			} else {
				state.BorrowSTBL = state.BorrowSTBL - r.sentQty
			}
		case "USDC":
			if r.sentQty > state.BorrowUSDC {
				processed[0].sentQty = state.BorrowUSDC
				r.sentQty = r.sentQty - state.BorrowUSDC
				r.borrow = true
				r.comment = "AlgoFi - Repay - Borrowing Fee"
				processed = append(processed, r)
				state.BorrowUSDC = 0
			} else {
				state.BorrowUSDC = state.BorrowUSDC - r.sentQty
			}
		case "goBTC":
			if r.sentQty > state.BorrowgoBTC {
				processed[0].sentQty = state.BorrowgoBTC
				r.sentQty = r.sentQty - state.BorrowgoBTC
				r.borrow = true
				r.comment = "AlgoFi - Repay - Borrowing Fee"
				processed = append(processed, r)
				state.BorrowgoBTC = 0
			} else {
				state.BorrowgoBTC = state.BorrowgoBTC - r.sentQty
			}
		case "goETH":
			if r.sentQty > state.BorrowgoETH {
				processed[0].sentQty = state.BorrowgoETH
				r.sentQty = r.sentQty - state.BorrowgoETH
				r.borrow = true
				r.comment = "AlgoFi - Repay - Borrowing Fee"
				processed = append(processed, r)
				state.BorrowgoETH = 0
			} else {
				state.BorrowgoETH = state.BorrowgoETH - r.sentQty
			}
		default:
			return records, state, fmt.Errorf("invalid ApplAlgoFiMarket() Repay record | onCompletion: %s | action: %s | records length: %d | txns length: %d\n", onCompletion, action, len(records), len(txns))
		}
		return processed, state, nil
	}

	return records, state, fmt.Errorf("invalid ApplAlgoFiMarket() record | onCompletion: %s | action: %s | records length: %d | txns length: %d\n", onCompletion, action, len(records), len(txns))
}

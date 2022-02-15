package exporter

import (
	"fmt"
	"time"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/shopspring/decimal"
)

func (r ExportRecord) IsAlgomint() bool {
	switch {
	case r.IsAssetIDDeposit(386192725) || r.IsAssetIDWithdrawal(386192725):	// goBTC
	case r.IsAssetIDDeposit(386195940) || r.IsAssetIDWithdrawal(386195940): // goETH
	default:
		return false
	}

	if r.sender == r.receiver {
		return false
	}

	// Check if sender or receiver is a AlgoMint.
	if r.sender == "ETGSQKACKC56JWGMDAEP5S2JVQWRKTQUVKCZTMPNUGZLDVCWPY63LSI3H4" ||
		r.receiver == "ETGSQKACKC56JWGMDAEP5S2JVQWRKTQUVKCZTMPNUGZLDVCWPY63LSI3H4" {
		return true
	}
	return false
}

// Algomint
// https://algomint.io/
func DAppAlgomint(records []ExportRecord, assetMap map[uint64]models.Asset) ([]ExportRecord, error) {	
	// Algomint transactions are done in 1 deposit/withdrawal transaction.
	if !IsLengthExcludeReward(records, 1) || !records[0].IsAlgomint() {
		return records, fmt.Errorf("invalid DAppAlgomint() record")
	}
	var processed []ExportRecord

	r := records[0]
	
	/*
		https://algomint-1.gitbook.io/algomint/business-model#fees
		Fees
		Users will pay two types of fees:
		1. When minting tokens:
		0.2% minting fee charged by Algomint. 0% Until 12.01am AEST the 1st of March 2022.
		Plus a mining fee to cover sweeping costs.
		ETH Transactions 0.0025 ETH
		BTC Transactions 0.0001 BTC

		2. When burning tokens:
		0.2% burning fee charged by Algomint.
		Plus a mining fee to cover delivery costs.
		ETH Transactions 0.0025 ETH
		BTC Transactions 0.0001 BTC
		Where the Algomint charged fees go:
		100% of the fees go the Algomint DAO treasury.
	*/

	// Mint goBTC
	if r.IsAssetIDDeposit(386192725) && IsLengthExcludeReward(records, 1) {
		processed = append(processed, records...)
		processed[0].comment = "Algomint - Mint goBTC"
		processed[0].sentCustomQty = assetIDFmt(processed[0].recvQty, processed[0].recvASA, assetMap)
		processed[0].sentCustomCurrency = "BTC"

		btcDepositRecord := records[0]
		btcDepositRecord.recvQty = 0
		btcDepositRecord.recvASA = 0
		btcDepositRecord.sentQty = 0
		btcDepositRecord.sentASA = 0
		btcDepositRecord.topTxID = "btc-deposit-" + records[0].txid
		btcDepositRecord.recvCustomQty = assetIDFmt(records[0].recvQty, records[0].recvASA, assetMap)
		btcDepositRecord.recvCustomCurrency = "BTC"
		btcDepositRecord.comment = "Algomint - Mint goBTC - BTC deposit"
		processed = append(processed, btcDepositRecord)

		// 0.0001 BTC Mining Fee.
		miningRecord := records[0]
		miningRecord.recvQty = 0
		miningRecord.recvASA = 0
		miningRecord.sentQty = 0
		miningRecord.sentASA = 0
		miningRecord.otherFee = true
		miningRecord.topTxID = "mining-fee-" + miningRecord.txid
		miningRecord.sentCustomQty = "0.0001"
		miningRecord.sentCustomCurrency = "BTC"
		miningRecord.feeCustom = "0.0001"
		miningRecord.feeCustomCurrency = "BTC"
		miningRecord.comment = "Algomint - Mint goBTC - mining fee"
		processed = append(processed, miningRecord)
		
		// Add corresponding deposit for the BTC Mining Fee.
		miningDepositRecord := records[0]
		miningDepositRecord.recvQty = 0
		miningDepositRecord.recvASA = 0
		miningDepositRecord.sentQty = 0
		miningDepositRecord.sentASA = 0
		miningDepositRecord.topTxID = "mining-fee-deposit-" + miningDepositRecord.txid
		miningDepositRecord.recvCustomQty = "0.0001"
		miningDepositRecord.recvCustomCurrency = "BTC"
		miningDepositRecord.comment = "Algomint - Mint goBTC - mining fee deposit"
		processed = append(processed, miningDepositRecord)

		// 0.2% minting fee charged by Algomint. 0% Until 12.01am AEST the 1st of March 2022.
		// 12:01 AM Sunday, Australian Eastern Standard Time (AEST) is 2:01 PM Saturday, Coordinated Universal Time (UTC).
		if processed[0].blockTime.After(time.Date(2022, 3, 1, 14, 0, 1, 0, time.UTC)) {
			recvQty, err := decimal.NewFromString(assetIDFmt(processed[0].recvQty, processed[0].recvASA, assetMap))
			if err != nil {
				return processed, err
			}
			// preMintingQty * 0.998% = recvQty 
			// mintingFee = preMintingQty - recvQty
			preMintingQty := recvQty.Div(decimal.RequireFromString("0.998")).RoundFloor(8)	// BTC has 8 decimal places.
			mintingFee := preMintingQty.Sub(recvQty)

			// 0.2% minting fee.
			mintingRecord := records[0]
			mintingRecord.recvQty = 0
			mintingRecord.recvASA = 0
			mintingRecord.sentQty = 0
			mintingRecord.sentASA = 0
			mintingRecord.otherFee = true
			mintingRecord.topTxID = "minting-fee-" + mintingRecord.txid
			mintingRecord.sentCustomQty = mintingFee.String()
			mintingRecord.sentCustomCurrency = "BTC"
			mintingRecord.feeCustom = mintingFee.String()
			mintingRecord.feeCustomCurrency = "BTC"
			mintingRecord.comment = "Algomint - Mint goBTC - minting fee"
			processed = append(processed, mintingRecord)
			
			// Add corresponding deposit for the 0.2% minting fee.
			mintingDepositRecord := records[0]
			mintingDepositRecord.recvQty = 0
			mintingDepositRecord.recvASA = 0
			mintingDepositRecord.sentQty = 0
			mintingDepositRecord.sentASA = 0
			mintingDepositRecord.topTxID = "minting-fee-deposit" + mintingDepositRecord.txid
			mintingDepositRecord.recvCustomQty = mintingFee.String()
			mintingDepositRecord.recvCustomCurrency = "BTC"
			mintingDepositRecord.comment = "Algomint - Mint goBTC - minting fee deposit"
			processed = append(processed, mintingDepositRecord)
		}
		return processed, nil
	}
	
	// Unlock goBTC
	if r.IsAssetIDWithdrawal(386192725) && IsLengthExcludeReward(records, 2) {
		processed = append(processed, records...)
		processed[0].sentQty = processed[0].sentQty - 100000
		processed[0].comment = "Algomint Unlock goBTC"

		// 0.0001 goBTC Mining Fee.
		miningRecord := records[0]
		miningRecord.recvQty = 0
		miningRecord.recvASA = 0
		miningRecord.sentQty = 0
		miningRecord.sentASA = 0
		miningRecord.otherFee = true
		miningRecord.topTxID = "mining-fee-" + miningRecord.txid
		miningRecord.sentQty = 100000
		miningRecord.sentASA = processed[0].sentASA
		miningRecord.feeCustom = assetIDFmt(miningRecord.sentQty, miningRecord.sentASA, assetMap)
		miningRecord.feeCustomCurrency = asaFmt(miningRecord.sentASA, assetMap)
		miningRecord.comment = "Algomint - Unlock goBTC - mining fee"
		processed = append(processed, miningRecord)

		// burningFee = recvQty * 0.2%
		burningFee := records[0].recvQty * 2 / 1000

		// 0.2% burning fee.
		burningRecord := records[0]
		burningRecord.recvQty = 0
		burningRecord.recvASA = 0
		burningRecord.sentQty = 0
		burningRecord.sentASA = 0
		burningRecord.otherFee = true
		burningRecord.topTxID = "minting-fee-" + burningRecord.txid
		burningRecord.sentQty = burningFee
		burningRecord.sentASA = processed[0].sentASA
		burningRecord.feeCustom = assetIDFmt(burningRecord.sentQty, burningRecord.sentASA, assetMap)
		burningRecord.feeCustomCurrency = asaFmt(burningRecord.sentASA, assetMap)
		burningRecord.comment = "Algomint - Unlock goBTC - burning fee"
		processed = append(processed, burningRecord)

		return processed, nil
	}
	
	// Mint goETH
	if r.IsAssetIDDeposit(386195940) && IsLengthExcludeReward(records, 1) {
		processed = append(processed, records...)
		processed[0].comment = "Algomint - Mint goETH"
		processed[0].sentCustomQty = assetIDFmt(processed[0].recvQty, processed[0].recvASA, assetMap)
		processed[0].sentCustomCurrency = "ETH"

		ethDepositRecord := records[0]
		ethDepositRecord.recvQty = 0
		ethDepositRecord.recvASA = 0
		ethDepositRecord.sentQty = 0
		ethDepositRecord.sentASA = 0
		ethDepositRecord.topTxID = "eth-deposit-" + records[0].txid
		ethDepositRecord.recvCustomQty = assetIDFmt(records[0].recvQty, records[0].recvASA, assetMap)
		ethDepositRecord.recvCustomCurrency = "ETH"
		ethDepositRecord.comment = "Algomint - Mint goETH - ETH deposit"
		processed = append(processed, ethDepositRecord)

		// 0.0001 ETH Mining Fee.
		miningRecord := records[0]
		miningRecord.recvQty = 0
		miningRecord.recvASA = 0
		miningRecord.sentQty = 0
		miningRecord.sentASA = 0
		miningRecord.otherFee = true
		miningRecord.topTxID = "mining-fee-" + miningRecord.txid
		miningRecord.sentCustomQty = "0.0025"
		miningRecord.sentCustomCurrency = "ETH"
		miningRecord.feeCustom = "0.0025"
		miningRecord.feeCustomCurrency = "ETH"
		miningRecord.comment = "Algomint - Mint goETH - mining fee"
		processed = append(processed, miningRecord)
		
		// Add corresponding deposit for the goETH Mining Fee.
		miningDepositRecord := records[0]
		miningDepositRecord.recvQty = 0
		miningDepositRecord.recvASA = 0
		miningDepositRecord.sentQty = 0
		miningDepositRecord.sentASA = 0
		miningDepositRecord.topTxID = "mining-fee-deposit-" + miningDepositRecord.txid
		miningDepositRecord.recvCustomQty = "0.0025"
		miningDepositRecord.recvCustomCurrency = "goETH"
		miningDepositRecord.comment = "Algomint - Mint goETH - mining fee deposit"
		processed = append(processed, miningDepositRecord)

		// 0.2% minting fee charged by Algomint. 0% Until 12.01am AEST the 1st of March 2022.
		// 12:01 AM Sunday, Australian Eastern Standard Time (AEST) is 2:01 PM Saturday, Coordinated Universal Time (UTC).
		if processed[0].blockTime.After(time.Date(2022, 3, 1, 14, 0, 1, 0, time.UTC)) {
			recvQty, err := decimal.NewFromString(assetIDFmt(processed[0].recvQty, processed[0].recvASA, assetMap))
			if err != nil {
				return processed, err
			}
			// preMintingQty * 0.998% = recvQty 
			// mintingFee = preMintingQty - recvQty
			preMintingQty := recvQty.Div(decimal.RequireFromString("0.998")).RoundFloor(8)	// goETH has 8 decimal places.
			mintingFee := preMintingQty.Sub(recvQty)

			// 0.2% minting fee.
			mintingRecord := records[0]
			mintingRecord.recvQty = 0
			mintingRecord.recvASA = 0
			mintingRecord.sentQty = 0
			mintingRecord.sentASA = 0
			mintingRecord.otherFee = true
			mintingRecord.topTxID = "minting-fee-" + mintingRecord.txid
			mintingRecord.sentCustomQty = mintingFee.String()
			mintingRecord.sentCustomCurrency = "ETH"
			mintingRecord.feeCustom = mintingFee.String()
			mintingRecord.feeCustomCurrency = "ETH"
			mintingRecord.comment = "Algomint - Mint goETH - minting fee"
			processed = append(processed, mintingRecord)
			
			// Add corresponding deposit for the 0.2% minting fee.
			mintingDepositRecord := records[0]
			mintingDepositRecord.recvQty = 0
			mintingDepositRecord.recvASA = 0
			mintingDepositRecord.sentQty = 0
			mintingDepositRecord.sentASA = 0
			mintingDepositRecord.topTxID = "minting-fee-deposit" + mintingDepositRecord.txid
			mintingDepositRecord.recvCustomQty = mintingFee.String()
			mintingDepositRecord.recvCustomCurrency = "ETH"
			mintingDepositRecord.comment = "Algomint - Mint goETH - minting fee deposit"
			processed = append(processed, mintingDepositRecord)
		}
		return processed, nil
	}
	
	// Unlock goETH
	if r.IsAssetIDWithdrawal(386195940) && IsLengthExcludeReward(records, 2) {
		processed = append(processed, records...)
		processed[0].sentQty = processed[0].sentQty - 2500000
		processed[0].comment = "Algomint - Unlock goETH"

		// 0.0025 goETH Mining Fee.
		miningRecord := records[0]
		miningRecord.recvQty = 0
		miningRecord.recvASA = 0
		miningRecord.sentQty = 0
		miningRecord.sentASA = 0
		miningRecord.otherFee = true
		miningRecord.topTxID = "mining-fee-" + miningRecord.txid
		miningRecord.sentQty = 2500000
		miningRecord.sentASA = processed[0].sentASA
		miningRecord.feeCustom = assetIDFmt(miningRecord.sentQty, miningRecord.sentASA, assetMap)
		miningRecord.feeCustomCurrency = asaFmt(miningRecord.sentASA, assetMap)
		miningRecord.comment = "Algomint - Unlock goETH - mining fee"
		processed = append(processed, miningRecord)

		// burningFee = recvQty * 0.2%
		burningFee := records[0].recvQty * 2 / 1000

		// 0.2% burning fee.
		burningRecord := records[0]
		burningRecord.recvQty = 0
		burningRecord.recvASA = 0
		burningRecord.sentQty = 0
		burningRecord.sentASA = 0
		burningRecord.otherFee = true
		burningRecord.topTxID = "minting-fee-" + burningRecord.txid
		burningRecord.sentQty = burningFee
		burningRecord.sentASA = processed[0].sentASA
		burningRecord.feeCustom = assetIDFmt(burningRecord.sentQty, burningRecord.sentASA, assetMap)
		burningRecord.feeCustomCurrency = asaFmt(burningRecord.sentASA, assetMap)
		burningRecord.comment = "Algomint - Unlock goETH - burning fee"
		processed = append(processed, burningRecord)

		return processed, nil
	}

	return records, fmt.Errorf("invalid DAppAlgomint() record | records length: %d", len(records))
}

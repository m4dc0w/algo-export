package exporter

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/types"

	"github.com/shopspring/decimal"
)

type ExportFactory func() Interface

var formats = map[string]ExportFactory{}

// Verified ASA that export well to tax tracker websites (i.e. cointracking.info).
// The string value allows for token disambiguation to the tax software.
// Token disambiguation is needed when there are multiple coins with the same name.
// https://cointracking.info/coin_charts.php
var verifiedASA = map[uint64]string{
	163650: "",         // ARCC
	226701642: "",      // YLDY
	230946361: "GEMS3", // Algogems
	27165954: "",       // PLANETS
	283820866: "",      // XET
	287867876: "",      // OPUL
}

func registerFormat(format string, factory ExportFactory) {
	formats[format] = factory
}

func Formats() []string {
	var formatNams []string
	for format := range formats {
		formatNams = append(formatNams, format)
	}
	return formatNams
}

func GetFormatter(format string) Interface {
	if factory, found := formats[format]; found {
		return factory()
	}
	return nil
}

// ExportRecord will contain entries for all sends from a specific account, or receives to the account.
// Sends to separate accounts in a single transaction
type ExportRecord struct {
	blockTime time.Time
	topTxID   string
	txid      string
	recvQty   uint64
	recvASA   uint64
	receiver  string
	sentQty   uint64
	sentASA   uint64
	sender    string
	fee       uint64
	label     string
	comment   string

	airdrop   bool  // Is this an airdrop - treat as income.
	appl      bool  // Is this an application.
	mining    bool  // Is this a mining.
	reward    bool  // Is this a reward transaction - treat as income.
	spend     bool  // Is this a spend transaction.
	trade     bool  // Is this a trade transaction.
	feeTx     bool  // Is this a fee transaction.
	txRaw     models.Transaction
	account   string
}

// appendPostFilter is a simple post-processing filter that ignores records that are all 0
// as well as adjusting s
func appendPostFilter(records []ExportRecord, record ExportRecord) []ExportRecord {
	if record.recvQty == 0 && record.sentQty == 0 && record.fee == 0 {
		// Nothing sent, nothing received, nothing in fees... ignore !
		return records
	}
	// The only time we have send and receive at same time is when sending to ourselves.
	// Will typically be equivalent to just a send of the fee
	if record.sentQty != 0 && record.recvQty != 0 {
		record.sentQty = record.sentQty - record.recvQty
		record.recvQty = 0
	}
	return append(records, record)
}

// Interface defines the generic CSV 'exporter' interface which CSV-export implementations must implement.
type Interface interface {
	Name() string
	WriteHeader(writer io.Writer)
	WriteRecord(writer io.Writer, assetMap map[uint64]models.Asset, record ExportRecord)
}

func algoFmt(algos uint64) string {
	return fmt.Sprintf("%.6f", types.MicroAlgos(algos).ToAlgos())
}

func assetIDFmt(amount, assetID uint64, assetMap map[uint64]models.Asset) string {
	if assetID == 0 {
		return algoFmt(amount)
	}
	if val, ok := assetMap[assetID]; ok {
		if val.Params.Decimals != 0 {
			// models.Params.Decimals must be between 0 and 19 (inclusive).
			tokens := decimal.RequireFromString(strconv.FormatUint(amount, 10))
			return tokens.Shift(int32(val.Params.Decimals) * -1).StringFixed(int32(val.Params.Decimals))
		}
		return strconv.FormatUint(amount, 10)
	}
	log.Fatalln("unknown decimal amount for AssetID:", assetID)
	return strconv.FormatUint(amount, 10)
}

func asaFmt(assetID uint64, assetMap map[uint64]models.Asset) string {
	if assetID == 0 {
		return "ALGO"
	}
	val, ok := assetMap[assetID]
	if !ok {
		log.Fatalln("unknown unit name for AssetID:", assetID)
		return ""
	}
	if asaName, ok := verifiedASA[assetID]; ok {
		if asaName != "" {
			return asaName
		}
		return val.Params.UnitName
	}
	return fmt.Sprintf("%x", assetID % 4294967295)  // Limit to 8 characters.
}

func asaUnitName(assetID uint64, assetMap map[uint64]models.Asset) string {
	if assetID == 0 {
		return "ALGO"
	}
	val, ok := assetMap[assetID]
	if !ok {
		log.Fatalln("unknown unit name for AssetID:", assetID)
		return ""
	}
	return val.Params.UnitName
}

func asaComment(assetID uint64, assetMap map[uint64]models.Asset) string {
	if assetID == 0 {
		return ""
	}
	if _, ok := verifiedASA[assetID]; ok {
		return ""
	}
	val, ok := assetMap[assetID]
	if !ok {
		log.Fatalln("unknown unit name for AssetID:", assetID)
		return ""
	}
	return fmt.Sprintf("%s-%d | %s", val.Params.UnitName, assetID, val.Params.Name)
}

func (r ExportRecord) IsASADeposit() bool {
	return r.recvASA != 0 && r.IsDeposit()
}

func (r ExportRecord) IsASAMining(assetID uint64) bool {
	return r.recvASA == assetID && r.IsASADeposit()
}

func (r ExportRecord) IsDeposit() bool {
	return r.recvQty != 0 && r.sentQty == 0
}

func (r ExportRecord) IsReward() bool {
	return r.reward
}

func (r ExportRecord) IsTrade() bool {
	return r.recvQty != 0 && r.sentQty != 0
}

func (r ExportRecord) IsWithdrawal() bool {
	return r.recvQty == 0 && r.sentQty != 0
}

func (r ExportRecord) String() string {
	return fmt.Sprintf("| TopTxID: %s | txID: %s | Group: %s | recv: %d %d | sent: %d %d | sender: %s | receiver: %s | comment: %s", r.topTxID, r.txid, base64.StdEncoding.EncodeToString(r.txRaw.Group), r.recvQty, r.recvASA, r.sentQty, r.sentASA, r.sender, r.receiver, r.comment)
}

func IsLengthExcludeReward(records []ExportRecord, length int) bool {
	if length < 0 {
		return false
	}
	return length == 0 && len(records) == length || (len(records) == length && !records[length-1].IsReward()) || (len(records) == (length+1) && records[length].IsReward())
}

func ExtractApplication(txns []models.Transaction) (models.TransactionApplication, error) {
	for _, tx := range txns {
		if tx.Type == "appl" {
			return tx.ApplicationTransaction, nil
		}
	}
	return models.TransactionApplication{}, fmt.Errorf("transaction application not found")
}

// Parse a transaction block, converting into simple send / receive equivalents.
// Sending from the account being scanned, or receiving (sometimes both in one tx)
// Tracking apps seem to treat 'fees' a little differently and seem to assume they're specifically for trades.
// Since this code is focused on on-chain send/receive activity, the fees are better expressed as 'total send' amount
// send amount + tx fee, vs receive amount.  The tracking sites will then express that as a chain fee.
func FilterTransaction(tx models.Transaction, topTxID, account string, assetMap map[uint64]models.Asset) []ExportRecord {
	var (
		blockTime  = time.Unix(int64(tx.RoundTime), 0).UTC()
		recvAmount uint64
		sendAmount uint64
		rewards    uint64
		records    []ExportRecord
	)

	switch tx.Type {
	case "pay":
		if tx.PaymentTransaction.Receiver == account || tx.PaymentTransaction.CloseRemainderTo == account {
			// We could potentially be receiver, AND close-to account so check independently
			// We could be sender as well - so handle appropriately.
			if tx.PaymentTransaction.Receiver == account {
				recvAmount += tx.PaymentTransaction.Amount
				rewards += tx.ReceiverRewards
			}
			if tx.PaymentTransaction.CloseRemainderTo == account {
				recvAmount += tx.PaymentTransaction.CloseAmount + tx.ClosingAmount
				rewards += tx.CloseRewards
			}
			// ...we could've sent to ourselves!
			if tx.Sender == account {
				sendAmount = tx.PaymentTransaction.Amount + tx.Fee
				rewards += tx.SenderRewards
			}
			// Ignore transaction fee if we are only the receiver.
			if tx.Sender != account && tx.PaymentTransaction.Receiver == account {
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime,
					topTxID:   topTxID,
					txid:      tx.Id,
					recvQty:   recvAmount,
					receiver:  account,
					sentQty:   sendAmount,
					sender:    tx.Sender,
					txRaw:     tx,
					account:   account,
				})
			} else {
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime,
					topTxID:   topTxID,
					txid:      tx.Id,
					recvQty:   recvAmount,
					receiver:  account,
					sentQty:   sendAmount,
					sender:    tx.Sender,
					fee:       tx.Fee,
					txRaw:     tx,
					account:   account,
				})
			}
		} else {
			// only choice at this point are sending transactions
			rewards = tx.SenderRewards

			// handle case where we close-to an account and it's not same as receiver so treat as if two sends for export purposes
			// so receives can be matched in different accounts if user has both
			if tx.PaymentTransaction.CloseRemainderTo != "" && tx.PaymentTransaction.Receiver != tx.PaymentTransaction.CloseRemainderTo {
				// First, add transaction for close-to... (without fee)
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime,
					topTxID:   topTxID,
					txid:      tx.Id,
					receiver:  tx.PaymentTransaction.CloseRemainderTo,
					sentQty:   tx.PaymentTransaction.CloseAmount + tx.ClosingAmount,
					sender:    account,
					txRaw:     tx,
					account:   account,
				})
				// then add an extra transaction 1-sec later to base receiver (with fee)
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime.Add(1 * time.Second),
					topTxID:   topTxID,
					txid:      tx.Id,
					receiver:  tx.PaymentTransaction.Receiver,
					sentQty:   tx.PaymentTransaction.Amount + tx.Fee,
					sender:    account,
					fee:       tx.Fee,
					txRaw:     tx,
					account:   account,
				})
			} else {
				// either a regular receive or a receive and close-remainder-to but to same account.
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime,
					topTxID:   topTxID,
					txid:      tx.Id,
					receiver:  tx.PaymentTransaction.Receiver,
					sentQty:   tx.PaymentTransaction.Amount + tx.PaymentTransaction.CloseAmount + tx.ClosingAmount + tx.Fee,
					sender:    account,
					fee:       tx.Fee,
					txRaw:     tx,
					account:   account,
				})
			}
		}
	case "axfer":
		if tx.AssetTransferTransaction.Receiver == account || tx.AssetTransferTransaction.CloseTo == account {
			// We could potentially be receiver, AND close-to account so check independently
			// We could be sender as well - so handle appropriately.
			if tx.AssetTransferTransaction.Receiver == account {
				recvAmount += tx.AssetTransferTransaction.Amount
				rewards += tx.ReceiverRewards
			}
			if tx.AssetTransferTransaction.CloseTo == account {
				recvAmount += tx.AssetTransferTransaction.CloseAmount + tx.AssetTransferTransaction.CloseAmount
				rewards += tx.CloseRewards
			}
			// ...we could've sent to ourselves!
			if tx.Sender == account {
				sendAmount = tx.AssetTransferTransaction.Amount
				rewards += tx.SenderRewards
			}

			records = appendPostFilter(records, ExportRecord{
				blockTime: blockTime,
				topTxID:   topTxID,
				txid:      tx.Id,
				recvQty:   recvAmount,
			  recvASA:   tx.AssetTransferTransaction.AssetId,
				receiver:  account,
				sentQty:   sendAmount,
			  sentASA:   tx.AssetTransferTransaction.AssetId,
				sender:    tx.Sender,
				txRaw:     tx,
				account:   account,
			})
		} else {
			// only choice at this point are sending transactions
			rewards = tx.SenderRewards

			// handle case where we close-to an account and it's not same as receiver so treat as if two sends for export purposes
			// so receives can be matched in different accounts if user has both
			if tx.AssetTransferTransaction.CloseTo != "" && tx.AssetTransferTransaction.Receiver != tx.AssetTransferTransaction.CloseTo {
				// First, add transaction for close-to... (without fee)
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime,
					topTxID:   topTxID,
					txid:      tx.Id,
					receiver:  tx.AssetTransferTransaction.CloseTo,
					sentQty:   tx.AssetTransferTransaction.CloseAmount + tx.AssetTransferTransaction.CloseAmount,
			    sentASA:   tx.AssetTransferTransaction.AssetId,
					sender:    account,
					txRaw:     tx,
					account:   account,
				})
				// then add an extra transaction 1-sec later to base receiver.
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime.Add(1 * time.Second),
					topTxID:   topTxID,
					txid:      tx.Id,
					receiver:  tx.AssetTransferTransaction.Receiver,
					sentQty:   tx.AssetTransferTransaction.Amount,
			    sentASA:   tx.AssetTransferTransaction.AssetId,
					sender:    account,
					txRaw:     tx,
					account:   account,
				})
			} else {
				// either a regular receive or a receive and close-remainder-to but to same account.
				records = appendPostFilter(records, ExportRecord{
					blockTime: blockTime,
					topTxID:   topTxID,
					txid:      tx.Id,
					receiver:  tx.AssetTransferTransaction.Receiver,
					sentQty:   tx.AssetTransferTransaction.Amount + tx.AssetTransferTransaction.CloseAmount,
			    sentASA:   tx.AssetTransferTransaction.AssetId,
					sender:    account,
					txRaw:     tx,
					account:   account,
				})
			}
		}
		// Split out fees into separate record because fee currency is different than ASA currency.
		if tx.Sender == account {
			records = appendPostFilter(records, ExportRecord{
				blockTime: blockTime,
				topTxID:   topTxID,
				txid:      tx.Id,
				sentQty:   tx.Fee,
				fee:       tx.Fee,
				sender:    account,
				feeTx:     true,
				txRaw:     tx,
				account:   account,
			})
		}
	case "keyreg", "acfg", "afrz", "appl":
		// Just track the fees and rewards for now as a result of the transaction
		// Ignore the ASA activity that is not an asset transfer transaction.
		if tx.AssetTransferTransaction.Receiver == account {
			rewards += tx.ReceiverRewards
		}
		if tx.Sender == account {
			records = appendPostFilter(records, ExportRecord{
				blockTime: blockTime,
				topTxID:   topTxID,
				txid:      tx.Id,
				sentQty:   tx.Fee,
				fee:       tx.Fee,
				sender:    account,
				feeTx:     true,
				txRaw:     tx,
				account:   account,
			})
			rewards = tx.SenderRewards
		}
	default:
		log.Fatalln("unknown transaction type:", tx.Type)
	}

	// now handle rewards (effectively us receiving them - either we sent and received pending rewards
	// or received a payment and also were assigned the pending rewards.  Treat both as a standalone receive.
	// The transaction is exported with a timestamp 1 second before the real on-chain transaction
	// so the extra balance is there for deductions and we don't go negative.  The transaction is defined as a
	// rewards so it can be tracked as income by the tax tracker.
	if rewards != 0 {
		// Apply rewards 'first' (earlier timestamp)
		records = appendPostFilter(records, ExportRecord{
			blockTime: blockTime.Add(-1 * time.Second),
			topTxID:   topTxID,
			txid:      tx.Id,
			reward:    true,
			recvQty:   rewards,
			receiver:  account,
			txRaw:     tx,
			account:   account,
		})
	}
	return records
}

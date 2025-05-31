package broadcast

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// TransactionInfo contains transaction details from broadcast files
type TransactionInfo struct {
	Hash         string `json:"hash"`
	BlockNumber  uint64 `json:"blockNumber"`
	From         string `json:"from"`
	To           string `json:"to"`
	Value        string `json:"value"`
	Data         string `json:"data"`
	ContractName string `json:"contractName,omitempty"`
	ContractAddr string `json:"contractAddress,omitempty"`
}

// BundleTransactionInfo groups transaction info with bundle context
type BundleTransactionInfo struct {
	BundleID     common.Hash
	Transactions []TransactionInfo
	IsSafe       bool // True if this is a Safe multisig transaction
}



// MatchBundleTransactions matches broadcast transactions to bundle IDs
func MatchBundleTransactions(bundleID common.Hash, txInfos []TransactionInfo, senderAddr common.Address) *BundleTransactionInfo {
	bundleInfo := &BundleTransactionInfo{
		BundleID:     bundleID,
		Transactions: []TransactionInfo{},
		IsSafe:       false,
	}
	
	// Match transactions from the same sender
	for _, tx := range txInfos {
		if strings.EqualFold(tx.From, senderAddr.Hex()) {
			bundleInfo.Transactions = append(bundleInfo.Transactions, tx)
		}
	}
	
	// Check if this is a Safe transaction (one tx for all deployments)
	if len(bundleInfo.Transactions) == 1 && isMultiSendData(bundleInfo.Transactions[0].Data) {
		bundleInfo.IsSafe = true
	}
	
	return bundleInfo
}

// isMultiSendData checks if transaction data is a MultiSend call
func isMultiSendData(data string) bool {
	// MultiSend selector: 0x8d80ff0a
	return strings.HasPrefix(strings.ToLower(data), "0x8d80ff0a")
}
package script

import (
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// extractScriptNameFromBroadcastPath extracts the script name from a broadcast file path
// e.g., "broadcast/DeployWithTreb.s.sol/31337/run-latest.json" -> "DeployWithTreb.s.sol"
func extractScriptNameFromBroadcastPath(broadcastPath string) string {
	// Split the path into components
	parts := strings.Split(filepath.ToSlash(broadcastPath), "/")
	
	// Look for the "broadcast" directory and get the next component
	for i, part := range parts {
		if part == "broadcast" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	
	return ""
}

// convertBroadcastFileToTransactionInfos converts a BroadcastFile to TransactionInfo slice
func convertBroadcastFileToTransactionInfos(broadcastFile *broadcast.BroadcastFile) []broadcast.TransactionInfo {
	var txInfos []broadcast.TransactionInfo
	
	for i, tx := range broadcastFile.Transactions {
		// Get block number from corresponding receipt
		var blockNumber uint64
		contractAddr := tx.ContractAddress
		
		if i < len(broadcastFile.Receipts) {
			receipt := broadcastFile.Receipts[i]
			// Parse hex block number
			if strings.HasPrefix(receipt.BlockNumber, "0x") {
				if parsed, err := strconv.ParseUint(receipt.BlockNumber[2:], 16, 64); err == nil {
					blockNumber = parsed
				}
			}
			// Use receipt's contract address if available
			if receipt.ContractAddress != "" && receipt.ContractAddress != "0x0000000000000000000000000000000000000000" {
				contractAddr = receipt.ContractAddress
			}
		}
		
		txInfos = append(txInfos, broadcast.TransactionInfo{
			Hash:         tx.Hash,
			BlockNumber:  blockNumber,
			From:         tx.Transaction.From,
			To:           tx.Transaction.To,
			Value:        tx.Transaction.Value,
			Data:         tx.Transaction.Data,
			ContractName: tx.ContractName,
			ContractAddr: contractAddr,
		})
		
		// Also add entries for additional contracts (CreateX deployments)
		for _, additional := range tx.AdditionalContracts {
			// Only include CREATE and CREATE2 contracts, not CREATE3 proxy contracts
			if additional.TransactionType == "CREATE" || additional.TransactionType == "CREATE2" {
				txInfos = append(txInfos, broadcast.TransactionInfo{
					Hash:         tx.Hash,
					BlockNumber:  blockNumber,
					From:         tx.Transaction.From,
					To:           tx.Transaction.To,
					Value:        tx.Transaction.Value,
					Data:         tx.Transaction.Data,
					ContractName: tx.ContractName,
					ContractAddr: additional.Address,
				})
			}
		}
	}
	
	return txInfos
}

// ConvertRawLogToTypesLog converts our internal RawLog format (from forge JSON output) to go-ethereum types.Log
func ConvertRawLogToTypesLog(rawLog RawLog) (*types.Log, error) {
	// Decode the hex data string to bytes
	data, err := hex.DecodeString(strings.TrimPrefix(rawLog.Data, "0x"))
	if err != nil {
		return nil, err
	}

	return &types.Log{
		Address: rawLog.Address,
		Topics:  rawLog.Topics,
		Data:    data,
		// These fields would be filled from context if available:
		BlockNumber: 0,
		TxHash:      common.Hash{},
		TxIndex:     0,
		BlockHash:   common.Hash{},
		Index:       0,
		Removed:     false,
	}, nil
}

// ConvertEventsLogToTypesLog converts our events.Log format to go-ethereum types.Log
func ConvertEventsLogToTypesLog(log events.Log) (*types.Log, error) {
	// Decode the hex data string to bytes
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, err
	}

	return &types.Log{
		Address: log.Address,
		Topics:  log.Topics,
		Data:    data,
		// These fields would be filled from context if available:
		BlockNumber: 0,
		TxHash:      common.Hash{},
		TxIndex:     0,
		BlockHash:   common.Hash{},
		Index:       0,
		Removed:     false,
	}, nil
}
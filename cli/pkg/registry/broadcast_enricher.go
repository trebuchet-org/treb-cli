package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
)

// BroadcastEnricher enriches registry updates with broadcast data
type BroadcastEnricher struct{}

// parseGasUsed parses gas used from string
func parseGasUsed(gasStr string) uint64 {
	if strings.HasPrefix(gasStr, "0x") {
		gas, _ := strconv.ParseUint(gasStr[2:], 16, 64)
		return gas
	}
	gas, _ := strconv.ParseUint(gasStr, 10, 64)
	return gas
}

// NewBroadcastEnricher creates a new broadcast enricher
func NewBroadcastEnricher() *BroadcastEnricher {
	return &BroadcastEnricher{}
}

// EnrichFromBroadcastFile enriches a registry update with data from a broadcast file
func (be *BroadcastEnricher) EnrichFromBroadcastFile(update *RegistryUpdate, broadcastPath string) error {
	if broadcastPath == "" {
		return nil
	}

	// Read broadcast file
	data, err := os.ReadFile(broadcastPath)
	if err != nil {
		return fmt.Errorf("failed to read broadcast file: %w", err)
	}

	var broadcastData broadcast.BroadcastFile
	if err := json.Unmarshal(data, &broadcastData); err != nil {
		return fmt.Errorf("failed to parse broadcast file: %w", err)
	}

	// Build a map of deployed addresses to internal tx IDs
	addressToInternalID := make(map[common.Address]string)
	for internalID, depUpdate := range update.Deployments {
		addr := common.HexToAddress(depUpdate.Deployment.Address)
		addressToInternalID[addr] = internalID
	}

	// Process broadcast transactions
	for i, tx := range broadcastData.Transactions {
		if i >= len(broadcastData.Receipts) {
			continue
		}
		receipt := broadcastData.Receipts[i]

		// Parse block number
		var blockNumber uint64
		if strings.HasPrefix(receipt.BlockNumber, "0x") {
			blockNumber, _ = strconv.ParseUint(receipt.BlockNumber[2:], 16, 64)
		} else {
			blockNumber, _ = strconv.ParseUint(receipt.BlockNumber, 10, 64)
		}

		// Check if this transaction deployed any of our contracts
		var matchedInternalIDs []string

		// Check main contract deployment
		if tx.ContractAddress != "" && tx.ContractAddress != "0x0000000000000000000000000000000000000000" {
			addr := common.HexToAddress(tx.ContractAddress)
			if internalID, exists := addressToInternalID[addr]; exists {
				matchedInternalIDs = append(matchedInternalIDs, internalID)
			}
		}

		// Check additional contracts (for CreateX deployments)
		for _, additional := range tx.AdditionalContracts {
			if additional.TransactionType == "CREATE" || additional.TransactionType == "CREATE2" {
				addr := common.HexToAddress(additional.Address)
				if internalID, exists := addressToInternalID[addr]; exists {
					matchedInternalIDs = append(matchedInternalIDs, internalID)
				}
			}
		}

		// If we found matches, enrich them
		if len(matchedInternalIDs) > 0 {
			enrichment := &BroadcastEnrichment{
				TransactionHash: tx.Hash,
				BlockNumber:     blockNumber,
				GasUsed:         parseGasUsed(receipt.GasUsed),
				Timestamp:       uint64(broadcastData.Timestamp), // Use broadcast timestamp
			}

			// Enrich all matched deployments
			// (multiple deployments can be in the same transaction)
			for _, internalID := range matchedInternalIDs {
				if err := update.EnrichFromBroadcast(internalID, enrichment); err != nil {
					return fmt.Errorf("failed to enrich deployment %s: %w", internalID, err)
				}
			}
		}
	}

	// Update metadata
	update.Metadata.BroadcastPath = broadcastPath

	return nil
}

// EnrichFromBroadcastParser uses the existing broadcast parser to enrich
func (be *BroadcastEnricher) EnrichFromBroadcastParser(update *RegistryUpdate, scriptName string, chainID uint64) error {
	// Extract script name and find broadcast file
	parser := broadcast.NewParser(".")
	_, err := parser.ParseLatestBroadcast(scriptName, chainID)
	if err != nil {
		return fmt.Errorf("failed to parse broadcast: %w", err)
	}

	// Get the broadcast data
	broadcastPath := filepath.Join("broadcast", scriptName, fmt.Sprintf("%d", chainID), "run-latest.json")
	
	// Use the file enricher with the actual path
	actualPath := filepath.Join(".", broadcastPath)
	return be.EnrichFromBroadcastFile(update, actualPath)
}
package broadcast

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Parser handles parsing of Foundry broadcast files
type Parser struct {
	projectRoot string
}

type BroadcastFile domain.BroadcastFile

// NewParser creates a new broadcast file parser
func NewParser(projectRoot string) *Parser {
	return &Parser{
		projectRoot: projectRoot,
	}
}

// ParseBroadcastFile parses a broadcast file
func (p *Parser) ParseBroadcastFile(file string) (*BroadcastFile, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read broadcast file: %w", err)
	}

	var broadcast BroadcastFile
	if err := json.Unmarshal(data, &broadcast); err != nil {
		return nil, fmt.Errorf("failed to parse broadcast file: %w", err)
	}

	return &broadcast, nil
}

// ParseLatestBroadcast parses the latest broadcast file for a given script and chain
func (p *Parser) ParseLatestBroadcast(scriptName string, chainID uint64) (*BroadcastFile, error) {
	broadcastPath := p.getBroadcastPath(scriptName, chainID)
	latestFile := filepath.Join(broadcastPath, "run-latest.json")

	if _, err := os.Stat(latestFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("broadcast file not found: %s", latestFile)
	}

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read broadcast file: %w", err)
	}

	var broadcast BroadcastFile
	if err := json.Unmarshal(data, &broadcast); err != nil {
		return nil, fmt.Errorf("failed to parse broadcast file: %w", err)
	}

	return &broadcast, nil
}

// getBroadcastPath returns the path to broadcast files for a script and chain
func (p *Parser) getBroadcastPath(scriptName string, chainID uint64) string {
	return filepath.Join(p.projectRoot, "broadcast", scriptName, fmt.Sprintf("%d", chainID))
}

// GetAllBroadcastFiles returns all broadcast files for a given script
func (p *Parser) GetAllBroadcastFiles(scriptName string, chainID uint64) ([]string, error) {
	broadcastPath := p.getBroadcastPath(scriptName, chainID)

	if _, err := os.Stat(broadcastPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("broadcast directory not found: %s", broadcastPath)
	}

	files, err := filepath.Glob(filepath.Join(broadcastPath, "run-*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list broadcast files: %w", err)
	}

	return files, nil
}

// GetTransactionHashForAddress returns the transaction hash that created the given contract address
func (b *BroadcastFile) GetTransactionHashForAddress(address common.Address) (common.Hash, uint64, error) {
	addressStr := strings.ToLower(address.Hex())

	for _, tx := range b.Transactions {
		// Check direct contract creation
		if strings.ToLower(tx.ContractAddress) == addressStr {
			// Find corresponding receipt for block number
			for _, receipt := range b.Receipts {
				if receipt.TransactionHash == tx.Hash {
					blockNumber, err := parseHexToUint64(receipt.BlockNumber)
					if err != nil {
						return common.Hash{}, 0, fmt.Errorf("failed to parse block number: %w", err)
					}
					return common.HexToHash(tx.Hash), blockNumber, nil
				}
			}
			return common.HexToHash(tx.Hash), 0, nil
		}

		// Check additional contracts (CreateX deployments)
		for _, additionalContract := range tx.AdditionalContracts {
			if strings.ToLower(additionalContract.ContractAddress) == addressStr {
				// Find corresponding receipt for block number
				for _, receipt := range b.Receipts {
					if receipt.TransactionHash == tx.Hash {
						blockNumber, err := parseHexToUint64(receipt.BlockNumber)
						if err != nil {
							return common.Hash{}, 0, fmt.Errorf("failed to parse block number: %w", err)
						}
						return common.HexToHash(tx.Hash), blockNumber, nil
					}
				}
				return common.HexToHash(tx.Hash), 0, nil
			}
		}
	}

	// For CreateX deployments through treb-sol, Foundry may not properly track the deployed contract
	// in additionalContracts. As a fallback, if there's exactly one transaction that looks like a CreateX call,
	// we assume it created the contract we're looking for.
	if len(b.Transactions) == 1 {
		tx := b.Transactions[0]
		if tx.Function == "deployCreate3(bytes32,bytes)" || tx.Function == "deployCreate2(bytes32,bytes)" {
			for _, receipt := range b.Receipts {
				if receipt.TransactionHash == tx.Hash {
					blockNumber, err := parseHexToUint64(receipt.BlockNumber)
					if err != nil {
						return common.Hash{}, 0, fmt.Errorf("failed to parse block number: %w", err)
					}
					return common.HexToHash(tx.Hash), blockNumber, nil
				}
			}
			return common.HexToHash(tx.Hash), 0, nil
		}
	}

	return common.Hash{}, 0, fmt.Errorf("transaction not found for address %s - this suggests CreateX deployment tracking issue", address.Hex())
}

// parseHexToUint64 parses a hex string to uint64
func parseHexToUint64(hexStr string) (uint64, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return strconv.ParseUint(hexStr, 16, 64)
}

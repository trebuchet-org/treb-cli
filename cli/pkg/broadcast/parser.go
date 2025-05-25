package broadcast

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// BroadcastFile represents the structure of a Foundry broadcast file
type BroadcastFile struct {
	Transactions []Transaction `json:"transactions"`
	Receipts     []Receipt     `json:"receipts"`
	Libraries    []string      `json:"libraries"`
	Pending      []string      `json:"pending"`
	Returns      interface{}   `json:"returns"`
	Timestamp    int64         `json:"timestamp"`
	Chain        int64         `json:"chain"`
	Multi        bool          `json:"multi"`
	Commit       string        `json:"commit"`
}

type Transaction struct {
	Hash                string               `json:"hash"`
	TransactionType     string               `json:"transactionType"`
	ContractName        string               `json:"contractName"`
	ContractAddress     string               `json:"contractAddress"`
	Function            string               `json:"function"`
	Arguments           interface{}          `json:"arguments"`
	Transaction         TxData               `json:"transaction"`
	AdditionalContracts []AdditionalContract `json:"additionalContracts"`
	IsFixedGasLimit     bool                 `json:"isFixedGasLimit"`
}

type TxData struct {
	Type                 string        `json:"type"`
	From                 string        `json:"from"`
	To                   string        `json:"to"`
	Gas                  string        `json:"gas"`
	Value                string        `json:"value"`
	Data                 string        `json:"data"`
	Nonce                string        `json:"nonce"`
	AccessList           []interface{} `json:"accessList"`
	ChainId              string        `json:"chainId"`
	MaxFeePerGas         string        `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string        `json:"maxPriorityFeePerGas"`
}

type Receipt struct {
	TransactionHash   string `json:"transactionHash"`
	TransactionIndex  string `json:"transactionIndex"`
	BlockHash         string `json:"blockHash"`
	BlockNumber       string `json:"blockNumber"`
	From              string `json:"from"`
	To                string `json:"to"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	ContractAddress   string `json:"contractAddress"`
	Logs              []Log  `json:"logs"`
	LogsBloom         string `json:"logsBloom"`
	Status            string `json:"status"`
	EffectiveGasPrice string `json:"effectiveGasPrice"`
	Type              string `json:"type"`
}

type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

type AdditionalContract struct {
	TransactionType string `json:"transactionType"`
	Address         string `json:"address"`
	InitCode        string `json:"initCode"`
}

// Parser handles parsing of Foundry broadcast files
type Parser struct {
	projectRoot string
}

// NewParser creates a new broadcast file parser
func NewParser(projectRoot string) *Parser {
	return &Parser{
		projectRoot: projectRoot,
	}
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
			if strings.ToLower(additionalContract.Address) == addressStr {
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
	
	return common.Hash{}, 0, fmt.Errorf("transaction not found for address %s", address.Hex())
}

// parseHexToUint64 parses a hex string to uint64
func parseHexToUint64(hexStr string) (uint64, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return strconv.ParseUint(hexStr, 16, 64)
}

package broadcast

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bogdan/fdeploy/cli/pkg/types"
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
	Hash               string      `json:"hash"`
	TransactionType    string      `json:"transactionType"`
	ContractName       string      `json:"contractName"`
	ContractAddress    string      `json:"contractAddress"`
	Function           string      `json:"function"`
	Arguments          interface{} `json:"arguments"`
	Transaction        TxData      `json:"transaction"`
	AdditionalContracts []AdditionalContract `json:"additionalContracts"`
	IsFixedGasLimit    bool        `json:"isFixedGasLimit"`
}

type TxData struct {
	Type                 string `json:"type"`
	From                 string `json:"from"`
	To                   string `json:"to"`
	Gas                  string `json:"gas"`
	Value                string `json:"value"`
	Data                 string `json:"data"`
	Nonce                string `json:"nonce"`
	AccessList           []interface{} `json:"accessList"`
	ChainId              string `json:"chainId"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
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
func (p *Parser) ParseLatestBroadcast(scriptName string, chainID uint64) (*types.DeploymentResult, error) {
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

	return p.extractDeploymentResult(&broadcast, latestFile)
}

// ParsePredictionOutput parses the output from PredictAddress.s.sol script
func (p *Parser) ParsePredictionOutput(output []byte) (*types.PredictResult, error) {
	lines := strings.Split(string(output), "\n")
	
	result := &types.PredictResult{}
	inPredictionSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "=== PREDICTION_RESULT ===" {
			inPredictionSection = true
			continue
		}
		
		if line == "=== END_PREDICTION ===" {
			inPredictionSection = false
			break
		}
		
		if !inPredictionSection {
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "ADDRESS":
			result.Address = common.HexToAddress(value)
		case "SALT":
			if saltBytes := common.FromHex(value); len(saltBytes) == 32 {
				copy(result.Salt[:], saltBytes)
			}
		case "INIT_CODE_HASH":
			if hashBytes := common.FromHex(value); len(hashBytes) == 32 {
				copy(result.InitCodeHash[:], hashBytes)
			}
		}
	}
	
	if result.Address == (common.Address{}) {
		return nil, fmt.Errorf("failed to parse prediction output: address not found")
	}
	
	return result, nil
}

// ParseDeploymentOutput parses the structured output from CreateXDeployment script
func (p *Parser) ParseDeploymentOutput(output []byte) (*types.DeploymentResult, error) {
	lines := strings.Split(string(output), "\n")
	
	result := &types.DeploymentResult{}
	inDeploymentSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "=== DEPLOYMENT_RESULT ===" {
			inDeploymentSection = true
			continue
		}
		
		if line == "=== END_DEPLOYMENT ===" {
			inDeploymentSection = false
			break
		}
		
		if !inDeploymentSection {
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "ADDRESS":
			result.Address = common.HexToAddress(value)
		case "SALT":
			if saltBytes := common.FromHex(value); len(saltBytes) == 32 {
				copy(result.Salt[:], saltBytes)
			}
		case "INIT_CODE_HASH":
			if hashBytes := common.FromHex(value); len(hashBytes) == 32 {
				copy(result.InitCodeHash[:], hashBytes)
			}
		case "BLOCK_NUMBER":
			if blockNum, err := strconv.ParseUint(value, 10, 64); err == nil {
				result.BlockNumber = blockNum
			}
		case "TX_HASH":
			result.TxHash = common.HexToHash(value)
		case "DEPLOYMENT_TYPE":
			result.DeploymentType = strings.ToLower(value)
		case "DEPLOYMENT_LABEL":
			result.Label = value
		case "DEPLOYMENT_ENV":
			result.Env = value
		case "SAFE_TX_HASH":
			result.SafeTxHash = common.HexToHash(value)
		}
	}
	
	if result.Address == (common.Address{}) {
		return nil, fmt.Errorf("failed to parse deployment output: address not found")
	}
	
	return result, nil
}

// extractDeploymentResult extracts deployment information from broadcast file
func (p *Parser) extractDeploymentResult(broadcast *BroadcastFile, filePath string) (*types.DeploymentResult, error) {
	// Find the main deployment transaction
	var deployTx *Transaction
	var deployReceipt *Receipt
	var deployedContract *AdditionalContract
	
	for _, tx := range broadcast.Transactions {
		if tx.TransactionType == "CREATE" || tx.TransactionType == "CREATE2" {
			deployTx = &tx
			break
		}
		// Handle CreateX deployments (CALL to CreateX with additionalContracts)
		if tx.TransactionType == "CALL" && len(tx.AdditionalContracts) > 0 {
			// For CREATE3 deployments - prioritize the final CREATE contract (actual contract)
			for _, contract := range tx.AdditionalContracts {
				if contract.TransactionType == "CREATE" && len(contract.InitCode) > 100 {
					// This is the actual contract, not a small proxy
					deployTx = &tx
					deployedContract = &contract
					break
				}
			}
			// For CREATE2 deployments - if no CREATE found
			if deployTx == nil {
				for _, contract := range tx.AdditionalContracts {
					if contract.TransactionType == "CREATE2" {
						deployTx = &tx
						deployedContract = &contract
						break
					}
				}
			}
			if deployTx != nil {
				break
			}
		}
	}
	
	if deployTx == nil {
		return nil, fmt.Errorf("no deployment transaction found in broadcast file")
	}
	
	// Find corresponding receipt
	for _, receipt := range broadcast.Receipts {
		if receipt.TransactionHash == deployTx.Hash {
			deployReceipt = &receipt
			break
		}
	}
	
	if deployReceipt == nil {
		return nil, fmt.Errorf("no receipt found for deployment transaction")
	}
	
	// Parse block number
	blockNumber, err := p.parseHexToUint64(deployReceipt.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block number: %w", err)
	}
	
	// Determine contract address based on deployment type
	var contractAddress string
	if deployedContract != nil {
		// CreateX deployment - use address from additionalContracts
		contractAddress = deployedContract.Address
	} else {
		// Direct CREATE/CREATE2 deployment
		contractAddress = deployTx.ContractAddress
	}

	result := &types.DeploymentResult{
		Address:       common.HexToAddress(contractAddress),
		TxHash:        common.HexToHash(deployTx.Hash),
		BlockNumber:   blockNumber,
		BroadcastFile: filePath,
	}
	
	// Try to extract salt if it's a CREATE2 deployment
	if deployTx.TransactionType == "CREATE2" {
		// TODO: Extract salt from transaction data
		// This would require parsing the CREATE2 call data
	}
	
	return result, nil
}

// getBroadcastPath returns the path to broadcast files for a script and chain
func (p *Parser) getBroadcastPath(scriptName string, chainID uint64) string {
	return filepath.Join(p.projectRoot, "broadcast", scriptName, fmt.Sprintf("%d", chainID))
}

// parseHexToUint64 parses a hex string to uint64
func (p *Parser) parseHexToUint64(hexStr string) (uint64, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	
	return strconv.ParseUint(hexStr, 16, 64)
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
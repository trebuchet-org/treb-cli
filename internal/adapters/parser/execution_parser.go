package parser

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/trebuchet-org/treb-cli/internal/adapters/abi"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ExecutionParser is our new clean implementation of the execution parser
type ExecutionParser struct {
	projectRoot  string
	eventDecoder *abi.EventDecoder
}

// parseHexUint64 parses a hex string to uint64
func parseHexUint64(hexStr string) (uint64, bool) {
	if hexStr == "" {
		return 0, false
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	val, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

// NewExecutionParser creates a new internal parser
func NewExecutionParser(projectRoot string) (*ExecutionParser, error) {
	eventDecoder, err := abi.NewEventDecoder()
	if err != nil {
		return nil, fmt.Errorf("failed to create event decoder: %w", err)
	}

	return &ExecutionParser{
		projectRoot:  projectRoot,
		eventDecoder: eventDecoder,
	}, nil
}

// ParseExecution parses the script output into a structured execution result
func (p *ExecutionParser) ParseExecution(
	ctx context.Context,
	output *usecase.ScriptExecutionOutput,
) (*domain.ScriptExecution, error) {
	execution := &domain.ScriptExecution{
		Success: output.Success,
		Logs:    p.extractLogs(output.RawOutput),
	}

	// If we have JSON output, extract events from it
	if output.JSONOutput != nil {
		if err := p.parseJSONOutput(output.JSONOutput, execution); err != nil {
			// Log error but continue - we can still get data from broadcast
			fmt.Printf("Warning: Failed to parse JSON output: %v\n", err)
		}
	}

	// If we have a broadcast path, parse the broadcast file
	if output.BroadcastPath != "" {
		broadcast, err := p.parseBroadcastFile(output.BroadcastPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse broadcast file: %w", err)
		}

		// Extract transactions
		execution.Transactions = p.extractTransactions(broadcast)

		// Extract deployments
		execution.Deployments = p.extractDeployments(broadcast, execution.Transactions, execution)

		// Calculate total gas used
		for _, tx := range execution.Transactions {
			if tx.GasUsed != nil {
				execution.GasUsed += *tx.GasUsed
			}
		}

		execution.BroadcastPath = output.BroadcastPath
	}

	// Set error state if execution failed
	if !output.Success {
		execution.Success = false
		// Try to extract error from raw output
		if len(output.RawOutput) > 0 {
			execution.Error = string(output.RawOutput)
		}
	}

	return execution, nil
}

// parseBroadcastFile reads and parses a Foundry broadcast JSON file
func (p *ExecutionParser) parseBroadcastFile(path string) (*domain.BroadcastFile, error) {
	// Check if the path is already absolute
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(p.projectRoot, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read broadcast file: %w", err)
	}

	var broadcast domain.BroadcastFile
	if err := json.Unmarshal(data, &broadcast); err != nil {
		return nil, fmt.Errorf("failed to parse broadcast JSON: %w", err)
	}

	return &broadcast, nil
}

// extractTransactions converts broadcast transactions to domain transactions
func (p *ExecutionParser) extractTransactions(broadcast *domain.BroadcastFile) []domain.ScriptTransaction {
	var transactions []domain.ScriptTransaction

	// Map to store transactions by hash for receipt matching
	txByHash := make(map[string]*domain.ScriptTransaction)

	// Process transactions
	for i, btx := range broadcast.Transactions {
		// Generate transaction ID from hash
		var txID [32]byte
		if btx.Hash != "" {
			// Use first 32 bytes of hash for ID
			hashBytes := []byte(btx.Hash)
			copy(txID[:], hashBytes)
		}

		tx := domain.ScriptTransaction{
			ID:            fmt.Sprintf("tx-%d", i),
			TransactionID: txID,
			Hash:          btx.Hash,
			Status:        domain.TransactionStatusSimulated, // Default status
		}

		// Extract transaction data
		if txData, ok := btx.Transaction["from"].(string); ok {
			tx.From = txData
			tx.Sender = txData // Also set Sender as alias
		}
		if txData, ok := btx.Transaction["to"].(string); ok {
			tx.To = txData
		}
		if txData, ok := btx.Transaction["value"].(string); ok {
			tx.Value = txData
		}
		if txData, ok := btx.Transaction["input"].(string); ok {
			tx.Data = []byte(txData)
		} else if txData, ok := btx.Transaction["data"].(string); ok {
			tx.Data = []byte(txData)
		}
		if txData, ok := btx.Transaction["nonce"].(float64); ok {
			tx.Nonce = uint64(txData)
		}
		// Gas limit is not a field in ScriptTransaction, skip it

		// Check for deployments
		if btx.ContractName != "" && btx.ContractAddr != "" {
			tx.DeploymentIDs = append(tx.DeploymentIDs, btx.ContractAddr)
		}

		// Add additional contracts
		for _, additional := range btx.AdditionalContracts {
			if additional.ContractAddr != "" {
				tx.DeploymentIDs = append(tx.DeploymentIDs, additional.ContractAddr)
			}
		}

		transactions = append(transactions, tx)
		txByHash[tx.Hash] = &transactions[i] // Store pointer to the actual element in slice
	}

	// Update with receipt data
	for _, receipt := range broadcast.Receipts {
		if tx, ok := txByHash[receipt.TransactionHash]; ok {
			// Update status
			if receipt.Status == "0x1" || receipt.Status == "1" {
				tx.Status = domain.TransactionStatusExecuted
			} else {
				tx.Status = domain.TransactionStatusFailed
			}

			// Update gas used
			if gasUsed, ok := parseHexUint64(receipt.GasUsed); ok {
				tx.GasUsed = &gasUsed
			}

			// Update block number
			if blockNum, ok := parseHexUint64(receipt.BlockNumber); ok {
				tx.BlockNumber = &blockNum
			}

			// Also set TxHash to match v1 behavior
			tx.TxHash = &receipt.TransactionHash
		}
	}

	return transactions
}

// extractDeployments extracts deployment information from broadcast data
func (p *ExecutionParser) extractDeployments(broadcast *domain.BroadcastFile, transactions []domain.ScriptTransaction, execution *domain.ScriptExecution) []domain.ScriptDeployment {
	var deployments []domain.ScriptDeployment
	deploymentMap := make(map[string]*domain.ScriptDeployment)

	// Create transaction lookup map
	txByHash := make(map[string]*domain.ScriptTransaction)
	for i := range transactions {
		txByHash[transactions[i].Hash] = &transactions[i]
	}

	// First check if we have parsed events from JSON output
	if execution != nil && len(execution.Events) > 0 {
		// Debug
		if os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: extractDeployments - processing %d events from JSON\n", len(execution.Events))
		}
		// Process parsed events
		for _, event := range execution.Events {
			switch event.EventType {
			case domain.EventContractDeployed:
				deployment := &domain.ScriptDeployment{
					Address:        event.Address,
					ContractName:   event.ContractName,
					Label:          event.Label,
					Deployer:       event.Deployer,
					DeploymentType: domain.SingletonDeployment,
					Salt:           event.Salt,
				}

				// Use the transaction ID from the event
				deployment.TransactionID = event.TransactionID

				// Use address as key to handle duplicates
				deploymentMap[event.Address] = deployment

			case domain.EventProxyDeployed:
				deployment := &domain.ScriptDeployment{
					Address:        event.Address,
					ContractName:   event.ContractName,
					Label:          event.Label,
					Deployer:       event.Deployer,
					DeploymentType: domain.ProxyDeployment,
					IsProxy:        true,
					ProxyInfo: &domain.ProxyInfo{
						Implementation: event.Implementation,
						Type:           "TransparentUpgradeableProxy",
					},
				}

				// Use the transaction ID from the event
				deployment.TransactionID = event.TransactionID

				deploymentMap[event.Address] = deployment

			case domain.EventCreateXContractCreation:
				// CreateX events don't contain contract names, so we need to match with broadcast data
				if deployment, exists := deploymentMap[event.Address]; exists {
					deployment.Salt = event.Salt
				}
			}
		}
	} else {
		// Fall back to parsing events from broadcast receipts
		for _, receipt := range broadcast.Receipts {
			tx, hasTx := txByHash[receipt.TransactionHash]

			for _, log := range receipt.Logs {
				// Convert to types.Log for the decoder
				ethLog, err := p.convertBroadcastLog(log)
				if err != nil {
					continue
				}

				// Try to decode the event
				event, err := p.eventDecoder.DecodeLog(*ethLog)
				if err != nil {
					continue
				}

				// Process deployment events
				switch event.EventType {
				case domain.EventContractDeployed:
					deployment := &domain.ScriptDeployment{
						Address:        event.Address,
						ContractName:   event.ContractName,
						Label:          event.Label,
						Deployer:       event.Deployer,
						DeploymentType: domain.SingletonDeployment,
					}
					if hasTx {
						deployment.TransactionID = tx.TransactionID
					}
					// Use address as key to handle duplicates
					deploymentMap[event.Address] = deployment

				case domain.EventProxyDeployed:
					deployment := &domain.ScriptDeployment{
						Address:        event.Address,
						ContractName:   event.ContractName,
						Label:          event.Label,
						Deployer:       event.Deployer,
						DeploymentType: domain.ProxyDeployment,
						IsProxy:        true,
						ProxyInfo: &domain.ProxyInfo{
							Implementation: event.Implementation,
							Type:           "TransparentUpgradeableProxy",
						},
					}
					if hasTx {
						deployment.TransactionID = tx.TransactionID
					}
					deploymentMap[event.Address] = deployment

				case domain.EventCreateXContractCreation:
					// CreateX events don't contain contract names, so we need to match with broadcast data
					if deployment, exists := deploymentMap[event.Address]; exists {
						deployment.Salt = event.Salt
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, deployment := range deploymentMap {
		deployments = append(deployments, *deployment)
	}

	return deployments
}

// convertBroadcastLog converts a broadcast log to an Ethereum types.Log
func (p *ExecutionParser) convertBroadcastLog(log domain.BroadcastLog) (*types.Log, error) {
	ethLog := &types.Log{
		Address: common.HexToAddress(log.Address),
	}

	// Convert topics
	for _, topic := range log.Topics {
		ethLog.Topics = append(ethLog.Topics, common.HexToHash(topic))
	}

	// Convert data
	if log.Data != "" {
		data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode log data: %w", err)
		}
		ethLog.Data = data
	}

	return ethLog, nil
}

// extractLogs extracts console.log outputs from raw script output
func (p *ExecutionParser) extractLogs(output []byte) []string {
	var logs []string

	// Look for common log patterns
	patterns := []string{
		`console\.log\s*\([^)]+\)`,
		`Logs:.*`,
		`\[LOG\].*`,
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check if line matches any log pattern
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, trimmed); matched {
				logs = append(logs, trimmed)
				break
			}
		}

		// Also capture lines that look like structured output
		if strings.HasPrefix(trimmed, "==") || strings.HasPrefix(trimmed, "##") {
			logs = append(logs, trimmed)
		}
	}

	return logs
}

// parseJSONOutput parses the JSON output from forge script --json
func (p *ExecutionParser) parseJSONOutput(jsonOutput interface{}, execution *domain.ScriptExecution) error {
	// Import the ForgeJSONOutput type
	type ForgeEventLog struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
		Data    string   `json:"data"`
	}

	type ForgeJSONOutput struct {
		Success bool            `json:"success"`
		RawLogs []ForgeEventLog `json:"raw_logs"`
	}

	// Type assert to our expected structure
	forgeOutput, ok := jsonOutput.(*ForgeJSONOutput)
	if !ok {
		// Try to convert from generic interface
		jsonBytes, err := json.Marshal(jsonOutput)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON output: %w", err)
		}

		var output ForgeJSONOutput
		if err := json.Unmarshal(jsonBytes, &output); err != nil {
			return fmt.Errorf("failed to unmarshal JSON output: %w", err)
		}
		forgeOutput = &output
	}

	// Debug logging
	if os.Getenv("TREB_TEST_DEBUG") != "" {
		fmt.Printf("DEBUG: parseJSONOutput - found %d raw logs\n", len(forgeOutput.RawLogs))
	}

	// Store events for later processing
	execution.EventLogs = []interface{}{}

	// Convert forge logs to ethereum logs and decode events
	for _, forgeLog := range forgeOutput.RawLogs {
		ethLog, err := p.convertForgeLog(forgeLog)
		if err != nil {
			if os.Getenv("TREB_TEST_DEBUG") != "" {
				fmt.Printf("DEBUG: Failed to convert forge log: %v\n", err)
			}
			continue
		}

		execution.EventLogs = append(execution.EventLogs, *ethLog)

		// Try to decode the event
		event, err := p.eventDecoder.DecodeLog(*ethLog)
		if err != nil {
			if os.Getenv("TREB_TEST_DEBUG") != "" && !strings.Contains(err.Error(), "unknown") {
				fmt.Printf("DEBUG: Failed to decode event: %v\n", err)
			}
			continue
		}

		// Store decoded events
		if execution.Events == nil {
			execution.Events = []domain.DeploymentEvent{}
		}
		execution.Events = append(execution.Events, *event)

		if os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: Decoded event type: %s, address: %s\n", event.EventType, event.Address)
		}
	}

	return nil
}

// convertForgeLog converts a forge log to an Ethereum types.Log
func (p *ExecutionParser) convertForgeLog(forgeLog any) (*types.Log, error) {
	// Handle the ForgeEventLog type
	type ForgeEventLog struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
		Data    string   `json:"data"`
	}

	// Convert from interface if needed
	var log ForgeEventLog
	if fLog, ok := forgeLog.(ForgeEventLog); ok {
		log = fLog
	} else {
		// Try to convert through JSON
		jsonBytes, err := json.Marshal(forgeLog)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(jsonBytes, &log); err != nil {
			return nil, err
		}
	}

	ethLog := &types.Log{
		Address: common.HexToAddress(log.Address),
	}

	// Convert topics
	for _, topic := range log.Topics {
		ethLog.Topics = append(ethLog.Topics, common.HexToHash(topic))
	}

	// Convert data
	if log.Data != "" {
		data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode log data: %w", err)
		}
		ethLog.Data = data
	}

	return ethLog, nil
}

func (p *ExecutionParser) EnrichFromBroadcast(ctx context.Context, execution *domain.ScriptExecution, broadcastPath string) error {
	return nil
}

var _ usecase.ExecutionParser = (&ExecutionParser{})

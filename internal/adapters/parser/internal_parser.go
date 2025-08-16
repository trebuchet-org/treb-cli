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

// InternalParser is our new clean implementation of the execution parser
type InternalParser struct {
	projectRoot  string
	eventDecoder *abi.EventDecoder
}

// parseHexUint64 parses a hex string to uint64
func parseHexUint64(hexStr string) (uint64, bool) {
	if hexStr == "" {
		return 0, false
	}
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}
	val, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

// NewInternalParser creates a new internal parser
func NewInternalParser(projectRoot string) (*InternalParser, error) {
	eventDecoder, err := abi.NewEventDecoder()
	if err != nil {
		return nil, fmt.Errorf("failed to create event decoder: %w", err)
	}
	
	return &InternalParser{
		projectRoot:  projectRoot,
		eventDecoder: eventDecoder,
	}, nil
}

// ParseExecution parses the script output into a structured execution result
func (p *InternalParser) ParseExecution(
	ctx context.Context,
	output *usecase.ScriptExecutionOutput,
	network string,
	chainID uint64,
) (*domain.ScriptExecution, error) {
	execution := &domain.ScriptExecution{
		Network: network,
		ChainID: chainID,
		Success: output.Success,
		Logs:    p.extractLogs(output.RawOutput),
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
		execution.Deployments = p.extractDeployments(broadcast, execution.Transactions)
		
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
func (p *InternalParser) parseBroadcastFile(path string) (*domain.BroadcastFile, error) {
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
func (p *InternalParser) extractTransactions(broadcast *domain.BroadcastFile) []domain.ScriptTransaction {
	var transactions []domain.ScriptTransaction

	// Map to store transactions by hash for receipt matching
	txByHash := make(map[string]*domain.ScriptTransaction)

	// Process transactions
	for i, btx := range broadcast.Transactions {
		tx := domain.ScriptTransaction{
			ID:     fmt.Sprintf("tx-%d", i),
			Hash:   btx.Hash,
			Status: domain.TransactionStatusSimulated, // Default status
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
		txByHash[tx.Hash] = &tx
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
		}
	}

	return transactions
}

// extractDeployments extracts deployment information from broadcast data
func (p *InternalParser) extractDeployments(broadcast *domain.BroadcastFile, transactions []domain.ScriptTransaction) []domain.ScriptDeployment {
	var deployments []domain.ScriptDeployment
	deploymentMap := make(map[string]*domain.ScriptDeployment)

	// Create transaction lookup map
	txByHash := make(map[string]*domain.ScriptTransaction)
	for i := range transactions {
		txByHash[transactions[i].Hash] = &transactions[i]
	}

	// Parse events from receipts
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

	// Fall back to broadcast transaction data if no events found
	if len(deploymentMap) == 0 {
		for i, btx := range broadcast.Transactions {
			if btx.ContractName != "" && btx.ContractAddr != "" {
				deployment := domain.ScriptDeployment{
					Address:        btx.ContractAddr,
					ContractName:   btx.ContractName,
					DeploymentType: domain.SingletonDeployment,
				}

				// Extract deployer from transaction
				if i < len(transactions) {
					deployment.Deployer = transactions[i].From
					deployment.TransactionID = transactions[i].TransactionID
				}

				// Determine deployment type
				if strings.Contains(strings.ToLower(btx.ContractName), "proxy") {
					deployment.DeploymentType = domain.ProxyDeployment
					deployment.IsProxy = true
				}

				deployments = append(deployments, deployment)
			}

			// Process additional contracts
			for _, additional := range btx.AdditionalContracts {
				if additional.ContractName != "" && additional.ContractAddr != "" {
					deployment := domain.ScriptDeployment{
						Address:        additional.ContractAddr,
						ContractName:   additional.ContractName,
						DeploymentType: domain.SingletonDeployment,
					}

					if i < len(transactions) {
						deployment.Deployer = transactions[i].From
						deployment.TransactionID = transactions[i].TransactionID
					}

					deployments = append(deployments, deployment)
				}
			}
		}
	} else {
		// Convert map to slice
		for _, deployment := range deploymentMap {
			deployments = append(deployments, *deployment)
		}
	}

	return deployments
}

// convertBroadcastLog converts a broadcast log to an Ethereum types.Log
func (p *InternalParser) convertBroadcastLog(log domain.BroadcastLog) (*types.Log, error) {
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
func (p *InternalParser) extractLogs(output []byte) []string {
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

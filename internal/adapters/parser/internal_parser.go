package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// InternalParser is our new clean implementation of the execution parser
type InternalParser struct {
	projectRoot string
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
func NewInternalParser(projectRoot string) *InternalParser {
	return &InternalParser{
		projectRoot: projectRoot,
	}
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
func (p *InternalParser) parseBroadcastFile(relativePath string) (*domain.BroadcastFile, error) {
	fullPath := filepath.Join(p.projectRoot, relativePath)
	
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

	// Process main contract deployments
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
			}

			// Determine deployment type
			if strings.Contains(strings.ToLower(btx.ContractName), "proxy") {
				deployment.DeploymentType = domain.ProxyDeployment
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
				}

				deployments = append(deployments, deployment)
			}
		}
	}

	return deployments
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

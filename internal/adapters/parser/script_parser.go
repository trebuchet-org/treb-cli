package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Parser handles parsing of script execution outputs
type Parser struct {
	eventParser      EventParser
	transactionParser TransactionParser
	deploymentParser DeploymentParser
}

// EventParser parses events from script output
type EventParser interface {
	ParseEvents(output string) ([]domain.ScriptExecutionEvent, error)
}

// TransactionParser extracts transactions from execution data
type TransactionParser interface {
	ParseTransactions(broadcastFile *domain.BroadcastFile) ([]*domain.ExecutedTransaction, error)
}

// DeploymentParser extracts deployments from execution data
type DeploymentParser interface {
	ParseDeployments(events []domain.ScriptExecutionEvent, transactions []*domain.ExecutedTransaction) ([]*domain.DeploymentResult, error)
}

// NewParser creates a new script parser
func NewParser() *Parser {
	return &Parser{
		eventParser:       &defaultEventParser{},
		transactionParser: &defaultTransactionParser{},
		deploymentParser:  &defaultDeploymentParser{},
	}
}

// ParseScriptOutput parses the complete output from a script execution
func (p *Parser) ParseScriptOutput(output string, broadcastPath string, network string, chainID uint64) (*domain.ScriptExecution, error) {
	execution := &domain.ScriptExecution{
		Network:       network,
		ChainID:       chainID,
		BroadcastPath: broadcastPath,
		Logs:          extractLogs(output),
		Metadata:      make(map[string]string),
	}

	// Parse broadcast file if available
	if broadcastPath != "" {
		broadcastFile, err := p.parseBroadcastFile(broadcastPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse broadcast file: %w", err)
		}

		// Extract transactions
		_, err = p.transactionParser.ParseTransactions(broadcastFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transactions: %w", err)
		}
		// TODO: Convert ExecutedTransaction to ScriptTransaction
		// execution.Transactions = transactions

		// Calculate total gas used
		// for _, tx := range transactions {
		// 	execution.GasUsed += tx.GasUsed
		// }
	}

	// Parse events from output
	if _, err := p.eventParser.ParseEvents(output); err != nil {
		// Non-fatal: we can still have valid execution without events
		execution.Logs = append(execution.Logs, fmt.Sprintf("Warning: failed to parse events: %v", err))
	}

	// TODO: Convert between transaction types
	// Extract deployments from events and transactions
	// deployments, err := p.deploymentParser.ParseDeployments(events, execution.Transactions)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse deployments: %w", err)
	// }
	// execution.Deployments = deployments

	// Determine success based on transactions
	execution.Success = len(execution.Transactions) > 0
	for _, tx := range execution.Transactions {
		if tx.Status == "failed" {
			execution.Success = false
			execution.Error = "One or more transactions failed"
			break
		}
	}

	return execution, nil
}

// parseBroadcastFile reads and parses a Foundry broadcast file
func (p *Parser) parseBroadcastFile(path string) (*domain.BroadcastFile, error) {
	// In a real implementation, this would read the file
	// For now, return a placeholder
	return &domain.BroadcastFile{
		Chain: 31337, // Default to local chain
	}, nil
}

// extractLogs extracts console logs from script output
func extractLogs(output string) []string {
	var logs []string
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		// Look for console.log outputs
		if strings.Contains(line, "console.log") || strings.Contains(line, "Logs:") {
			logs = append(logs, strings.TrimSpace(line))
		}
	}
	
	return logs
}

// defaultEventParser is a simple event parser implementation
type defaultEventParser struct{}

func (p *defaultEventParser) ParseEvents(output string) ([]domain.ScriptExecutionEvent, error) {
	var events []domain.ScriptExecutionEvent
	
	// Look for structured event markers in output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "EVENT:") {
			// Parse structured event data
			eventData := strings.TrimPrefix(line, "EVENT:")
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(eventData), &data); err == nil {
				events = append(events, domain.ScriptExecutionEvent{
					Type: "script_event",
					Data: data,
				})
			}
		}
	}
	
	return events, nil
}

// defaultTransactionParser extracts transactions from broadcast data
type defaultTransactionParser struct{}

func (p *defaultTransactionParser) ParseTransactions(broadcast *domain.BroadcastFile) ([]*domain.ExecutedTransaction, error) {
	var transactions []*domain.ExecutedTransaction
	
	for i, btx := range broadcast.Transactions {
		tx := &domain.ExecutedTransaction{
			ID:     fmt.Sprintf("tx-%d", i),
			Hash:   btx.Hash,
			Status: "success", // Default, will be updated from receipts
		}
		
		// Extract transaction details
		if txData, ok := btx.Transaction["from"].(string); ok {
			tx.From = txData
		}
		if txData, ok := btx.Transaction["to"].(string); ok {
			tx.To = txData
		}
		if txData, ok := btx.Transaction["value"].(string); ok {
			tx.Value = txData
		}
		if txData, ok := btx.Transaction["data"].(string); ok {
			tx.Data = txData
		}
		if txData, ok := btx.Transaction["nonce"].(float64); ok {
			tx.Nonce = uint64(txData)
		}
		
		// Check if this is a deployment
		if btx.ContractName != "" && btx.ContractAddr != "" {
			tx.DeploymentIDs = append(tx.DeploymentIDs, btx.ContractAddr)
		}
		
		transactions = append(transactions, tx)
	}
	
	// Update transaction status from receipts
	for _, receipt := range broadcast.Receipts {
		for _, tx := range transactions {
			if tx.Hash == receipt.TransactionHash {
				if receipt.Status == "0x0" {
					tx.Status = "failed"
				}
				// Parse gas used
				if gasUsed, ok := parseHexUint64(receipt.GasUsed); ok {
					tx.GasUsed = gasUsed
				}
				// Parse block number
				if blockNum, ok := parseHexUint64(receipt.BlockNumber); ok {
					tx.BlockNumber = blockNum
				}
				break
			}
		}
	}
	
	return transactions, nil
}

// defaultDeploymentParser extracts deployments from events and transactions
type defaultDeploymentParser struct{}

func (p *defaultDeploymentParser) ParseDeployments(events []domain.ScriptExecutionEvent, transactions []*domain.ExecutedTransaction) ([]*domain.DeploymentResult, error) {
	var deployments []*domain.DeploymentResult
	
	// Extract deployments from transactions
	for _, tx := range transactions {
		for _, deploymentID := range tx.DeploymentIDs {
			deployment := &domain.DeploymentResult{
				ID:            deploymentID,
				TransactionID: tx.ID,
				Address:       deploymentID,
				Deployer:      tx.From,
			}
			deployments = append(deployments, deployment)
		}
	}
	
	// Enhance with event data
	for _, event := range events {
		if event.Type == "ContractDeployed" {
			// Match and enhance deployment data
			if addr, ok := event.Data["address"].(string); ok {
				for _, dep := range deployments {
					if strings.EqualFold(dep.Address, addr) {
						// Update with event data
						if name, ok := event.Data["contractName"].(string); ok {
							dep.ContractName = name
						}
						if artifact, ok := event.Data["artifact"].(string); ok {
							dep.ArtifactPath = artifact
						}
						break
					}
				}
			}
		}
	}
	
	return deployments, nil
}


// BroadcastParser reads and parses Foundry broadcast files
type BroadcastParser struct {
	projectRoot string
}

// NewBroadcastParser creates a new broadcast file parser
func NewBroadcastParser(projectRoot string) *BroadcastParser {
	return &BroadcastParser{
		projectRoot: projectRoot,
	}
}

// ParseBroadcastFile reads and parses a broadcast file
func (p *BroadcastParser) ParseBroadcastFile(relativePath string) (*domain.BroadcastFile, error) {
	// fullPath := filepath.Join(p.projectRoot, relativePath)
	
	// Read file content
	var content []byte // Would read from file
	
	var broadcast domain.BroadcastFile
	if err := json.Unmarshal(content, &broadcast); err != nil {
		return nil, fmt.Errorf("failed to parse broadcast JSON: %w", err)
	}
	
	return &broadcast, nil
}
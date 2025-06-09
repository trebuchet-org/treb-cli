package parser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/bindings"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Parser handles parsing of forge script results into a unified execution structure
type Parser struct {
	eventParser        *events.EventParser
	transactions       map[[32]byte]*Transaction
	transactionOrder   [][32]byte // Track order of transactions
	proxyRelationships map[common.Address]*ProxyRelationship
	mu                 sync.RWMutex
}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{
		eventParser:        events.NewEventParser(),
		transactions:       make(map[[32]byte]*Transaction),
		transactionOrder:   make([][32]byte, 0),
		proxyRelationships: make(map[common.Address]*ProxyRelationship),
	}
}

// Parse converts a forge.ScriptResult into a ScriptExecution
func (p *Parser) Parse(result *forge.ScriptResult, network string, chainID uint64) (*ScriptExecution, error) {
	// Reset parser state
	p.transactions = make(map[[32]byte]*Transaction)
	p.transactionOrder = make([][32]byte, 0)
	p.proxyRelationships = make(map[common.Address]*ProxyRelationship)

	execution := &ScriptExecution{
		Transactions:       []*Transaction{},
		SafeTransactions:   []*SafeTransaction{},
		Deployments:        []*DeploymentRecord{},
		ProxyRelationships: make(map[common.Address]*ProxyRelationship),
		Events:             []interface{}{},
		Logs:               []string{},
		TextOutput:         string(result.RawOutput),
		ParsedOutput:       result.ParsedOutput,
		Success:            result.Success,
		BroadcastPath:      result.BroadcastPath,
		Network:            network,
		ChainID:            chainID,
	}

	// Parse events if we have parsed output
	if result.ParsedOutput != nil && result.ParsedOutput.ScriptOutput != nil {
		// Parse events
		parsedEvents, err := p.eventParser.ParseEvents(result.ParsedOutput.ScriptOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to parse events: %w", err)
		}
		execution.Events = parsedEvents

		// Process events
		for _, event := range parsedEvents {
			// Process transaction events
			switch e := event.(type) {
			case *bindings.TrebTransactionSimulated:
				p.processTransactionSimulated(e)
			case *bindings.TrebSafeTransactionQueued:
				p.processSafeTransactionQueued(e, execution)
			case *bindings.TrebContractDeployed:
				execution.Deployments = append(execution.Deployments, &DeploymentRecord{
					TransactionID: e.TransactionId,
					Deployment:    &e.Deployment,
					Address:       e.Location,
					Deployer:      e.Deployer,
				})
			}

			// Process proxy events
			p.processProxyEvent(event)
		}

		// Enrich with forge output data
		if result.ParsedOutput != nil {
			// Process trace outputs
			for _, trace := range result.ParsedOutput.TraceOutputs {
				p.processTraceOutput(&trace)
			}

			// Process receipts
			for _, receipt := range result.ParsedOutput.Receipts {
				p.processReceipt(&receipt)
			}

			// Extract console logs
			execution.Logs = result.ParsedOutput.ConsoleLogs
		}

		// Get final transaction list
		execution.Transactions = p.getTransactions()

		// Copy proxy relationships
		for addr, rel := range p.proxyRelationships {
			execution.ProxyRelationships[addr] = rel
		}
	}

	// Enrich with broadcast data if available
	if result.BroadcastPath != "" {
		if err := p.enrichFromBroadcast(execution, result.BroadcastPath); err != nil {
			// Don't fail, just log warning
			fmt.Printf("Warning: Failed to enrich from broadcast: %v\n", err)
		}
	}

	return execution, nil
}

// processTransactionSimulated processes a TransactionSimulated event
func (p *Parser) processTransactionSimulated(event *bindings.TrebTransactionSimulated) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tx := &Transaction{
		SimulatedTransaction: bindings.SimulatedTransaction(event.SimulatedTx),
		Status:               types.TransactionStatusSimulated,
		Deployments:          []DeploymentInfo{},
	}

	p.transactions[event.SimulatedTx.TransactionId] = tx
	p.transactionOrder = append(p.transactionOrder, event.SimulatedTx.TransactionId)
}

// processSafeTransactionQueued processes a SafeTransactionQueued event
func (p *Parser) processSafeTransactionQueued(event *bindings.TrebSafeTransactionQueued, execution *ScriptExecution) {
	// Create Safe transaction record
	safeTx := &SafeTransaction{
		SafeTxHash:     event.SafeTxHash,
		Safe:           event.Safe,
		Proposer:       event.Proposer,
		TransactionIDs: event.TransactionIds,
	}
	execution.SafeTransactions = append(execution.SafeTransactions, safeTx)

	// Update all referenced transactions to QUEUED status
	p.mu.Lock()
	defer p.mu.Unlock()

	for idx, txID := range event.TransactionIds {
		if tx, exists := p.transactions[txID]; exists {
			tx.Status = types.TransactionStatusQueued
			tx.SafeAddress = &event.Safe
			safeTxHash := common.Hash(event.SafeTxHash)
			tx.SafeTxHash = &safeTxHash
			batchIdx := idx
			tx.SafeBatchIdx = &batchIdx
		}
	}
}

// processProxyEvent processes proxy-related events
func (p *Parser) processProxyEvent(event interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch e := event.(type) {
	case *events.ProxyDeployedEvent:
		p.proxyRelationships[e.ProxyAddress] = &ProxyRelationship{
			ProxyAddress:          e.ProxyAddress,
			ImplementationAddress: e.ImplementationAddress,
			ProxyType:             ProxyTypeMinimal,
		}
	case *events.UpgradedEvent:
		if rel, exists := p.proxyRelationships[e.ProxyAddress]; exists {
			rel.ImplementationAddress = e.ImplementationAddress
		} else {
			p.proxyRelationships[e.ProxyAddress] = &ProxyRelationship{
				ProxyAddress:          e.ProxyAddress,
				ImplementationAddress: e.ImplementationAddress,
				ProxyType:             ProxyTypeUUPS,
			}
		}
	case *events.AdminChangedEvent:
		if rel, exists := p.proxyRelationships[e.ProxyAddress]; exists {
			rel.AdminAddress = &e.NewAdmin
			if rel.ProxyType == ProxyTypeMinimal {
				rel.ProxyType = ProxyTypeTransparent
			}
		} else {
			p.proxyRelationships[e.ProxyAddress] = &ProxyRelationship{
				ProxyAddress:          e.ProxyAddress,
				ImplementationAddress: common.Address{}, // Will be set by Upgraded event if present
				AdminAddress:          &e.NewAdmin,
				ProxyType:             ProxyTypeTransparent,
			}
		}
	case *events.BeaconUpgradedEvent:
		if rel, exists := p.proxyRelationships[e.ProxyAddress]; exists {
			rel.BeaconAddress = &e.Beacon
			rel.ProxyType = ProxyTypeBeacon
		}
	}
}

// processTraceOutput processes a trace output to enrich transactions
func (p *Parser) processTraceOutput(trace *forge.TraceOutput) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Walk through trace nodes
	p.walkTraceNodes(trace.Arena)
}

// walkTraceNodes walks through trace nodes to match with transactions
func (p *Parser) walkTraceNodes(nodes []forge.TraceNode) {
	for i := range nodes {
		node := &nodes[i]
		if node.Trace.Kind == "CALL" || node.Trace.Kind == "CREATE" || node.Trace.Kind == "CREATE2" {
			// Try to match with existing transactions
			for _, tx := range p.transactions {
				// Match by various criteria (simplified)
				if tx.Transaction.To == node.Trace.Address && tx.Sender == node.Trace.Caller && common.Bytes2Hex(tx.Transaction.Data) == strings.TrimPrefix(node.Trace.Data, "0x") {
					tx.TraceData = node
					break
				}
			}
		}
	}
}

// processReceipt processes a receipt to enrich transactions
func (p *Parser) processReceipt(receipt *forge.Receipt) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to match with existing transactions by various criteria
	for _, tx := range p.transactions {
		// This is simplified - in practice we'd need more sophisticated matching
		if tx.TraceData != nil && tx.TraceData.Trace.Success {
			tx.ReceiptData = receipt
			if receipt.TxHash != "" {
				txHash := common.HexToHash(receipt.TxHash)
				tx.TxHash = &txHash
			}
			if receipt.BlockNumber > 0 {
				tx.BlockNumber = &receipt.BlockNumber
			}
			if receipt.GasUsed > 0 {
				tx.GasUsed = &receipt.GasUsed
			}
			break
		}
	}
}

// getTransactions returns all transactions in order
func (p *Parser) getTransactions() []*Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return transactions in the order they were simulated
	txs := make([]*Transaction, 0, len(p.transactionOrder))
	for _, txID := range p.transactionOrder {
		if tx, exists := p.transactions[txID]; exists {
			txs = append(txs, tx)
		}
	}

	return txs
}

// enrichFromBroadcast enriches the execution with data from the broadcast file
func (p *Parser) enrichFromBroadcast(execution *ScriptExecution, broadcastPath string) error {
	// Parse broadcast file
	parser := broadcast.NewParser(broadcastPath)
	broadcastData, err := parser.ParseBroadcastFile(broadcastPath)
	if err != nil {
		return fmt.Errorf("failed to parse broadcast file: %w", err)
	}

	// Match transactions with broadcast data
	// This is simplified - in practice we'd need more sophisticated matching
	for _, tx := range broadcastData.Transactions {
		// Find matching transaction by various criteria
		for _, execTx := range execution.Transactions {
			// Skip if already executed
			if execTx.Status == types.TransactionStatusExecuted {
				continue
			}

			toMatches := execTx.Transaction.To == common.HexToAddress(tx.Transaction.To)
			execTxDataHash := common.BytesToHash(execTx.Transaction.Data)
			txDataHash := common.BytesToHash(common.FromHex(tx.Transaction.Data))
			dataMatches := execTxDataHash == txDataHash
			fromMatches := execTx.Sender == common.HexToAddress(tx.Transaction.From)

			// Match by to address and data (simplified)
			if toMatches && dataMatches && fromMatches {
				// Update status to executed if we have a hash
				if tx.Hash != "" {
					execTx.Status = types.TransactionStatusExecuted
					txHash := common.HexToHash(tx.Hash)
					execTx.TxHash = &txHash
				}
				// Additional enrichment could go here
				break
			}
		}
	}

	return nil
}

// GetDeploymentByAddress returns the deployment record for a given address
func (e *ScriptExecution) GetDeploymentByAddress(address common.Address) *DeploymentRecord {
	for _, dep := range e.Deployments {
		if dep.Address == address {
			return dep
		}
	}
	return nil
}

// GetProxyInfo returns proxy info for an address if it's a proxy
func (e *ScriptExecution) GetProxyInfo(address common.Address) (*ProxyInfo, bool) {
	rel, exists := e.ProxyRelationships[address]
	if !exists {
		return nil, false
	}

	info := &ProxyInfo{
		Implementation: rel.ImplementationAddress,
		ProxyType:      string(rel.ProxyType),
		Admin:          rel.AdminAddress,
		Beacon:         rel.BeaconAddress,
	}

	return info, true
}

// GetTransactionByID returns a transaction by its ID
func (e *ScriptExecution) GetTransactionByID(txID [32]byte) *Transaction {
	for _, tx := range e.Transactions {
		if tx.TransactionId == txID {
			return tx
		}
	}
	return nil
}

// GetProxiesForImplementation returns all proxies pointing to an implementation
func (e *ScriptExecution) GetProxiesForImplementation(implAddress common.Address) []*ProxyRelationship {
	var proxies []*ProxyRelationship
	for _, rel := range e.ProxyRelationships {
		if rel.ImplementationAddress == implAddress {
			proxies = append(proxies, rel)
		}
	}
	return proxies
}

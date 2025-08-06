package parser

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/bindings"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
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
	contractsIndexer   *contracts.Indexer
	mu                 sync.RWMutex
}

// NewParser creates a new parser
func NewParser(contractsIndexer *contracts.Indexer) *Parser {
	return &Parser{
		eventParser:        events.NewEventParser(),
		transactions:       make(map[[32]byte]*Transaction),
		transactionOrder:   make([][32]byte, 0),
		proxyRelationships: make(map[common.Address]*ProxyRelationship),
		contractsIndexer:   contractsIndexer,
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
		Script:             result.Script,
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
			case *bindings.TrebSafeTransactionExecuted:
				p.processSafeTransactionExecuted(e, execution)
			case *bindings.TrebContractDeployed:
				deploymentRecord := &DeploymentRecord{
					TransactionID: e.TransactionId,
					Deployment:    &e.Deployment,
					Address:       e.Location,
					Deployer:      e.Deployer,
				}

				// Look up contract info by artifact
				if p.contractsIndexer != nil && e.Deployment.Artifact != "" {
					if contractInfo := p.contractsIndexer.GetContractByArtifact(e.Deployment.Artifact); contractInfo != nil {
						deploymentRecord.Contract = contractInfo
					}
				}

				execution.Deployments = append(execution.Deployments, deploymentRecord)
			}

			// Process proxy events
			p.processProxyEvent(event)
		}

		// Extract simulation traces BEFORE processing broadcast traces
		p.extractSimulationTraces(result.ParsedOutput.ScriptOutput)

		// Enrich with forge output data
		if result.ParsedOutput != nil {
			// Process trace outputs (for broadcast transactions)
			for _, trace := range result.ParsedOutput.TraceOutputs {
				p.processTraceOutput(&trace)
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
			tx.SafeTransaction = safeTx
			batchIdx := idx
			tx.SafeBatchIdx = &batchIdx
		}
	}
}

// processSafeTransactionExecuted processes a SafeTransactionExecuted event
func (p *Parser) processSafeTransactionExecuted(event *bindings.TrebSafeTransactionExecuted, execution *ScriptExecution) {
	// Create Safe transaction record  
	safeTx := &SafeTransaction{
		SafeTxHash:     event.SafeTxHash,
		Safe:           event.Safe,
		Proposer:       event.Executor, // Use executor as proposer for executed transactions
		TransactionIDs: event.TransactionIds,
		Executed:       true, // Mark as executed
	}
	execution.SafeTransactions = append(execution.SafeTransactions, safeTx)

	// Update all referenced transactions to EXECUTED status
	p.mu.Lock()
	defer p.mu.Unlock()

	for idx, txID := range event.TransactionIds {
		if tx, exists := p.transactions[txID]; exists {
			tx.Status = types.TransactionStatusExecuted
			tx.SafeTransaction = safeTx
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

// nodeMatchesTransaction checks if a trace node matches a transaction
func nodeMatchesTransaction(tx *Transaction, node *forge.TraceNode, prank *common.Address) bool {
	// Only match CALL/CREATE/CREATE2 nodes
	if node.Trace.Kind != "CALL" && node.Trace.Kind != "CREATE" && node.Trace.Kind != "CREATE2" {
		return false
	}

	// Compare call data (handle 0x prefix)
	txData := common.Bytes2Hex(tx.Transaction.Data)
	traceData := strings.TrimPrefix(node.Trace.Data, "0x")

	// Match sender
	callerMatches := tx.Sender == node.Trace.Caller
	prankMatches := prank != nil && tx.Sender == *prank

	if !callerMatches && !prankMatches {
		return false
	}

	// Match to address (for CALL only)
	if node.Trace.Kind == "CALL" && tx.Transaction.To != node.Trace.Address {
		return false
	}

	if txData != traceData {
		return false
	}

	// TODO: Also match value if needed

	return true
}

// processTraceOutput processes a trace output to enrich transactions
func (p *Parser) processTraceOutput(trace *forge.TraceOutput) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Walk through trace nodes
	if len(trace.Arena) == 0 {
		return
	}

	root := &trace.Arena[0]

	// Try to match with existing transactions
	for _, tx := range p.transactions {
		if nodeMatchesTransaction(tx, root, nil) {
			tx.TraceData = trace
			break
		}
	}
}

// extractSimulationTraces processes the full trace tree from ScriptOutput
// and extracts individual transaction traces
func (p *Parser) extractSimulationTraces(scriptOutput *forge.ScriptOutput) {
	if scriptOutput == nil || len(scriptOutput.Traces) == 0 {
		return
	}

	for _, traceWithLabel := range scriptOutput.Traces {
		// Process each labeled trace
		visitor := &traceVisitor{
			parser: p,
			label:  traceWithLabel.Label,
		}
		visitor.walkTraceTree(&traceWithLabel.Trace, 0)
	}
}

// traceVisitor helps walk the trace tree
type traceVisitor struct {
	parser *Parser
	label  string
}

// walkTraceTree recursively walks the trace tree
func (v *traceVisitor) walkTraceTree(fullTrace *forge.TraceOutput, nodeIdx int) {
	if nodeIdx >= len(fullTrace.Arena) {
		return
	}

	node := &fullTrace.Arena[nodeIdx]

	// Try to match this node with any transaction
	v.parser.mu.Lock()
	var matchedTx *Transaction
	for _, tx := range v.parser.transactions {
		// Skip if already has trace data
		if tx.TraceData != nil {
			continue
		}

		prank := v.getPrankedAddress(fullTrace, nodeIdx)

		if nodeMatchesTransaction(tx, node, prank) {
			matchedTx = tx
			break
		}
	}
	v.parser.mu.Unlock()

	// If we found a match, extract the subtree
	if matchedTx != nil {
		subtree := v.extractSubtree(fullTrace, nodeIdx)

		v.parser.mu.Lock()
		matchedTx.TraceData = subtree
		v.parser.mu.Unlock()
	}

	// Recursively process children
	for _, childIdx := range node.Children {
		v.walkTraceTree(fullTrace, childIdx)
	}
}

func (v *traceVisitor) getPrankedAddress(fullTrace *forge.TraceOutput, nodeIdx int) *common.Address {
	if nodeIdx == 0 {
		return nil
	}

	node := fullTrace.Arena[nodeIdx]
	prevNode := fullTrace.Arena[nodeIdx-1]

	if prevNode.Parent == nil || node.Parent == nil || *prevNode.Parent != *node.Parent {
		return nil
	}

	if prevNode.Trace.Kind != "CALL" {
		return nil
	}

	if prevNode.Trace.Address != common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D") {
		return nil
	}

	if strings.HasPrefix(prevNode.Trace.Data, "0xca669fa7") { // vm.prank
		prank := common.HexToAddress(strings.TrimPrefix(prevNode.Trace.Data, "0xca669fa7000000000000000000000000"))
		return &prank
	}

	return nil
}

// extractSubtree creates a new TraceOutput containing only the subtree
func (v *traceVisitor) extractSubtree(fullTrace *forge.TraceOutput, rootIdx int) *forge.TraceOutput {
	// Map old indices to new indices
	indexMap := make(map[int]int)
	newArena := []forge.TraceNode{}

	// BFS to collect all nodes in subtree
	queue := []int{rootIdx}

	for len(queue) > 0 {
		oldIdx := queue[0]
		queue = queue[1:]

		if _, exists := indexMap[oldIdx]; exists {
			continue
		}

		newIdx := len(newArena)
		indexMap[oldIdx] = newIdx

		// Copy node with updated indices
		node := fullTrace.Arena[oldIdx]
		newNode := node // Copy
		newNode.Idx = newIdx

		// Update parent reference
		if node.Parent != nil {
			if newParentIdx, exists := indexMap[*node.Parent]; exists {
				newNode.Parent = &newParentIdx
			} else {
				// Parent not in subtree, this is the root
				newNode.Parent = nil
			}
		} else {
			newNode.Parent = nil
		}

		// Clear children, will be fixed in second pass
		newNode.Children = []int{}

		newArena = append(newArena, newNode)

		// Add children to queue
		queue = append(queue, node.Children...)
	}

	// Second pass: fix children references
	for oldIdx, newIdx := range indexMap {
		oldNode := &fullTrace.Arena[oldIdx]
		newNode := &newArena[newIdx]

		for _, oldChildIdx := range oldNode.Children {
			if newChildIdx, exists := indexMap[oldChildIdx]; exists {
				newNode.Children = append(newNode.Children, newChildIdx)
			}
		}
	}

	return &forge.TraceOutput{Arena: newArena}
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

	// First, match regular transactions with broadcast data
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
					for _, receipt := range execution.ParsedOutput.Receipts {
						if receipt.TxHash == tx.Hash {
							execTx.GasUsed = &receipt.GasUsed
							execTx.BlockNumber = &receipt.BlockNumber
							break
						}
					}
				}
				// Additional enrichment could go here
				break
			}
		}
	}

	// Second, match Safe execTransaction calls with SafeTransactionExecuted events
	// Group SafeTransactionExecuted events by Safe address
	executedSafeTxBySafe := make(map[common.Address][]*SafeTransaction)
	for _, safeTx := range execution.SafeTransactions {
		if safeTx.Executed {
			executedSafeTxBySafe[safeTx.Safe] = append(executedSafeTxBySafe[safeTx.Safe], safeTx)
		}
	}

	// Find broadcast transactions that are execTransaction calls
	for _, broadcastTx := range broadcastData.Transactions {
		// Check if this is an execTransaction call
		txData := common.FromHex(broadcastTx.Transaction.Data)
		if len(txData) < 4 || !strings.EqualFold(common.Bytes2Hex(txData[:4]), "6a761202") {
			continue
		}

		// This is an execTransaction call to a Safe
		safeAddress := common.HexToAddress(broadcastTx.Transaction.To)
		
		// Find SafeTransactionExecuted events for this Safe
		safeTxs, exists := executedSafeTxBySafe[safeAddress]
		if !exists || len(safeTxs) == 0 {
			continue
		}

		// Match by order - the first unmatched SafeTransactionExecuted event
		// for this Safe should correspond to this broadcast transaction
		for _, safeTx := range safeTxs {
			// Skip if already has execution hash
			if safeTx.ExecutionTxHash != nil {
				continue
			}

			// This SafeTransactionExecuted event matches this broadcast transaction
			txHash := common.HexToHash(broadcastTx.Hash)
			safeTx.ExecutionTxHash = &txHash

			// Also find receipt for block number
			for _, receipt := range broadcastData.Receipts {
				if receipt.TransactionHash == broadcastTx.Hash {
					blockNum, _ := strconv.ParseUint(strings.TrimPrefix(receipt.BlockNumber, "0x"), 16, 64)
					safeTx.ExecutionBlockNumber = &blockNum
					break
				}
			}

			// Only match one SafeTransactionExecuted per broadcast transaction
			break
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

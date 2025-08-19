package forge

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/internal/domain/bindings"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// RunResultHydrator is our new clean implementation of the execution parser
type RunResultHydrator struct {
	projectRoot        string
	parser             usecase.ABIParser
	indexer            usecase.ContractIndexer
	transactions       map[[32]byte]*forge.Transaction
	transactionOrder   [][32]byte // Track order of transactions
	proxyRelationships map[common.Address]*forge.ProxyRelationship
	mu                 sync.RWMutex
}

// NewRunResultHydrator creates a new internal parser
func NewRunResultHydrator(projectRoot string, parser usecase.ABIParser) (*RunResultHydrator, error) {
	return &RunResultHydrator{
		projectRoot: projectRoot,
		parser:      parser,
	}, nil
}

// ParseExecution parses the script output into a structured execution result
func (h *RunResultHydrator) Hydrate(
	ctx context.Context,
	runResult *forge.RunResult,
) (*forge.HydratedRunResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Reset parser state
	h.transactions = make(map[[32]byte]*forge.Transaction)
	h.transactionOrder = make([][32]byte, 0)
	h.proxyRelationships = make(map[common.Address]*forge.ProxyRelationship)

	hydrated := &HydratedRunResult{
		RunResult:          *runResult,
		Transactions:       []*forge.Transaction{},
		SafeTransactions:   []*forge.SafeTransaction{},
		Deployments:        []*forge.Deployment{},
		ProxyRelationships: make(map[common.Address]*forge.ProxyRelationship),
		Collisions:         make(map[common.Address]*bindings.TrebDeploymentCollision),
	}

	// Parse events if we have parsed output
	if runResult.ParsedOutput != nil && runResult.ParsedOutput.ScriptOutput != nil {
		// Parse events
		parsedEvents, err := h.parser.ParseEvents(runResult.ParsedOutput.ScriptOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to parse events: %w", err)
		}
		hydrated.Events = parsedEvents

		// Process events
		for _, event := range parsedEvents {
			// Process transaction events
			switch e := event.(type) {
			case *bindings.TrebTransactionSimulated:
				h.processTransactionSimulated(e)
			case *bindings.TrebSafeTransactionQueued:
				h.processSafeTransactionQueued(e, hydrated)
			case *bindings.TrebSafeTransactionExecuted:
				h.processSafeTransactionExecuted(e, hydrated)
			case *bindings.TrebContractDeployed:
				deployment := &forge.Deployment{
					TransactionID: e.TransactionId,
					Event:         &e.Deployment,
					Address:       e.Location,
					Deployer:      e.Deployer,
				}

				// Look up contract info by artifact
				if h.indexer != nil && e.Deployment.Artifact != "" {
					if contractInfo := h.indexer.GetContractByArtifact(ctx, e.Deployment.Artifact); contractInfo != nil {
						deployment.Contract = contractInfo
					}
				}

				hydrated.Deployments = append(hydrated.Deployments, deployment)
			case *bindings.TrebDeploymentCollision:
				hydrated.Collisions[e.ExistingContract] = e
			}

			// Process proxy events
			h.processProxyEvent(event)
		}

		// Extract simulation traces BEFORE processing broadcast traces
		h.extractSimulationTraces(runResult.ParsedOutput.ScriptOutput)

		// Enrich with forge output data
		if runResult.ParsedOutput != nil {
			// Process trace outputs (for broadcast transactions)
			for _, trace := range runResult.ParsedOutput.TraceOutputs {
				h.processTraceOutput(&trace)
			}
		}

		// Get final transaction list
		hydrated.Transactions = h.getTransactions()

		// Copy proxy relationships
		for addr, rel := range h.proxyRelationships {
			hydrated.ProxyRelationships[addr] = rel
		}
	}

	// Enrich with broadcast data if available
	if runResult.BroadcastPath != "" {
		if err := h.enrichFromBroadcast(hydrated, runResult.BroadcastPath); err != nil {
			// Don't fail, just log warning
			fmt.Printf("Warning: Failed to enrich from broadcast: %v\n", err)
		}
	}

	// Unwrap the local type
	forgeHydrated := forge.HydratedRunResult(*hydrated)
	return &forgeHydrated, nil
}

// processTransactionSimulated processes a TransactionSimulated event
func (h *RunResultHydrator) processTransactionSimulated(event *bindings.TrebTransactionSimulated) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tx := &forge.Transaction{
		SimulatedTransaction: event.SimulatedTx,
		Status:               models.TransactionStatusSimulated,
		Deployments:          []forge.DeploymentInfo{},
	}

	h.transactions[event.SimulatedTx.TransactionId] = tx
	h.transactionOrder = append(h.transactionOrder, event.SimulatedTx.TransactionId)
}

// processSafeTransactionQueued processes a SafeTransactionQueued event
func (h *RunResultHydrator) processSafeTransactionQueued(event *bindings.TrebSafeTransactionQueued, hydrated *HydratedRunResult) {
	// Create Safe transaction record
	safeTx := &forge.SafeTransaction{
		SafeTxHash:     event.SafeTxHash,
		Safe:           event.Safe,
		Proposer:       event.Proposer,
		TransactionIds: event.TransactionIds,
		Executed:       false, // Mark as executed
	}
	hydrated.SafeTransactions = append(hydrated.SafeTransactions, safeTx)

	// Update all referenced transactions to QUEUED status
	h.mu.Lock()
	defer h.mu.Unlock()

	for idx, txID := range event.TransactionIds {
		if tx, exists := h.transactions[txID]; exists {
			tx.Status = models.TransactionStatusQueued
			tx.SafeTransaction = safeTx
			batchIdx := idx
			tx.SafeBatchIdx = &batchIdx
		}
	}
}

// processSafeTransactionExecuted processes a SafeTransactionExecuted event
func (h *RunResultHydrator) processSafeTransactionExecuted(event *bindings.TrebSafeTransactionExecuted, hydrated *HydratedRunResult) {
	// Create Safe transaction record
	safeTx := &forge.SafeTransaction{
		SafeTxHash:     event.SafeTxHash,
		Safe:           event.Safe,
		Proposer:       event.Executor,
		TransactionIds: event.TransactionIds,
		Executed:       true, // Mark as executed
	}
	hydrated.SafeTransactions = append(hydrated.SafeTransactions, safeTx)

	// Update all referenced transactions to EXECUTED status
	h.mu.Lock()
	defer h.mu.Unlock()

	for idx, txID := range event.TransactionIds {
		if tx, exists := h.transactions[txID]; exists {
			tx.Status = models.TransactionStatusExecuted
			tx.SafeTransaction = safeTx
			batchIdx := idx
			tx.SafeBatchIdx = &batchIdx
		}
	}
}

// processProxyEvent processes proxy-related events
func (h *RunResultHydrator) processProxyEvent(event interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	switch e := event.(type) {
	case *events.ProxyDeployedEvent:
		h.proxyRelationships[e.ProxyAddress] = &forge.ProxyRelationship{
			ProxyAddress:          e.ProxyAddress,
			ImplementationAddress: e.ImplementationAddress,
			ProxyType:             forge.ProxyTypeMinimal,
		}
	case *events.UpgradedEvent:
		if rel, exists := h.proxyRelationships[e.ProxyAddress]; exists {
			rel.ImplementationAddress = e.ImplementationAddress
		} else {
			h.proxyRelationships[e.ProxyAddress] = &forge.ProxyRelationship{
				ProxyAddress:          e.ProxyAddress,
				ImplementationAddress: e.ImplementationAddress,
				ProxyType:             forge.ProxyTypeUUPS,
			}
		}
	case *events.AdminChangedEvent:
		if rel, exists := h.proxyRelationships[e.ProxyAddress]; exists {
			rel.AdminAddress = &e.NewAdmin
			if rel.ProxyType == forge.ProxyTypeMinimal {
				rel.ProxyType = forge.ProxyTypeTransparent
			}
		} else {
			h.proxyRelationships[e.ProxyAddress] = &forge.ProxyRelationship{
				ProxyAddress:          e.ProxyAddress,
				ImplementationAddress: common.Address{}, // Will be set by Upgraded event if present
				AdminAddress:          &e.NewAdmin,
				ProxyType:             forge.ProxyTypeTransparent,
			}
		}
	case *events.BeaconUpgradedEvent:
		if rel, exists := h.proxyRelationships[e.ProxyAddress]; exists {
			rel.BeaconAddress = &e.Beacon
			rel.ProxyType = forge.ProxyTypeBeacon
		}
	}
}

// nodeMatchesTransaction checks if a trace node matches a transaction
func nodeMatchesTransaction(tx *forge.Transaction, node *forge.TraceNode, prank *common.Address) bool {
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
func (h *RunResultHydrator) processTraceOutput(trace *forge.TraceOutput) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Walk through trace nodes
	if len(trace.Arena) == 0 {
		return
	}

	root := &trace.Arena[0]

	// Try to match with existing transactions
	for _, tx := range h.transactions {
		if nodeMatchesTransaction(tx, root, nil) {
			tx.TraceData = trace
			break
		}
	}
}

// extractSimulationTraces processes the full trace tree from ScriptOutput
// and extracts individual transaction traces
func (h *RunResultHydrator) extractSimulationTraces(scriptOutput *forge.ScriptOutput) {
	if scriptOutput == nil || len(scriptOutput.Traces) == 0 {
		return
	}

	for _, traceWithLabel := range scriptOutput.Traces {
		// Process each labeled trace
		visitor := &traceVisitor{
			hydrator: h,
			label:    traceWithLabel.Label,
		}
		visitor.walkTraceTree(&traceWithLabel.Trace, 0)
	}
}

// traceVisitor helps walk the trace tree
type traceVisitor struct {
	hydrator *RunResultHydrator
	label    string
}

// walkTraceTree recursively walks the trace tree
func (v *traceVisitor) walkTraceTree(fullTrace *forge.TraceOutput, nodeIdx int) {
	if nodeIdx >= len(fullTrace.Arena) {
		return
	}

	node := &fullTrace.Arena[nodeIdx]

	// Try to match this node with any transaction
	v.hydrator.mu.Lock()
	var matchedTx *forge.Transaction
	for _, tx := range v.hydrator.transactions {
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
	v.hydrator.mu.Unlock()

	// If we found a match, extract the subtree
	if matchedTx != nil {
		subtree := v.extractSubtree(fullTrace, nodeIdx)

		v.hydrator.mu.Lock()
		matchedTx.TraceData = subtree
		v.hydrator.mu.Unlock()
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
func (h *RunResultHydrator) getTransactions() []*forge.Transaction {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return transactions in the order they were simulated
	txs := make([]*forge.Transaction, 0, len(h.transactionOrder))
	for _, txID := range h.transactionOrder {
		if tx, exists := h.transactions[txID]; exists {
			txs = append(txs, tx)
		}
	}

	return txs
}

// enrichFromBroadcast enriches the execution with data from the broadcast file
func (h *RunResultHydrator) enrichFromBroadcast(hydrated *HydratedRunResult, broadcastPath string) error {
	// Parse broadcast file
	parser := broadcast.NewParser(broadcastPath)
	broadcastData, err := parser.ParseBroadcastFile(broadcastPath)
	if err != nil {
		return fmt.Errorf("failed to parse broadcast file: %w", err)
	}

	// First, match regular transactions with broadcast data
	for _, tx := range broadcastData.Transactions {
		// Find matching transaction by various criteria
		for _, execTx := range hydrated.Transactions {
			// Skip if already executed
			if execTx.Status == models.TransactionStatusExecuted {
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
					execTx.Status = models.TransactionStatusExecuted
					txHash := common.HexToHash(tx.Hash)
					execTx.TxHash = &txHash
					// Try ParsedOutput receipts first
					if hydrated.ParsedOutput != nil {
						for _, receipt := range hydrated.ParsedOutput.Receipts {
							if receipt.TxHash == tx.Hash {
								execTx.GasUsed = &receipt.GasUsed
								execTx.BlockNumber = &receipt.BlockNumber
								break
							}
						}
					}
					// Fallback to broadcast receipts
					if execTx.BlockNumber == nil {
						for _, receipt := range broadcastData.Receipts {
							if receipt.TransactionHash == tx.Hash {
								blockNum, _ := strconv.ParseUint(strings.TrimPrefix(receipt.BlockNumber, "0x"), 16, 64)
								execTx.BlockNumber = &blockNum
								gasUsed, _ := strconv.ParseUint(strings.TrimPrefix(receipt.GasUsed, "0x"), 16, 64)
								execTx.GasUsed = &gasUsed
								break
							}
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
	executedSafeTxBySafe := make(map[common.Address][]*forge.SafeTransaction)
	for _, safeTx := range hydrated.SafeTransactions {
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

			// Also find receipt for block number and gas used
			var blockNum *uint64
			var gasUsed *uint64
			for _, receipt := range broadcastData.Receipts {
				if receipt.TransactionHash == broadcastTx.Hash {
					bn, _ := strconv.ParseUint(strings.TrimPrefix(receipt.BlockNumber, "0x"), 16, 64)
					blockNum = &bn
					gu, _ := strconv.ParseUint(strings.TrimPrefix(receipt.GasUsed, "0x"), 16, 64)
					gasUsed = &gu
					safeTx.ExecutionBlockNumber = blockNum
					break
				}
			}

			// Update all transactions that are part of this Safe transaction
			for _, txID := range safeTx.TransactionIds {
				for _, execTx := range hydrated.Transactions {
					if execTx.TransactionId == txID {
						// Update the transaction with execution details
						execTx.Status = models.TransactionStatusExecuted
						execTx.TxHash = &txHash
						if blockNum != nil {
							execTx.BlockNumber = blockNum
						}
						if gasUsed != nil {
							execTx.GasUsed = gasUsed
						}
						break
					}
				}
			}

			// Only match one SafeTransactionExecuted per broadcast transaction
			break
		}
	}

	return nil
}

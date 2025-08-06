package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// PruneItem represents an item that should be pruned with the reason
type PruneItem struct {
	ID      string
	Address string // For deployments
	Hash    string // For transactions
	Status  types.TransactionStatus
	Reason  string
}

// ItemsToPrune contains all items that should be pruned
type ItemsToPrune struct {
	Deployments      []PruneItem
	Transactions     []PruneItem
	SafeTransactions []SafePruneItem
}

// SafePruneItem represents a safe transaction that should be pruned
type SafePruneItem struct {
	SafeTxHash  string
	SafeAddress string
	Status      types.TransactionStatus
	Reason      string
}

// Pruner handles pruning of registry entries that no longer exist on-chain
type Pruner struct {
	manager *Manager
	client  *ethclient.Client
	chainID uint64
	ctx     context.Context
}

// NewPruner creates a new pruner instance
func (m *Manager) NewPruner(rpcURL string, chainID uint64) *Pruner {
	return &Pruner{
		manager: m,
		chainID: chainID,
		ctx:     context.Background(),
	}
}

// Connect establishes connection to the blockchain
func (p *Pruner) connect(rpcURL string) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	p.client = client

	// Verify chain ID matches (if chainID was specified)
	networkChainID, err := p.client.ChainID(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	// If chainID was 0, use the network's chain ID
	if p.chainID == 0 {
		p.chainID = networkChainID.Uint64()
	} else if networkChainID.Uint64() != p.chainID {
		return fmt.Errorf("chain ID mismatch: expected %d, got %d", p.chainID, networkChainID.Uint64())
	}

	return nil
}

// CollectItemsToPrune checks all registry entries and collects items that should be pruned
func (p *Pruner) CollectItemsToPrune(includePending bool) (*ItemsToPrune, error) {
	// Connect to RPC
	if p.client == nil {
		return nil, fmt.Errorf("RPC client not connected. Call Connect first")
	}

	p.manager.mu.RLock()
	defer p.manager.mu.RUnlock()

	items := &ItemsToPrune{
		Deployments:      []PruneItem{},
		Transactions:     []PruneItem{},
		SafeTransactions: []SafePruneItem{},
	}

	// Check deployments
	for id, deployment := range p.manager.deployments {
		// Only check deployments on the target chain
		if deployment.ChainID != p.chainID {
			continue
		}

		reason, shouldPrune := p.shouldPruneDeployment(deployment)
		if shouldPrune {
			items.Deployments = append(items.Deployments, PruneItem{
				ID:      id,
				Address: deployment.Address,
				Reason:  reason,
			})
		}
	}

	// Check transactions
	for id, tx := range p.manager.transactions {
		// Only check transactions on the target chain
		if tx.ChainID != p.chainID {
			continue
		}

		// Skip pending transactions unless includePending is set
		if !includePending && (tx.Status == types.TransactionStatusSimulated || tx.Status == types.TransactionStatusQueued) {
			continue
		}

		reason, shouldPrune := p.shouldPruneTransaction(tx)
		if shouldPrune {
			items.Transactions = append(items.Transactions, PruneItem{
				ID:     id,
				Hash:   tx.Hash,
				Status: tx.Status,
				Reason: reason,
			})
		}
	}

	// Check safe transactions
	for hash, safeTx := range p.manager.safeTransactions {
		// Only check safe transactions on the target chain
		if safeTx.ChainID != p.chainID {
			continue
		}

		// Skip pending safe transactions unless includePending is set
		if !includePending && safeTx.Status == types.TransactionStatusQueued {
			continue
		}

		reason, shouldPrune := p.shouldPruneSafeTransaction(safeTx)
		if shouldPrune {
			items.SafeTransactions = append(items.SafeTransactions, SafePruneItem{
				SafeTxHash:  hash,
				SafeAddress: safeTx.SafeAddress,
				Status:      safeTx.Status,
				Reason:      reason,
			})
		}
	}

	return items, nil
}

// shouldPruneDeployment checks if a deployment should be pruned
func (p *Pruner) shouldPruneDeployment(deployment *types.Deployment) (string, bool) {
	// Check if contract exists at address
	address := common.HexToAddress(deployment.Address)
	
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	code, err := p.client.CodeAt(ctx, address, nil)
	if err != nil {
		return fmt.Sprintf("failed to check code: %v", err), true
	}

	// If no code at address, contract doesn't exist
	if len(code) == 0 {
		return "no code at address", true
	}

	// Additional check: if it's a proxy, verify the implementation exists
	if deployment.ProxyInfo != nil && deployment.ProxyInfo.Implementation != "" {
		implAddress := common.HexToAddress(deployment.ProxyInfo.Implementation)
		implCode, err := p.client.CodeAt(ctx, implAddress, nil)
		if err != nil {
			return fmt.Sprintf("failed to check implementation: %v", err), true
		}
		if len(implCode) == 0 {
			return "proxy implementation missing", true
		}
	}

	return "", false
}

// shouldPruneTransaction checks if a transaction should be pruned
func (p *Pruner) shouldPruneTransaction(tx *types.Transaction) (string, bool) {
	// If transaction has no hash, it was never broadcast
	if tx.Hash == "" {
		if tx.Status == types.TransactionStatusExecuted {
			return "executed transaction has no hash", true
		}
		// For simulated/queued transactions without hash, keep them unless includePending
		return "no transaction hash", false
	}

	// Check if transaction exists on-chain
	txHash := common.HexToHash(tx.Hash)
	
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	receipt, err := p.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "transaction not found on-chain", true
		}
		// For other errors, be conservative and don't prune
		return "", false
	}

	// Transaction exists, check if it matches our records
	if receipt.BlockNumber != nil && tx.BlockNumber > 0 {
		if receipt.BlockNumber.Uint64() != tx.BlockNumber {
			return fmt.Sprintf("block number mismatch: expected %d, got %d", 
				tx.BlockNumber, receipt.BlockNumber.Uint64()), true
		}
	}

	return "", false
}

// shouldPruneSafeTransaction checks if a safe transaction should be pruned
func (p *Pruner) shouldPruneSafeTransaction(safeTx *types.SafeTransaction) (string, bool) {
	// First check if the Safe contract exists
	safeAddress := common.HexToAddress(safeTx.SafeAddress)
	
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	code, err := p.client.CodeAt(ctx, safeAddress, nil)
	if err != nil {
		return fmt.Sprintf("failed to check Safe contract: %v", err), true
	}

	if len(code) == 0 {
		return "Safe contract doesn't exist", true
	}

	// For executed safe transactions, check if the execution transaction exists
	if safeTx.Status == types.TransactionStatusExecuted && safeTx.ExecutionTxHash != "" {
		txHash := common.HexToHash(safeTx.ExecutionTxHash)
		receipt, err := p.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return "execution transaction not found", true
			}
		} else if receipt.Status == 0 {
			return "execution transaction failed", true
		}
	}

	// For queued transactions, we could check the Safe's nonce
	// but this is complex and depends on Safe version, so skip for now

	return "", false
}

// ExecutePrune removes the collected items from the registry
func (p *Pruner) ExecutePrune(items *ItemsToPrune) error {
	p.manager.mu.Lock()
	defer p.manager.mu.Unlock()

	// Remove deployments
	for _, item := range items.Deployments {
		delete(p.manager.deployments, item.ID)
	}

	// Remove transactions
	for _, item := range items.Transactions {
		delete(p.manager.transactions, item.ID)
		
		// Also remove from safe transaction references if needed
		for _, safeTx := range p.manager.safeTransactions {
			for i, txID := range safeTx.TransactionIDs {
				if txID == item.ID {
					// Remove this transaction ID from the safe tx
					safeTx.TransactionIDs = append(safeTx.TransactionIDs[:i], safeTx.TransactionIDs[i+1:]...)
					break
				}
			}
		}
	}

	// Remove safe transactions
	for _, item := range items.SafeTransactions {
		delete(p.manager.safeTransactions, item.SafeTxHash)
	}

	// Rebuild lookups after pruning
	p.manager.rebuildLookups()

	// Update solidity registry
	p.manager.rebuildSolidityRegistry()

	// Save the updated registry
	return p.manager.save()
}

// Connect initializes the RPC client
func (p *Pruner) Connect(rpcURL string) error {
	return p.connect(rpcURL)
}
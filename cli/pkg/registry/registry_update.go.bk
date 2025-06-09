package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// RegistryUpdate represents all changes to be applied to the registry
// This is built up during script execution and can be inspected before applying
type RegistryUpdate struct {
	// Deployments to add, keyed by internal transaction ID
	Deployments map[string]*DeploymentUpdate `json:"deployments"`

	// Transactions to add, keyed by internal transaction ID
	Transactions map[string]*TransactionUpdate `json:"transactions"`

	// Safe transactions to add, keyed by safe tx hash
	SafeTransactions map[string]*SafeTransactionUpdate `json:"safeTransactions"`

	// Metadata about the update
	Metadata UpdateMetadata `json:"metadata"`
}

// DeploymentUpdate represents a deployment to be added
type DeploymentUpdate struct {
	// Internal reference ID (from script events)
	InternalTxID string `json:"internalTxId"`

	// Deployment data
	Deployment *types.Deployment `json:"deployment"`

	// Enrichment data from broadcast (if available)
	BroadcastEnrichment *BroadcastEnrichment `json:"broadcastEnrichment,omitempty"`
}

// TransactionUpdate represents a transaction to be added
type TransactionUpdate struct {
	// Internal transaction ID (from script events)
	InternalID string `json:"internalId"`

	// Transaction data
	Transaction *types.Transaction `json:"transaction"`

	// On-chain data from broadcast (if available)
	OnChainData *OnChainData `json:"onChainData,omitempty"`
}

// SafeTransactionUpdate represents a Safe transaction to be added
type SafeTransactionUpdate struct {
	SafeTransaction *types.SafeTransaction `json:"safeTransaction"`

	// Map internal tx IDs to their position in the batch
	InternalTxMapping map[string]int `json:"internalTxMapping"`
}

// BroadcastEnrichment contains data from broadcast file
type BroadcastEnrichment struct {
	TransactionHash string `json:"transactionHash"`
	BlockNumber     uint64 `json:"blockNumber"`
	GasUsed         uint64 `json:"gasUsed"`
	Timestamp       uint64 `json:"timestamp"`
}

// OnChainData contains on-chain transaction data
type OnChainData struct {
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"blockNumber"`
	GasUsed     uint64 `json:"gasUsed"`
	Status      uint64 `json:"status"` // 1 for success, 0 for failure
}

// UpdateMetadata contains metadata about the update
type UpdateMetadata struct {
	ScriptPath    string    `json:"scriptPath"`
	BroadcastPath string    `json:"broadcastPath,omitempty"`
	Namespace     string    `json:"namespace"`
	ChainID       uint64    `json:"chainId"`
	NetworkName   string    `json:"networkName"`
	CreatedAt     time.Time `json:"createdAt"`
	DryRun        bool      `json:"dryRun"`
}

// NewRegistryUpdate creates a new registry update
func NewRegistryUpdate(namespace string, chainID uint64, networkName string, scriptPath string) *RegistryUpdate {
	return &RegistryUpdate{
		Deployments:      make(map[string]*DeploymentUpdate),
		Transactions:     make(map[string]*TransactionUpdate),
		SafeTransactions: make(map[string]*SafeTransactionUpdate),
		Metadata: UpdateMetadata{
			ScriptPath:  scriptPath,
			Namespace:   namespace,
			ChainID:     chainID,
			NetworkName: networkName,
			CreatedAt:   time.Now(),
		},
	}
}

// AddDeployment adds a deployment to the update
func (ru *RegistryUpdate) AddDeployment(internalTxID string, deployment *types.Deployment) {
	ru.Deployments[internalTxID] = &DeploymentUpdate{
		InternalTxID: internalTxID,
		Deployment:   deployment,
	}
}

// AddTransaction adds a transaction to the update
func (ru *RegistryUpdate) AddTransaction(internalID string, tx *types.Transaction) {
	ru.Transactions[internalID] = &TransactionUpdate{
		InternalID:  internalID,
		Transaction: tx,
	}
}

// AddSafeTransaction adds a Safe transaction to the update
func (ru *RegistryUpdate) AddSafeTransaction(safeTx *types.SafeTransaction, internalTxMapping map[string]int) {
	ru.SafeTransactions[safeTx.SafeTxHash] = &SafeTransactionUpdate{
		SafeTransaction:   safeTx,
		InternalTxMapping: internalTxMapping,
	}
}

// EnrichFromBroadcast enriches the update with data from broadcast file
func (ru *RegistryUpdate) EnrichFromBroadcast(internalTxID string, enrichment *BroadcastEnrichment) error {
	// Update deployment if exists
	if depUpdate, exists := ru.Deployments[internalTxID]; exists {
		depUpdate.BroadcastEnrichment = enrichment
	}

	// Update transaction with on-chain data
	if txUpdate, exists := ru.Transactions[internalTxID]; exists {
		txUpdate.OnChainData = &OnChainData{
			Hash:        enrichment.TransactionHash,
			BlockNumber: enrichment.BlockNumber,
			GasUsed:     enrichment.GasUsed,
			Status:      1, // Assume success if in broadcast
		}

		// Update the transaction record
		txUpdate.Transaction.Hash = enrichment.TransactionHash
		txUpdate.Transaction.BlockNumber = enrichment.BlockNumber
		txUpdate.Transaction.Status = types.TransactionStatusExecuted
	}

	return nil
}

// Apply applies the registry update to the manager
func (ru *RegistryUpdate) Apply(manager *Manager) error {
	// First, add all transactions
	for internalID, txUpdate := range ru.Transactions {
		tx := txUpdate.Transaction

		// If we have on-chain data, update the transaction
		if txUpdate.OnChainData != nil {
			tx.ID = fmt.Sprintf("tx-%s", txUpdate.OnChainData.Hash)
			tx.Hash = txUpdate.OnChainData.Hash
			tx.BlockNumber = txUpdate.OnChainData.BlockNumber
			tx.Status = types.TransactionStatusExecuted
		} else {
			// Use internal ID for pending/dry-run transactions
			tx.ID = fmt.Sprintf("tx-internal-%s", internalID)
			tx.Status = types.TransactionStatusQueued
		}

		if err := manager.AddTransaction(tx); err != nil {
			return fmt.Errorf("failed to add transaction %s: %w", tx.ID, err)
		}
	}

	// Second, add all deployments with proper transaction references
	for internalTxID, depUpdate := range ru.Deployments {
		deployment := depUpdate.Deployment

		// Update transaction ID reference
		if txUpdate, exists := ru.Transactions[internalTxID]; exists {
			deployment.TransactionID = txUpdate.Transaction.ID
		}

		// Generate deployment ID
		deployment.ID = manager.GenerateDeploymentID(
			deployment.Namespace,
			deployment.ChainID,
			deployment.ContractName,
			deployment.Label,
			deployment.TransactionID,
		)

		if err := manager.AddDeployment(deployment); err != nil {
			return fmt.Errorf("failed to add deployment %s: %w", deployment.ID, err)
		}
	}

	// Third, add Safe transactions
	for _, safeTxUpdate := range ru.SafeTransactions {
		safeTx := safeTxUpdate.SafeTransaction

		// Update transaction IDs based on internal mapping
		newTxIDs := []string{}
		for internalID := range safeTxUpdate.InternalTxMapping {
			if txUpdate, exists := ru.Transactions[internalID]; exists {
				newTxIDs = append(newTxIDs, txUpdate.Transaction.ID)
			}
		}
		safeTx.TransactionIDs = newTxIDs

		if err := manager.AddSafeTransaction(safeTx); err != nil {
			return fmt.Errorf("failed to add safe transaction %s: %w", safeTx.SafeTxHash, err)
		}
	}

	return nil
}

// GetSummary returns a summary of the update
func (ru *RegistryUpdate) GetSummary() string {
	return fmt.Sprintf(
		"  Deployments: %d\n"+
			"  Transactions: %d\n"+
			"  Safe Transactions: %d\n"+
			"  Namespace: %s\n"+
			"  Chain ID: %d\n"+
			"  Dry Run: %v",
		len(ru.Deployments),
		len(ru.Transactions),
		len(ru.SafeTransactions),
		ru.Metadata.Namespace,
		ru.Metadata.ChainID,
		ru.Metadata.DryRun,
	)
}

// MatchDeploymentToBroadcast attempts to match a deployment to broadcast data
func MatchDeploymentToBroadcast(deployment *types.Deployment, contractAddress common.Address) bool {
	// Match by address
	return strings.EqualFold(deployment.Address, contractAddress.Hex())
}

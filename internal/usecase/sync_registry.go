package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// SyncRegistry handles syncing the registry with on-chain state
type SyncRegistry struct {
	deploymentStore     DeploymentStore
	transactionStore    TransactionStore
	safeTransactionStore SafeTransactionStore
	safeClient          SafeClient
	progress            ProgressSink
}

// NewSyncRegistry creates a new sync registry use case
func NewSyncRegistry(
	deploymentStore DeploymentStore,
	transactionStore TransactionStore,
	safeTransactionStore SafeTransactionStore,
	safeClient SafeClient,
	progress ProgressSink,
) *SyncRegistry {
	return &SyncRegistry{
		deploymentStore:      deploymentStore,
		transactionStore:     transactionStore,
		safeTransactionStore: safeTransactionStore,
		safeClient:           safeClient,
		progress:             progress,
	}
}

// SyncOptions contains options for syncing
type SyncOptions struct {
	Clean bool // Remove invalid entries while syncing
	Debug bool // Show debug information
}

// SyncResult contains the result of syncing
type SyncResult struct {
	PendingSafeTxsChecked   int
	SafeTxsExecuted         int
	TransactionsUpdated     int
	DeploymentsUpdated      int
	InvalidEntriesRemoved   int
	Errors                  []string
}

// Sync performs the registry sync operation
func (s *SyncRegistry) Sync(ctx context.Context, options SyncOptions) (*SyncResult, error) {
	result := &SyncResult{
		Errors: make([]string, 0),
	}

	// Report initial progress
	s.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "sync",
		Message: "Starting registry sync...",
		Spinner: true,
	})

	// Sync pending Safe transactions
	safeSyncResult, err := s.syncPendingSafeTransactions(ctx, options.Debug)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to sync Safe transactions: %v", err))
	} else {
		result.PendingSafeTxsChecked = safeSyncResult.Checked
		result.SafeTxsExecuted = safeSyncResult.Executed
		result.TransactionsUpdated = safeSyncResult.TransactionsUpdated
		result.DeploymentsUpdated = safeSyncResult.DeploymentsUpdated
	}

	// Clean invalid entries if requested
	if options.Clean {
		s.progress.OnProgress(ctx, ProgressEvent{
			Stage:   "sync",
			Message: "Cleaning invalid entries...",
			Spinner: true,
		})
		
		cleanResult, err := s.cleanInvalidEntries(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to clean invalid entries: %v", err))
		} else {
			result.InvalidEntriesRemoved = cleanResult
		}
	}

	// Report completion
	s.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "sync",
		Message: "Registry sync completed",
		Spinner: false,
	})

	return result, nil
}

// SafeSyncResult contains results from syncing Safe transactions
type SafeSyncResult struct {
	Checked              int
	Executed             int
	TransactionsUpdated  int
	DeploymentsUpdated   int
}

// syncPendingSafeTransactions checks pending Safe transactions and updates their status
func (s *SyncRegistry) syncPendingSafeTransactions(ctx context.Context, debug bool) (*SafeSyncResult, error) {
	result := &SafeSyncResult{}

	// Get all Safe transactions
	safeTxs, err := s.safeTransactionStore.ListSafeTransactions(ctx, SafeTransactionFilter{
		Status: domain.TransactionStatusQueued,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list Safe transactions: %w", err)
	}

	if len(safeTxs) == 0 {
		return result, nil
	}

	// Group by chain
	pendingByChain := make(map[uint64][]*domain.SafeTransaction)
	for _, safeTx := range safeTxs {
		pendingByChain[safeTx.ChainID] = append(pendingByChain[safeTx.ChainID], safeTx)
	}

	// Check each chain
	for chainID, chainSafeTxs := range pendingByChain {
		s.progress.OnProgress(ctx, ProgressEvent{
			Stage:   "sync",
			Message: fmt.Sprintf("Checking %d pending Safe transaction(s) on chain %d", len(chainSafeTxs), chainID),
			Current: result.Checked,
			Total:   len(safeTxs),
		})

		// Configure Safe client for this chain
		if err := s.safeClient.SetChain(ctx, chainID); err != nil {
			continue // Skip this chain if we can't configure the client
		}

		// Check each pending Safe transaction
		for _, safeTx := range chainSafeTxs {
			result.Checked++

			// Check if transaction is executed
			executionInfo, err := s.safeClient.GetTransactionExecutionInfo(ctx, safeTx.SafeTxHash)
			if err != nil {
				continue
			}

			if executionInfo.IsExecuted {
				// Update the Safe transaction
				safeTx.Status = domain.TransactionStatusExecuted
				safeTx.ExecutionTxHash = executionInfo.TxHash
				now := time.Now()
				safeTx.ExecutedAt = &now

				// Save the updated Safe transaction
				if err := s.safeTransactionStore.SaveSafeTransaction(ctx, safeTx); err != nil {
					continue
				}
				result.Executed++

				// Update related transactions
				updatedTxs, err := s.updateTransactionsForSafeTx(ctx, safeTx)
				if err == nil {
					result.TransactionsUpdated += updatedTxs
				}

				// Update related deployments
				updatedDeps, err := s.updateDeploymentsForSafeTx(ctx, safeTx)
				if err == nil {
					result.DeploymentsUpdated += updatedDeps
				}
			} else {
				// Update confirmation count if changed
				if executionInfo.Confirmations != safeTx.ConfirmationCount {
					safeTx.ConfirmationCount = executionInfo.Confirmations
					safeTx.Confirmations = executionInfo.ConfirmationDetails
					_ = s.safeTransactionStore.SaveSafeTransaction(ctx, safeTx)
				}
			}
		}
	}

	return result, nil
}

// updateTransactionsForSafeTx updates transaction records when a Safe tx is executed
func (s *SyncRegistry) updateTransactionsForSafeTx(ctx context.Context, safeTx *domain.SafeTransaction) (int, error) {
	updated := 0

	for _, txID := range safeTx.TransactionIDs {
		tx, err := s.transactionStore.GetTransaction(ctx, txID)
		if err != nil {
			continue
		}

		// Update transaction with execution details
		tx.Hash = safeTx.ExecutionTxHash
		tx.Status = domain.TransactionStatusExecuted
		if safeTx.ExecutedAt != nil {
			tx.CreatedAt = *safeTx.ExecutedAt
		}

		if err := s.transactionStore.SaveTransaction(ctx, tx); err == nil {
			updated++
		}
	}

	return updated, nil
}

// updateDeploymentsForSafeTx updates deployment records when a Safe tx is executed
func (s *SyncRegistry) updateDeploymentsForSafeTx(ctx context.Context, safeTx *domain.SafeTransaction) (int, error) {
	updated := 0

	// Get all deployments and check if they reference any of the transactions
	for _, txID := range safeTx.TransactionIDs {
		// Find deployments that reference this transaction
		deployments, err := s.deploymentStore.ListDeployments(ctx, DeploymentFilter{})
		if err != nil {
			continue
		}

		for _, deployment := range deployments {
			if deployment.TransactionID == txID {
				// The deployment's transaction is now executed
				// This might trigger additional verification or status updates
				deployment.UpdatedAt = time.Now()
				if err := s.deploymentStore.SaveDeployment(ctx, deployment); err == nil {
					updated++
				}
			}
		}
	}

	return updated, nil
}

// cleanInvalidEntries removes invalid entries from the registry
func (s *SyncRegistry) cleanInvalidEntries(ctx context.Context) (int, error) {
	removed := 0

	// TODO: Implement cleanup logic
	// This would involve:
	// 1. Finding deployments with invalid transaction references
	// 2. Finding transactions that don't exist on-chain
	// 3. Finding orphaned Safe transactions
	// 4. Removing or fixing these entries

	return removed, nil
}
package deployments

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// CollectPrunableItems checks all registry entries and collects items that should be pruned
func (s *RegistryStoreAdapter) CollectPrunableItems(
	ctx context.Context,
	chainID uint64,
	includePending bool,
	checker usecase.BlockchainChecker,
) (*domain.ItemsToPrune, error) {
	items := &domain.ItemsToPrune{
		Deployments:      []domain.PruneItem{},
		Transactions:     []domain.PruneItem{},
		SafeTransactions: []domain.SafePruneItem{},
	}

	// Check deployments
	deployments := s.manager.GetAllDeployments()
	for _, deployment := range deployments {
		// Only check deployments on the target chain
		if deployment.ChainID != chainID {
			continue
		}

		reason, shouldPrune := s.shouldPruneDeployment(ctx, deployment, checker)
		if shouldPrune {
			items.Deployments = append(items.Deployments, domain.PruneItem{
				ID:      deployment.ID,
				Address: deployment.Address,
				Reason:  reason,
			})
		}
	}

	// Check transactions
	transactions := s.manager.GetAllTransactions()
	for _, tx := range transactions {
		// Only check transactions on the target chain
		if tx.ChainID != chainID {
			continue
		}

		// Skip pending transactions unless includePending is set
		if !includePending && (tx.Status == models.TransactionStatusSimulated || tx.Status == models.TransactionStatusQueued) {
			continue
		}

		reason, shouldPrune := s.shouldPruneTransaction(ctx, tx, checker)
		if shouldPrune {
			items.Transactions = append(items.Transactions, domain.PruneItem{
				ID:     tx.ID,
				Hash:   tx.Hash,
				Status: models.TransactionStatus(tx.Status),
				Reason: reason,
			})
		}
	}

	// Check safe transactions
	safeTransactions := s.manager.GetAllSafeTransactions()
	for _, safeTx := range safeTransactions {
		// Only check safe transactions on the target chain
		if safeTx.ChainID != chainID {
			continue
		}

		// Skip pending safe transactions unless includePending is set
		if !includePending && safeTx.Status == models.SafeTxStatusQueued {
			continue
		}

		reason, shouldPrune := s.shouldPruneSafeTransaction(ctx, safeTx, checker)
		if shouldPrune {
			items.SafeTransactions = append(items.SafeTransactions, domain.SafePruneItem{
				SafeTxHash:  safeTx.SafeTxHash,
				SafeAddress: safeTx.SafeAddress,
				Status:      models.TransactionStatus(safeTx.Status),
				Reason:      reason,
			})
		}
	}

	return items, nil
}

// shouldPruneDeployment checks if a deployment should be pruned
func (s *RegistryStoreAdapter) shouldPruneDeployment(
	ctx context.Context,
	deployment *models.Deployment,
	checker usecase.BlockchainChecker,
) (string, bool) {
	// Check if contract exists at address
	exists, reason, err := checker.CheckDeploymentExists(ctx, deployment.Address)
	if err != nil {
		// Be conservative on errors - don't prune
		return "", false
	}

	if !exists {
		return reason, true
	}

	// Additional check: if it's a proxy, verify the implementation exists
	if deployment.ProxyInfo != nil && deployment.ProxyInfo.Implementation != "" {
		implExists, implReason, err := checker.CheckDeploymentExists(ctx, deployment.ProxyInfo.Implementation)
		if err != nil {
			// Be conservative on errors - don't prune
			return "", false
		}
		if !implExists {
			return fmt.Sprintf("proxy implementation missing: %s", implReason), true
		}
	}

	return "", false
}

// shouldPruneTransaction checks if a transaction should be pruned
func (s *RegistryStoreAdapter) shouldPruneTransaction(
	ctx context.Context,
	tx *models.Transaction,
	checker usecase.BlockchainChecker,
) (string, bool) {
	// If transaction has no hash, it was never broadcast
	if tx.Hash == "" {
		if tx.Status == models.TransactionStatusExecuted {
			return "executed transaction has no hash", true
		}
		// For simulated/queued transactions without hash, don't prune unless includePending
		return "", false
	}

	// Check if transaction exists on-chain
	exists, blockNumber, reason, err := checker.CheckTransactionExists(ctx, tx.Hash)
	if err != nil {
		// Be conservative on errors - don't prune
		return "", false
	}

	if !exists {
		return reason, true
	}

	// Transaction exists, check if block number matches our records
	if blockNumber > 0 && tx.BlockNumber > 0 && blockNumber != tx.BlockNumber {
		return fmt.Sprintf("block number mismatch: expected %d, got %d", tx.BlockNumber, blockNumber), true
	}

	return "", false
}

// shouldPruneSafeTransaction checks if a safe transaction should be pruned
func (s *RegistryStoreAdapter) shouldPruneSafeTransaction(
	ctx context.Context,
	safeTx *models.SafeTransaction,
	checker usecase.BlockchainChecker,
) (string, bool) {
	// First check if the Safe contract exists
	exists, reason, err := checker.CheckSafeContract(ctx, safeTx.SafeAddress)
	if err != nil {
		// Be conservative on errors - don't prune
		return "", false
	}

	if !exists {
		return fmt.Sprintf("Safe contract doesn't exist: %s", reason), true
	}

	// For executed safe transactions, check if the execution transaction exists
	if safeTx.Status == models.SafeTxStatusExecuted && safeTx.ExecutionTxHash != "" {
		txExists, _, txReason, err := checker.CheckTransactionExists(ctx, safeTx.ExecutionTxHash)
		if err != nil {
			// Be conservative on errors - don't prune
			return "", false
		}
		if !txExists {
			return fmt.Sprintf("execution transaction not found: %s", txReason), true
		}
	}

	return "", false
}

// ExecutePrune removes the collected items from the registry
func (s *RegistryStoreAdapter) ExecutePrune(ctx context.Context, items *domain.ItemsToPrune) error {
	// Remove deployments
	for _, item := range items.Deployments {
		if err := s.manager.RemoveDeployment(item.ID); err != nil {
			return fmt.Errorf("failed to remove deployment %s: %w", item.ID, err)
		}
	}

	// Remove transactions
	for _, item := range items.Transactions {
		if err := s.manager.RemoveTransaction(item.ID); err != nil {
			return fmt.Errorf("failed to remove transaction %s: %w", item.ID, err)
		}
	}

	// Remove safe transactions
	for _, item := range items.SafeTransactions {
		if err := s.manager.RemoveSafeTransaction(item.SafeTxHash); err != nil {
			return fmt.Errorf("failed to remove safe transaction %s: %w", item.SafeTxHash, err)
		}
	}

	return nil
}

// Ensure RegistryStoreAdapter implements RegistryPruner
var _ usecase.RegistryPruner = (*RegistryStoreAdapter)(nil)

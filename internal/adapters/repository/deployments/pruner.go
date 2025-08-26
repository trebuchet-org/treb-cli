package deployments

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

type Pruner struct {
	repo    usecase.DeploymentRepository
	checker usecase.BlockchainChecker
}

func NewPruner(repo usecase.DeploymentRepository, checker usecase.BlockchainChecker) *Pruner {
	return &Pruner{repo: repo, checker: checker}
}

// CollectPrunableItems checks all registry entries and collects items that should be pruned
func (p *Pruner) CollectPrunableItems(
	ctx context.Context,
	chainID uint64,
	includePending bool,
) (*models.Changeset, error) {
	changeset := &models.Changeset{
		Delete: models.ChangesetModels{
			Deployments:      []*models.Deployment{},
			Transactions:     []*models.Transaction{},
			SafeTransactions: []*models.SafeTransaction{},
			Metadata: models.ChangesetMetadata{
				Reasons: map[string]string{},
			},
		},
	}

	// Check deployments
	deployments := p.repo.GetAllDeployments(ctx)
	for _, deployment := range deployments {
		// Only check deployments on the target chain
		if deployment.ChainID != chainID {
			continue
		}

		reason, shouldPrune := p.shouldPruneDeployment(ctx, deployment)
		if shouldPrune {
			changeset.Delete.Deployments = append(changeset.Delete.Deployments, deployment)
			changeset.Delete.Metadata.Reasons[deployment.ID] = reason
		}
	}

	// Check transactions
	transactions := p.repo.GetAllTransactions(ctx)
	for _, tx := range transactions {
		// Only check transactions on the target chain
		if tx.ChainID != chainID {
			continue
		}

		// Skip pending transactions unless includePending is set
		if !includePending && (tx.Status == models.TransactionStatusSimulated || tx.Status == models.TransactionStatusQueued) {
			continue
		}

		reason, shouldPrune := p.shouldPruneTransaction(ctx, tx)
		if shouldPrune {
			changeset.Delete.Transactions = append(changeset.Delete.Transactions, tx)
			changeset.Delete.Metadata.Reasons[tx.ID] = reason
		}
	}

	// Check safe transactions
	safeTransactions := p.repo.GetAllSafeTransactions(ctx)
	for _, safeTx := range safeTransactions {
		// Only check safe transactions on the target chain
		if safeTx.ChainID != chainID {
			continue
		}

		// Skip pending safe transactions unless includePending is set
		if !includePending && safeTx.Status == models.TransactionStatusQueued {
			continue
		}

		reason, shouldPrune := p.shouldPruneSafeTransaction(ctx, safeTx)
		if shouldPrune {
			changeset.Delete.SafeTransactions = append(changeset.Create.SafeTransactions, safeTx)
			changeset.Delete.Metadata.Reasons[safeTx.SafeTxHash] = reason
		}
	}

	return changeset, nil
}

// shouldPruneDeployment checks if a deployment should be pruned
func (p *Pruner) shouldPruneDeployment(
	ctx context.Context,
	deployment *models.Deployment,
) (string, bool) {
	// Check if contract exists at address
	exists, reason, err := p.checker.CheckDeploymentExists(ctx, deployment.Address)
	if err != nil {
		// Be conservative on errors - don't prune
		return "", false
	}

	if !exists {
		return reason, true
	}

	// Additional check: if it's a proxy, verify the implementation exists
	if deployment.ProxyInfo != nil && deployment.ProxyInfo.Implementation != "" {
		implExists, implReason, err := p.checker.CheckDeploymentExists(ctx, deployment.ProxyInfo.Implementation)
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
func (p *Pruner) shouldPruneTransaction(
	ctx context.Context,
	tx *models.Transaction,
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
	exists, blockNumber, reason, err := p.checker.CheckTransactionExists(ctx, tx.Hash)
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
func (p *Pruner) shouldPruneSafeTransaction(
	ctx context.Context,
	safeTx *models.SafeTransaction,
) (string, bool) {
	// First check if the Safe contract exists
	exists, reason, err := p.checker.CheckSafeContract(ctx, safeTx.SafeAddress)
	if err != nil {
		// Be conservative on errors - don't prune
		return "", false
	}

	if !exists {
		return fmt.Sprintf("Safe contract doesn't exist: %s", reason), true
	}

	// For executed safe transactions, check if the execution transaction exists
	if safeTx.Status == models.TransactionStatusExecuted && safeTx.ExecutionTxHash != "" {
		txExists, _, txReason, err := p.checker.CheckTransactionExists(ctx, safeTx.ExecutionTxHash)
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

// Ensure RegistryStoreAdapter implements RegistryPruner
var _ usecase.DeploymentRepositoryPruner = (*Pruner)(nil)

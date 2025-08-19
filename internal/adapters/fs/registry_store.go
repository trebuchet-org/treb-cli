package fs

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/adapters/registry"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// RegistryStoreAdapter wraps the internal registry.Manager to implement DeploymentStore
type RegistryStoreAdapter struct {
	manager *registry.Manager
}

// NewRegistryStoreAdapter creates a new adapter wrapping the internal registry manager
func NewRegistryStoreAdapter(cfg *config.RuntimeConfig) (*RegistryStoreAdapter, error) {
	manager, err := registry.NewManager(cfg.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}
	return &RegistryStoreAdapter{manager: manager}, nil
}

// GetDeployment retrieves a deployment by ID
func (r *RegistryStoreAdapter) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	dep, err := r.manager.GetDeployment(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return dep, nil
}

// GetDeploymentByAddress retrieves a deployment by chain ID and address
func (r *RegistryStoreAdapter) GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*domain.Deployment, error) {
	dep, err := r.manager.GetDeploymentByAddress(chainID, address)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return dep, nil
}

// ListDeployments retrieves deployments matching the filter
func (r *RegistryStoreAdapter) ListDeployments(ctx context.Context, filter domain.DeploymentFilter) ([]*domain.Deployment, error) {
	// Get all deployments and filter them
	allDeps := r.manager.GetAllDeploymentsHydrated()

	var result []*domain.Deployment
	for _, dep := range allDeps {
		// Apply filters
		if filter.Namespace != "" && dep.Namespace != filter.Namespace {
			continue
		}
		if filter.ChainID != 0 && dep.ChainID != filter.ChainID {
			continue
		}
		if filter.ContractName != "" && dep.ContractName != filter.ContractName {
			continue
		}
		if filter.Label != "" && dep.Label != filter.Label {
			continue
		}
		if filter.Type != "" && dep.Type != filter.Type {
			continue
		}

		result = append(result, dep)
	}

	return result, nil
}

// SaveDeployment saves a deployment
func (r *RegistryStoreAdapter) SaveDeployment(ctx context.Context, deployment *domain.Deployment) error {
	return r.manager.SaveDeployment(deployment)
}

// DeleteDeployment deletes a deployment by ID
func (r *RegistryStoreAdapter) DeleteDeployment(ctx context.Context, id string) error {
	return r.manager.DeleteDeployment(id)
}

// UpdateDeploymentVerification updates the verification status of a deployment
func (r *RegistryStoreAdapter) UpdateDeploymentVerification(ctx context.Context, id string, status domain.VerificationStatus) error {
	return r.manager.UpdateDeploymentVerification(id, status, nil)
}

// TagDeployment adds a tag to a deployment
func (r *RegistryStoreAdapter) TagDeployment(ctx context.Context, id string, tag string) error {
	return r.manager.TagDeployment(id, tag)
}

// GetTransaction retrieves a transaction by ID
func (r *RegistryStoreAdapter) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	tx, err := r.manager.GetTransaction(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return tx, nil
}

// SaveTransaction saves a transaction
func (r *RegistryStoreAdapter) SaveTransaction(ctx context.Context, tx *domain.Transaction) error {
	return r.manager.SaveTransaction(tx)
}

// RemoveDeployments removes multiple deployments
func (r *RegistryStoreAdapter) RemoveDeployments(ctx context.Context, deploymentIDs []string) error {
	for _, id := range deploymentIDs {
		if err := r.manager.DeleteDeployment(id); err != nil {
			return fmt.Errorf("failed to delete deployment %s: %w", id, err)
		}
	}
	return nil
}

// RemoveTransactions removes multiple transactions
func (r *RegistryStoreAdapter) RemoveTransactions(ctx context.Context, transactionIDs []string) error {
	// TODO: Implement transaction removal in manager
	return fmt.Errorf("transaction removal not yet implemented")
}

// RemoveSafeTransactions removes multiple safe transactions
func (r *RegistryStoreAdapter) RemoveSafeTransactions(ctx context.Context, safeTxIDs []string) error {
	// TODO: Implement safe transaction removal in manager
	return fmt.Errorf("safe transaction removal not yet implemented")
}

// GetSafeTransaction retrieves a safe transaction by ID
func (r *RegistryStoreAdapter) GetSafeTransaction(ctx context.Context, id string) (*domain.SafeTransaction, error) {
	tx, err := r.manager.GetSafeTransaction(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return tx, nil
}

// SaveSafeTransaction saves a safe transaction
func (r *RegistryStoreAdapter) SaveSafeTransaction(ctx context.Context, tx *domain.SafeTransaction) error {
	return r.manager.SaveSafeTransaction(tx)
}

// GetPendingSafeTransactions returns all pending safe transactions
func (r *RegistryStoreAdapter) GetPendingSafeTransactions(ctx context.Context) ([]*domain.SafeTransaction, error) {
	return r.manager.GetPendingSafeTransactions()
}

// UpdateVerification updates verification info for a deployment
func (r *RegistryStoreAdapter) UpdateVerification(ctx context.Context, deploymentID string, info domain.VerificationInfo) error {
	dep, err := r.manager.GetDeployment(deploymentID)
	if err != nil {
		return err
	}

	// Update verification info
	dep.Verification = info

	// Save the updated deployment
	return r.manager.SaveDeployment(dep)
}

// UpdateSafeTransaction updates a safe transaction
func (r *RegistryStoreAdapter) UpdateSafeTransaction(ctx context.Context, safeTx *domain.SafeTransaction) error {
	return r.manager.UpdateSafeTransaction(safeTx)
}

// ListTransactions lists transactions based on filter criteria
func (r *RegistryStoreAdapter) ListTransactions(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, error) {
	all := r.manager.GetAllTransactions()
	var result []*domain.Transaction

	for _, tx := range all {
		// Apply filters
		if filter.ChainID != 0 && tx.ChainID != filter.ChainID {
			continue
		}
		if filter.Status != "" && tx.Status != filter.Status {
			continue
		}
		if filter.Namespace != "" && tx.Environment != filter.Namespace {
			continue
		}

		result = append(result, tx)
	}

	return result, nil
}

// ListSafeTransactions lists safe transactions based on filter criteria
func (r *RegistryStoreAdapter) ListSafeTransactions(ctx context.Context, filter domain.SafeTransactionFilter) ([]*domain.SafeTransaction, error) {
	all := r.manager.GetAllSafeTransactions()
	var result []*domain.SafeTransaction

	for _, tx := range all {
		// Apply filters
		if filter.ChainID != 0 && tx.ChainID != filter.ChainID {
			continue
		}
		if filter.SafeAddress != "" && tx.SafeAddress != filter.SafeAddress {
			continue
		}
		if filter.Status != "" && !matchSafeTransactionStatus(tx.Status, filter.Status) {
			continue
		}

		result = append(result, tx)
	}

	return result, nil
}


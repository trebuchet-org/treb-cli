package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// ResetRegistryParams contains parameters for resetting the registry
type ResetRegistryParams struct {
	DryRun bool // If true, only collect items without executing reset
}

// ResetRegistryResult contains the result of resetting the registry
type ResetRegistryResult struct {
	Changeset *models.Changeset
}

// ResetRegistry is a use case for resetting registry entries for a namespace/network
type ResetRegistry struct {
	config          *config.RuntimeConfig
	repo            DeploymentRepository
	registryUpdater DeploymentRepositoryUpdater
}

// NewResetRegistry creates a new ResetRegistry use case
func NewResetRegistry(
	config *config.RuntimeConfig,
	repo DeploymentRepository,
	registryUpdater DeploymentRepositoryUpdater,
) *ResetRegistry {
	return &ResetRegistry{
		config:          config,
		repo:            repo,
		registryUpdater: registryUpdater,
	}
}

// Run executes the reset registry use case
func (uc *ResetRegistry) Run(ctx context.Context, params ResetRegistryParams) (*ResetRegistryResult, error) {
	if uc.config.Network == nil {
		return nil, fmt.Errorf("network is required for reset")
	}

	namespace := uc.config.Namespace
	chainID := uc.config.Network.ChainID

	// Collect deployments matching namespace and chainID
	deployments, err := uc.repo.ListDeployments(ctx, domain.DeploymentFilter{
		Namespace: namespace,
		ChainID:   chainID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Collect transactions matching namespace and chainID
	transactions, err := uc.repo.ListTransactions(ctx, domain.TransactionFilter{
		Namespace: namespace,
		ChainID:   chainID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	// Collect safe transactions matching chainID
	safeTransactions, err := uc.repo.ListSafeTransactions(ctx, domain.SafeTransactionFilter{
		ChainID: chainID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list safe transactions: %w", err)
	}

	changeset := &models.Changeset{
		Delete: models.ChangesetModels{
			Deployments:      deployments,
			Transactions:     transactions,
			SafeTransactions: safeTransactions,
		},
	}

	if !changeset.HasChanges() || params.DryRun {
		return &ResetRegistryResult{
			Changeset: changeset,
		}, nil
	}

	if err := uc.registryUpdater.ApplyChangeset(ctx, changeset); err != nil {
		return nil, fmt.Errorf("failed to reset registry: %w", err)
	}

	return &ResetRegistryResult{
		Changeset: changeset,
	}, nil
}

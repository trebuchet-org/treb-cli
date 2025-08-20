package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// PruneRegistryParams contains parameters for pruning the registry
type PruneRegistryParams struct {
	NetworkName    string
	IncludePending bool
	DryRun         bool // If true, only collect items without executing prune
}

// PruneRegistryResult contains the result of pruning the registry
type PruneRegistryResult struct {
	Changeset *models.Changeset
}

// PruneRegistry is a use case for pruning invalid registry entries
type PruneRegistry struct {
	networkResolver   NetworkResolver
	blockchainChecker BlockchainChecker
	registryPruner    DeploymentRepositoryPruner
	registryUpdater   DeploymentRepositoryUpdater
	progress          ProgressSink
}

// NewPruneRegistry creates a new PruneRegistry use case
func NewPruneRegistry(
	networkResolver NetworkResolver,
	blockchainChecker BlockchainChecker,
	registryPruner DeploymentRepositoryPruner,
	registryUpdater DeploymentRepositoryUpdater,
	progress ProgressSink,
) *PruneRegistry {
	if progress == nil {
		progress = NopProgress{}
	}
	return &PruneRegistry{
		networkResolver:   networkResolver,
		blockchainChecker: blockchainChecker,
		registryPruner:    registryPruner,
		registryUpdater:   registryUpdater,
		progress:          progress,
	}
}

// Run executes the prune registry use case
func (uc *PruneRegistry) Run(ctx context.Context, params PruneRegistryParams) (*PruneRegistryResult, error) {
	// Resolve network configuration
	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "resolve_network",
		Message: fmt.Sprintf("Resolving network: %s", params.NetworkName),
		Spinner: true,
	})

	networkInfo, err := uc.networkResolver.ResolveNetwork(ctx, params.NetworkName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network: %w", err)
	}

	// Connect to blockchain
	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "connect_blockchain",
		Message: fmt.Sprintf("Connecting to RPC: %s", networkInfo.RPCURL),
		Spinner: true,
	})

	if err := uc.blockchainChecker.Connect(ctx, networkInfo.RPCURL, networkInfo.ChainID); err != nil {
		return nil, fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	// Collect items to prune
	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "collect_items",
		Message: "Checking registry entries against on-chain state",
		Spinner: true,
	})

	changeset, err := uc.registryPruner.CollectPrunableItems(
		ctx,
		networkInfo.ChainID,
		params.IncludePending,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to collect items to prune: %w", err)
	}

	// If no items to prune or dry run, return early
	if !changeset.HasChanges() {
		return &PruneRegistryResult{
			Changeset: changeset,
		}, nil
	}

	// Execute prune
	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage:   "execute_prune",
		Message: fmt.Sprintf("Pruning %d items from registry", changeset.Delete.Count()),
		Spinner: true,
	})

	if err := uc.registryUpdater.ApplyChangeset(ctx, changeset); err != nil {
		return nil, fmt.Errorf("failed to prune items: %w", err)
	}

	return &PruneRegistryResult{
		Changeset: changeset,
	}, nil
}

package usecase

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// ListDeploymentsParams contains parameters for listing deployments
type ListDeploymentsParams struct {
	// Filter parameters (namespace and chainID come from RuntimeConfig)
	ContractName string
	Label        string
	Type         models.DeploymentType
	ForkOnly     bool // Only show fork-added deployments
	NoFork       bool // Only show pre-fork deployments
}

// ListDeployments is the use case for listing deployments
type ListDeployments struct {
	config          *config.RuntimeConfig
	repo            DeploymentRepository
	networkResolver NetworkResolver
	forkState       ForkStateStore
}

// NewListDeployments creates a new ListDeployments use case
func NewListDeployments(cfg *config.RuntimeConfig, repo DeploymentRepository, networkResolver NetworkResolver, forkState ForkStateStore) *ListDeployments {
	return &ListDeployments{
		config:          cfg,
		repo:            repo,
		networkResolver: networkResolver,
		forkState:       forkState,
	}
}

// Run executes the list deployments use case
func (uc *ListDeployments) Run(ctx context.Context, params ListDeploymentsParams) (*DeploymentListResult, error) {
	// Create filter from params and runtime config
	filter := domain.DeploymentFilter{
		Namespace:    uc.config.Namespace,
		ContractName: params.ContractName,
		Label:        params.Label,
		Type:         params.Type,
	}

	if uc.config.Network != nil {
		filter.ChainID = uc.config.Network.ChainID
	}

	// Get deployments from store
	deployments, err := uc.repo.ListDeployments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Check for active fork and compute fork deployment IDs
	forkDeploymentIDs := uc.computeForkDeploymentIDs(ctx)

	// Filter based on fork flags
	if forkDeploymentIDs != nil && (params.ForkOnly || params.NoFork) {
		deployments = filterByFork(deployments, forkDeploymentIDs, params.ForkOnly)
	}

	// Sort deployments for consistent output
	sortDeployments(deployments)

	// Calculate summary
	summary := calculateSummary(deployments)

	// Build network names map
	networkNames := make(map[uint64]string)
	for chainID := range summary.ByChain {
		// Try to find network name for this chain ID
		networkName := uc.findNetworkName(ctx, chainID)
		if networkName != "" {
			networkNames[chainID] = networkName
		}
	}

	return &DeploymentListResult{
		Deployments:       deployments,
		Summary:           summary,
		NetworkNames:      networkNames,
		ForkDeploymentIDs: forkDeploymentIDs,
	}, nil
}

// computeForkDeploymentIDs determines which deployments were added during fork mode.
// Returns nil if no fork is active for the current network.
func (uc *ListDeployments) computeForkDeploymentIDs(ctx context.Context) map[string]bool {
	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil
	}

	// Determine network name
	networkName := ""
	if uc.config.Network != nil {
		networkName = uc.config.Network.Name
	}
	if networkName == "" {
		return nil
	}

	if !state.IsForkActive(networkName) {
		return nil
	}

	// Load current deployment IDs
	currentPath := filepath.Join(uc.config.DataDir, "deployments.json")
	currentIDs := loadDeploymentIDs(currentPath)

	// Load initial backup deployment IDs (snapshot 0)
	backupPath := filepath.Join(uc.config.DataDir, "priv", "fork", networkName, "snapshots", "0", "deployments.json")
	backupIDs := loadDeploymentIDs(backupPath)

	// Compute fork-added IDs (in current but not in backup)
	forkIDs := make(map[string]bool)
	for id := range currentIDs {
		if !backupIDs[id] {
			forkIDs[id] = true
		}
	}

	return forkIDs
}

// filterByFork filters deployments based on fork status
func filterByFork(deployments []*models.Deployment, forkIDs map[string]bool, forkOnly bool) []*models.Deployment {
	filtered := make([]*models.Deployment, 0, len(deployments))
	for _, dep := range deployments {
		isFork := forkIDs[dep.ID]
		if forkOnly && isFork {
			filtered = append(filtered, dep)
		} else if !forkOnly && !isFork {
			filtered = append(filtered, dep)
		}
	}
	return filtered
}

// sortDeployments sorts deployments by namespace, chain, contract name, and label
func sortDeployments(deployments []*models.Deployment) {
	sort.Slice(deployments, func(i, j int) bool {
		// Sort by namespace
		if deployments[i].Namespace != deployments[j].Namespace {
			return deployments[i].Namespace < deployments[j].Namespace
		}

		// Then by chain ID
		if deployments[i].ChainID != deployments[j].ChainID {
			return deployments[i].ChainID < deployments[j].ChainID
		}

		// Then by contract name
		if deployments[i].ContractName != deployments[j].ContractName {
			return deployments[i].ContractName < deployments[j].ContractName
		}

		// Finally by label
		return deployments[i].Label < deployments[j].Label
	})
}

// calculateSummary calculates summary statistics for deployments
func calculateSummary(deployments []*models.Deployment) DeploymentSummary {
	summary := DeploymentSummary{
		Total:       len(deployments),
		ByNamespace: make(map[string]int),
		ByChain:     make(map[uint64]int),
		ByType:      make(map[models.DeploymentType]int),
	}

	for _, dep := range deployments {
		// Count by namespace
		summary.ByNamespace[dep.Namespace]++

		// Count by chain
		summary.ByChain[dep.ChainID]++

		// Count by type
		summary.ByType[dep.Type]++
	}

	return summary
}

// findNetworkName attempts to find a network name for a chain ID
func (uc *ListDeployments) findNetworkName(ctx context.Context, chainID uint64) string {
	// Get all available network names
	networkNames := uc.networkResolver.GetNetworks(ctx)

	// Try to resolve each network to find matching chain ID
	for _, name := range networkNames {
		network, err := uc.networkResolver.ResolveNetwork(ctx, name)
		if err != nil {
			continue
		}

		if network.ChainID == chainID {
			return name
		}
	}

	// Return empty string if no network found
	return ""
}

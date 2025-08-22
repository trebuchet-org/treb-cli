package usecase

import (
	"context"
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
}

// ListDeployments is the use case for listing deployments
type ListDeployments struct {
	config *config.RuntimeConfig
	repo   DeploymentRepository
}

// NewListDeployments creates a new ListDeployments use case
func NewListDeployments(cfg *config.RuntimeConfig, repo DeploymentRepository) *ListDeployments {
	return &ListDeployments{
		config: cfg,
		repo:   repo,
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

	// Sort deployments for consistent output
	sortDeployments(deployments)

	// Calculate summary
	summary := calculateSummary(deployments)

	return &DeploymentListResult{
		Deployments: deployments,
		Summary:     summary,
	}, nil
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

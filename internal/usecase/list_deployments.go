package usecase

import (
	"context"
	"sort"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ListDeploymentsParams contains parameters for listing deployments
type ListDeploymentsParams struct {
	// Filter parameters (namespace and chainID come from RuntimeConfig)
	ContractName string
	Label        string
	Type         domain.DeploymentType
}

// ListDeployments is the use case for listing deployments
type ListDeployments struct {
	config *config.RuntimeConfig
	store  DeploymentStore
	sink   ProgressSink
}

// NewListDeployments creates a new ListDeployments use case
func NewListDeployments(cfg *config.RuntimeConfig, store DeploymentStore, sink ProgressSink) *ListDeployments {
	return &ListDeployments{
		config: cfg,
		store:  store,
		sink:   sink,
	}
}

// Run executes the list deployments use case
func (uc *ListDeployments) Run(ctx context.Context, params ListDeploymentsParams) (*DeploymentListResult, error) {
	// Report progress
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "loading",
		Message: "Loading deployments from registry",
		Spinner: true,
	})

	// Create filter from params and runtime config
	filter := DeploymentFilter{
		Namespace:    uc.config.Namespace,
		ContractName: params.ContractName,
		Label:        params.Label,
		Type:         params.Type,
		ChainID:      uc.config.Network.ChainID,
	}

	// Get deployments from store
	deployments, err := uc.store.ListDeployments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Sort deployments for consistent output
	sortDeployments(deployments)

	// Calculate summary
	summary := calculateSummary(deployments)

	// Report completion
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "complete",
		Current: len(deployments),
		Total:   len(deployments),
		Message: "Deployments loaded",
	})

	return &DeploymentListResult{
		Deployments: deployments,
		Summary:     summary,
	}, nil
}

// sortDeployments sorts deployments by namespace, chain, contract name, and label
func sortDeployments(deployments []*domain.Deployment) {
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
func calculateSummary(deployments []*domain.Deployment) DeploymentSummary {
	summary := DeploymentSummary{
		Total:       len(deployments),
		ByNamespace: make(map[string]int),
		ByChain:     make(map[uint64]int),
		ByType:      make(map[domain.DeploymentType]int),
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

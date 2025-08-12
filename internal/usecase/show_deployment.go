package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ShowDeploymentParams contains parameters for showing a deployment
type ShowDeploymentParams struct {
	// Deployment reference (can be ID, address, contract name, etc.)
	DeploymentRef string
	
	// Optional filters for resolution
	ChainID   uint64
	Namespace string
	
	// Optional: resolve proxy implementation
	ResolveProxy bool
}

// ShowDeployment is the use case for showing deployment details
type ShowDeployment struct {
	store DeploymentStore
	sink  ProgressSink
}

// NewShowDeployment creates a new ShowDeployment use case
func NewShowDeployment(store DeploymentStore, sink ProgressSink) *ShowDeployment {
	return &ShowDeployment{
		store: store,
		sink:  sink,
	}
}

// Run executes the show deployment use case
func (uc *ShowDeployment) Run(ctx context.Context, params ShowDeploymentParams) (*domain.Deployment, error) {
	// Report progress
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "loading",
		Message: "Loading deployment details",
		Spinner: true,
	})
	
	deployment, err := uc.resolveDeployment(ctx, params)
	if err != nil {
		return nil, err
	}
	
	// If it's a proxy and we should resolve implementation
	if params.ResolveProxy && deployment.Type == domain.ProxyDeployment && deployment.ProxyInfo != nil {
		uc.sink.OnProgress(ctx, ProgressEvent{
			Stage:   "resolving",
			Message: "Resolving proxy implementation",
			Spinner: true,
		})
		
		// Try to load the implementation deployment
		if deployment.ProxyInfo.Implementation != "" {
			// First try by address
			impl, err := uc.store.GetDeploymentByAddress(ctx, deployment.ChainID, deployment.ProxyInfo.Implementation)
			if err == nil {
				deployment.Implementation = impl
			}
			// If not found by address, it's OK - the implementation might not be tracked
		}
	}
	
	// Report completion
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "complete",
		Message: "Deployment loaded",
	})
	
	return deployment, nil
}

// resolveDeployment resolves a deployment reference to a deployment
func (uc *ShowDeployment) resolveDeployment(ctx context.Context, params ShowDeploymentParams) (*domain.Deployment, error) {
	ref := params.DeploymentRef
	
	// 1. Try as deployment ID
	deployment, err := uc.store.GetDeployment(ctx, ref)
	if err == nil {
		return deployment, nil
	}
	
	// 2. Try as address (starts with 0x and is 42 chars)
	if strings.HasPrefix(ref, "0x") && len(ref) == 42 {
		if params.ChainID != 0 {
			// If chain ID is specified, try direct lookup
			deployment, err = uc.store.GetDeploymentByAddress(ctx, params.ChainID, ref)
			if err == nil {
				return deployment, nil
			}
		} else {
			// Search all deployments for this address
			deployments, err := uc.store.ListDeployments(ctx, DeploymentFilter{})
			if err != nil {
				return nil, fmt.Errorf("failed to list deployments: %w", err)
			}
			
			for _, dep := range deployments {
				if strings.EqualFold(dep.Address, ref) {
					return dep, nil
				}
			}
		}
	}
	
	// 3. Try as contract name (with optional label)
	parts := strings.Split(ref, ":")
	contractName := parts[0]
	var label string
	if len(parts) > 1 {
		label = parts[1]
	}
	
	// 4. Check if it contains namespace or chain ID prefixes
	// Format: namespace/contract or chainID/contract
	if strings.Contains(contractName, "/") {
		prefixParts := strings.SplitN(contractName, "/", 2)
		prefix := prefixParts[0]
		contractName = prefixParts[1]
		
		// Try to parse as chain ID
		if chainID := parseChainID(prefix); chainID != 0 {
			params.ChainID = chainID
		} else {
			// Otherwise treat as namespace
			params.Namespace = prefix
		}
	}
	
	// Apply filters
	filter := DeploymentFilter{
		ContractName: contractName,
		Label:        label,
		ChainID:      params.ChainID,
		Namespace:    params.Namespace,
	}
	
	deployments, err := uc.store.ListDeployments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	
	if len(deployments) == 0 {
		return nil, fmt.Errorf("no deployments found matching '%s'", ref)
	}
	
	// If multiple deployments found, return an error
	// In the future, this could support interactive mode with a picker
	if len(deployments) > 1 {
		return nil, fmt.Errorf("multiple deployments found matching '%s', please specify a unique identifier", ref)
	}
	
	return deployments[0], nil
}

// parseChainID tries to parse a string as a chain ID
func parseChainID(s string) uint64 {
	var chainID uint64
	_, err := fmt.Sscanf(s, "%d", &chainID)
	if err != nil {
		return 0
	}
	return chainID
}
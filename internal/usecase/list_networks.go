package usecase

import (
	"context"
)

// ListNetworksParams contains parameters for listing networks
type ListNetworksParams struct {
	// Currently no parameters, but we keep the struct for future extensibility
}

// ListNetworksResult contains the result of listing networks
type ListNetworksResult struct {
	Networks []NetworkStatus
}

// NetworkStatus represents the status of a network
type NetworkStatus struct {
	Name    string
	ChainID uint64
	Error   error
}

// ListNetworks is a use case for listing available networks
type ListNetworks struct {
	resolver NetworkResolver
}

// NewListNetworks creates a new ListNetworks use case
func NewListNetworks(resolver NetworkResolver) *ListNetworks {
	return &ListNetworks{
		resolver: resolver,
	}
}

// Run executes the use case
func (uc *ListNetworks) Run(ctx context.Context, params ListNetworksParams) (*ListNetworksResult, error) {
	// Get all configured networks
	networkNames := uc.resolver.GetNetworks(ctx)

	// Check each network's status
	networks := make([]NetworkStatus, 0, len(networkNames))
	for _, name := range networkNames {
		status := NetworkStatus{
			Name: name,
		}

		// Try to resolve network to get chain ID
		info, err := uc.resolver.ResolveNetwork(ctx, name)
		if err != nil {
			status.Error = err
		} else {
			status.ChainID = info.ChainID
		}

		networks = append(networks, status)
	}

	return &ListNetworksResult{
		Networks: networks,
	}, nil
}

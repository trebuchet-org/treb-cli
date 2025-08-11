package config

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NetworkResolverAdapter wraps the existing network.Resolver to implement NetworkResolver
type NetworkResolverAdapter struct {
	resolver *network.Resolver
}

// NewNetworkResolverAdapter creates a new adapter wrapping the existing network resolver
func NewNetworkResolverAdapter(projectRoot string) (*NetworkResolverAdapter, error) {
	resolver, err := network.NewResolver(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create network resolver: %w", err)
	}
	
	return &NetworkResolverAdapter{resolver: resolver}, nil
}

// ResolveNetwork resolves a network name to network information
func (n *NetworkResolverAdapter) ResolveNetwork(ctx context.Context, name string) (*domain.NetworkInfo, error) {
	info, err := n.resolver.ResolveNetwork(name)
	if err != nil {
		return nil, err
	}
	
	// Get explorer URL if available
	explorerURL, _ := n.resolver.GetExplorerURL(name)
	
	return &domain.NetworkInfo{
		ChainID:     info.ChainID,
		Name:        info.Name,
		RPCURL:      info.RpcUrl,
		ExplorerURL: explorerURL,
	}, nil
}

// GetPreferredNetwork returns the preferred network name for a chain ID
func (n *NetworkResolverAdapter) GetPreferredNetwork(ctx context.Context, chainID uint64) (string, error) {
	return n.resolver.GetPreferredNetwork(chainID)
}

// ListNetworks returns all configured networks
func (n *NetworkResolverAdapter) ListNetworks(ctx context.Context) ([]*domain.NetworkInfo, error) {
	// Get all network names
	networkNames := n.resolver.GetNetworks()
	
	// Resolve each network to get full info
	var networks []*domain.NetworkInfo
	for _, name := range networkNames {
		info, err := n.ResolveNetwork(ctx, name)
		if err != nil {
			// Skip networks that can't be resolved
			continue
		}
		networks = append(networks, info)
	}
	
	return networks, nil
}

// Ensure the adapter implements the interface
var _ usecase.NetworkResolver = (*NetworkResolverAdapter)(nil)
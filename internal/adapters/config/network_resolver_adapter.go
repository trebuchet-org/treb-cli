package config

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NetworkResolverAdapter adapts the config.NetworkResolver to the usecase.NetworkResolver interface
type NetworkResolverAdapter struct {
	resolver *config.NetworkResolver
}

// NewNetworkResolverAdapter creates a new adapter
func NewNetworkResolverAdapter(resolver *config.NetworkResolver) *NetworkResolverAdapter {
	return &NetworkResolverAdapter{
		resolver: resolver,
	}
}

// GetNetworks returns all configured network names
func (a *NetworkResolverAdapter) GetNetworks(ctx context.Context) []string {
	// The underlying resolver doesn't use context, but we accept it for interface compatibility
	return a.resolver.GetNetworks()
}

// ResolveNetwork resolves a network name to its configuration
func (a *NetworkResolverAdapter) ResolveNetwork(ctx context.Context, networkName string) (*domain.Network, error) {
	// Call the underlying resolver
	return a.resolver.Resolve(networkName)
}

// Ensure the adapter implements the interface
var _ usecase.NetworkResolver = (*NetworkResolverAdapter)(nil)


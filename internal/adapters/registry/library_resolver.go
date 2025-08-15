package registry

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// LibraryResolverAdapter adapts the internal library resolver
type LibraryResolverAdapter struct {
	resolver *InternalLibraryResolver
}

// NewLibraryResolverAdapter creates a new library resolver adapter
func NewLibraryResolverAdapter(deploymentStore usecase.DeploymentStore) *LibraryResolverAdapter {
	return &LibraryResolverAdapter{
		resolver: NewInternalLibraryResolver(deploymentStore),
	}
}

// GetDeployedLibraries gets all deployed libraries for the given context
func (a *LibraryResolverAdapter) GetDeployedLibraries(
	ctx context.Context,
	namespace string,
	chainID uint64,
) ([]usecase.LibraryReference, error) {
	return a.resolver.GetDeployedLibraries(ctx, namespace, chainID)
}
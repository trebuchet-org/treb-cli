package registry

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// LibraryResolverAdapter adapts the registry to resolve deployed libraries
type LibraryResolverAdapter struct {
	manager *registry.Manager
}

// NewLibraryResolverAdapter creates a new library resolver adapter
func NewLibraryResolverAdapter(projectPath string) (*LibraryResolverAdapter, error) {
	manager, err := registry.NewManager(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}

	return &LibraryResolverAdapter{
		manager: manager,
	}, nil
}

// GetDeployedLibraries gets all deployed libraries for the given context
func (a *LibraryResolverAdapter) GetDeployedLibraries(
	ctx context.Context,
	namespace string,
	chainID uint64,
) ([]usecase.LibraryReference, error) {
	// Get all deployments for the namespace
	deployments, err := a.manager.GetDeploymentsByNamespace(namespace, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to load deployments: %w", err)
	}

	var libraries []usecase.LibraryReference

	// Filter for library deployments
	for _, deployment := range deployments {
		// Check if this is a library deployment
		if deployment.Type == types.LibraryDeployment {
			// Format the library reference
			if deployment.Artifact.Path != "" {
				lib := usecase.LibraryReference{
					Path:    deployment.Artifact.Path,
					Name:    deployment.ContractName,
					Address: deployment.Address,
				}
				libraries = append(libraries, lib)
			}
		}
	}

	return libraries, nil
}
package resolvers

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// LibraryResolver resolves deployed libraries without pkg dependencies
type LibraryResolver struct {
	deploymentRepo usecase.DeploymentRepository
}

// NewLibraryResolver creates a new internal library resolver
func NewLibraryResolver(deploymentRepo usecase.DeploymentRepository) *LibraryResolver {
	return &LibraryResolver{
		deploymentRepo: deploymentRepo,
	}
}

// GetDeployedLibraries gets all deployed libraries for the given context
func (r *LibraryResolver) GetDeployedLibraries(
	ctx context.Context,
	namespace string,
	chainID uint64,
) ([]usecase.LibraryReference, error) {
	// Get all deployments for the namespace
	filter := domain.DeploymentFilter{
		Namespace: namespace,
		ChainID:   chainID,
	}
	deployments, err := r.deploymentRepo.ListDeployments(ctx, filter)
	if err != nil {
		return nil, err
	}

	var libraries []usecase.LibraryReference

	// Filter for library deployments
	for _, deployment := range deployments {
		// Check if this is a library deployment
		if deployment.Type == models.LibraryDeployment {
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

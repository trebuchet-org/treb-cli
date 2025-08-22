package usecase

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// ShowDeploymentParams contains parameters for showing a deployment
type ShowDeploymentParams struct {
	// Deployment reference (can be ID, address, contract name, etc.)
	DeploymentRef string

	// Optional: resolve proxy implementation
	ResolveProxy bool
}

// ShowDeployment is the use case for showing deployment details
type ShowDeployment struct {
	config   *config.RuntimeConfig
	repo     DeploymentRepository
	resolver DeploymentResolver
}

// NewShowDeployment creates a new ShowDeployment use case
func NewShowDeployment(
	cfg *config.RuntimeConfig,
	repo DeploymentRepository,
	resolver DeploymentResolver,
) *ShowDeployment {
	return &ShowDeployment{
		config:   cfg,
		repo:     repo,
		resolver: resolver,
	}
}

// Run executes the show deployment use case
func (uc *ShowDeployment) Run(ctx context.Context, params ShowDeploymentParams) (*models.Deployment, error) {
	// Use the deployment resolver
	query := domain.DeploymentQuery{
		Reference: params.DeploymentRef,
		ChainID:   0,  // Will use runtime config
		Namespace: "", // Will use runtime config
	}
	deployment, err := uc.resolver.ResolveDeployment(ctx, query)
	if err != nil {
		return nil, err
	}

	// If it's a proxy and we should resolve implementation
	if params.ResolveProxy && deployment.Type == models.ProxyDeployment && deployment.ProxyInfo != nil {
		// Try to load the implementation deployment
		if deployment.ProxyInfo.Implementation != "" {
			// First try by address
			impl, err := uc.repo.GetDeploymentByAddress(ctx, deployment.ChainID, deployment.ProxyInfo.Implementation)
			if err == nil {
				deployment.Implementation = impl
			}
			// If not found by address, it's OK - the implementation might not be tracked
		}
	}

	return deployment, nil
}

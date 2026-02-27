package usecase

import (
	"context"
	"fmt"
	"path/filepath"

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

	// NoFork skips fork-added deployments (only resolves from pre-fork entries)
	NoFork bool
}

// ShowDeploymentResult contains the result of showing a deployment
type ShowDeploymentResult struct {
	Deployment       *models.Deployment
	IsForkDeployment bool
}

// ShowDeployment is the use case for showing deployment details
type ShowDeployment struct {
	config    *config.RuntimeConfig
	repo      DeploymentRepository
	resolver  DeploymentResolver
	forkState ForkStateStore
}

// NewShowDeployment creates a new ShowDeployment use case
func NewShowDeployment(
	cfg *config.RuntimeConfig,
	repo DeploymentRepository,
	resolver DeploymentResolver,
	forkState ForkStateStore,
) *ShowDeployment {
	return &ShowDeployment{
		config:    cfg,
		repo:      repo,
		resolver:  resolver,
		forkState: forkState,
	}
}

// Run executes the show deployment use case
func (uc *ShowDeployment) Run(ctx context.Context, params ShowDeploymentParams) (*ShowDeploymentResult, error) {
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

	// Check fork status
	isFork := uc.isForkDeployment(ctx, deployment.ID)

	// If --no-fork is set and this is a fork-added deployment, skip it
	if params.NoFork && isFork {
		return nil, fmt.Errorf("deployment %q was added during fork mode (use without --no-fork to view)", params.DeploymentRef)
	}

	return &ShowDeploymentResult{
		Deployment:       deployment,
		IsForkDeployment: isFork,
	}, nil
}

// isForkDeployment checks if a deployment ID was added during fork mode.
func (uc *ShowDeployment) isForkDeployment(ctx context.Context, deploymentID string) bool {
	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return false
	}

	networkName := ""
	if uc.config.Network != nil {
		networkName = uc.config.Network.Name
	}
	if networkName == "" {
		return false
	}

	if !state.IsForkActive(networkName) {
		return false
	}

	// Load initial backup deployment IDs (snapshot 0)
	backupPath := filepath.Join(uc.config.DataDir, "priv", "fork", networkName, "snapshots", "0", "deployments.json")
	backupIDs := loadDeploymentIDs(backupPath)

	// If the deployment ID is NOT in the backup, it was added during fork mode
	return !backupIDs[deploymentID]
}

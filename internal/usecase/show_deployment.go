package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ShowDeploymentParams contains parameters for showing a deployment
type ShowDeploymentParams struct {
	// Either ID or ChainID+Address can be used
	ID      string
	ChainID uint64
	Address string
	
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
	
	var deployment *domain.Deployment
	var err error
	
	// Get deployment by ID or by chain+address
	if params.ID != "" {
		deployment, err = uc.store.GetDeployment(ctx, params.ID)
	} else if params.ChainID != 0 && params.Address != "" {
		deployment, err = uc.store.GetDeploymentByAddress(ctx, params.ChainID, params.Address)
	} else {
		return nil, fmt.Errorf("either deployment ID or chain ID + address must be provided")
	}
	
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
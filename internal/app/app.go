package app

import (
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// App is the main application container that holds all use cases
type App struct {
	// Configuration
	Config *config.RuntimeConfig
	
	// Use cases
	ListDeployments         *usecase.ListDeployments
	ShowDeployment          *usecase.ShowDeployment
	GenerateDeploymentScript *usecase.GenerateDeploymentScript
	ListNetworks            *usecase.ListNetworks
	PruneRegistry           *usecase.PruneRegistry
	
	// Add more use cases as they are implemented
	// RunScript       *usecase.RunScript
	// VerifyContract  *usecase.VerifyContract
	// InitProject     *usecase.InitProject
}

// NewApp creates a new application instance with all use cases
func NewApp(
	cfg *config.RuntimeConfig,
	listDeployments *usecase.ListDeployments,
	showDeployment *usecase.ShowDeployment,
	generateDeploymentScript *usecase.GenerateDeploymentScript,
	listNetworks *usecase.ListNetworks,
	pruneRegistry *usecase.PruneRegistry,
) (*App, error) {
	return &App{
		Config:                   cfg,
		ListDeployments:          listDeployments,
		ShowDeployment:           showDeployment,
		GenerateDeploymentScript: generateDeploymentScript,
		ListNetworks:             listNetworks,
		PruneRegistry:            pruneRegistry,
	}, nil
}
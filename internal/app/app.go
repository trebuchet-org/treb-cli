package app

import (
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// App is the main application container that holds all use cases
type App struct {
	// Configuration
	Config *config.RuntimeConfig

	// Shared dependencies
	Selector usecase.DeploymentSelector

	// Use cases
	ListDeployments          *usecase.ListDeployments
	ShowDeployment           *usecase.ShowDeployment
	GenerateDeploymentScript *usecase.GenerateDeploymentScript
	ListNetworks             *usecase.ListNetworks
	PruneRegistry            *usecase.PruneRegistry
	ShowConfig               *usecase.ShowConfig
	SetConfig                *usecase.SetConfig
	RemoveConfig             *usecase.RemoveConfig
	RunScript                *usecase.RunScript
	VerifyDeployment         *usecase.VerifyDeployment
	ComposeDeployment        *usecase.ComposeDeployment
	SyncRegistry             *usecase.SyncRegistry
	TagDeployment            *usecase.TagDeployment
	ManageAnvil              *usecase.ManageAnvil
	InitProject              *usecase.InitProject

	// Adapters (needed for special cases like log streaming)
	AnvilManager usecase.AnvilManager

	// Renderers
	GenerateRenderer render.Renderer[*usecase.GenerateScriptResult]
	ScriptRenderer   *render.ScriptRenderer
	ComposeRenderer  *render.ComposeRenderer
}

// NewApp creates a new application instance with all use cases
func NewApp(
	cfg *config.RuntimeConfig,
	selector usecase.DeploymentSelector,
	listDeployments *usecase.ListDeployments,
	showDeployment *usecase.ShowDeployment,
	generateDeploymentScript *usecase.GenerateDeploymentScript,
	listNetworks *usecase.ListNetworks,
	pruneRegistry *usecase.PruneRegistry,
	showConfig *usecase.ShowConfig,
	setConfig *usecase.SetConfig,
	removeConfig *usecase.RemoveConfig,
	runScript *usecase.RunScript,
	verifyDeployment *usecase.VerifyDeployment,
	composeDeployment *usecase.ComposeDeployment,
	syncRegistry *usecase.SyncRegistry,
	tagDeployment *usecase.TagDeployment,
	manageAnvil *usecase.ManageAnvil,
	initProject *usecase.InitProject,
	anvilManager usecase.AnvilManager,
	generateRenderer render.Renderer[*usecase.GenerateScriptResult],
	scriptRenderer *render.ScriptRenderer,
	composeRenderer *render.ComposeRenderer,
) (*App, error) {
	return &App{
		Config:                   cfg,
		Selector:                 selector,
		ListDeployments:          listDeployments,
		ShowDeployment:           showDeployment,
		GenerateDeploymentScript: generateDeploymentScript,
		ListNetworks:             listNetworks,
		PruneRegistry:            pruneRegistry,
		ShowConfig:               showConfig,
		SetConfig:                setConfig,
		RemoveConfig:             removeConfig,
		RunScript:                runScript,
		VerifyDeployment:         verifyDeployment,
		ComposeDeployment:        composeDeployment,
		SyncRegistry:             syncRegistry,
		TagDeployment:            tagDeployment,
		ManageAnvil:              manageAnvil,
		InitProject:              initProject,
		AnvilManager:             anvilManager,
		GenerateRenderer:         generateRenderer,
		ScriptRenderer:           scriptRenderer,
		ComposeRenderer:          composeRenderer,
	}, nil
}

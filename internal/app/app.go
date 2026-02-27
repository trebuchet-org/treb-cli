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
	ResetRegistry            *usecase.ResetRegistry
	ShowConfig               *usecase.ShowConfig
	SetConfig                *usecase.SetConfig
	RemoveConfig             *usecase.RemoveConfig
	RunScript                *usecase.RunScript
	VerifyDeployment         *usecase.VerifyDeployment
	ComposeDeployment        *usecase.ComposeDeployment
	SyncRegistry             *usecase.SyncRegistry
	TagDeployment            *usecase.TagDeployment
	RegisterDeployment       *usecase.RegisterDeployment
	ManageAnvil              *usecase.ManageAnvil
	InitProject              *usecase.InitProject

	// Fork use cases
	EnterFork  *usecase.EnterFork
	ExitFork   *usecase.ExitFork
	RevertFork *usecase.RevertFork

	// Adapters (needed for special cases like log streaming)
	AnvilManager    usecase.AnvilManager
	NetworkResolver usecase.NetworkResolver

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
	resetRegistry *usecase.ResetRegistry,
	showConfig *usecase.ShowConfig,
	setConfig *usecase.SetConfig,
	removeConfig *usecase.RemoveConfig,
	runScript *usecase.RunScript,
	verifyDeployment *usecase.VerifyDeployment,
	composeDeployment *usecase.ComposeDeployment,
	syncRegistry *usecase.SyncRegistry,
	tagDeployment *usecase.TagDeployment,
	registerDeployment *usecase.RegisterDeployment,
	manageAnvil *usecase.ManageAnvil,
	initProject *usecase.InitProject,
	enterFork *usecase.EnterFork,
	exitFork *usecase.ExitFork,
	revertFork *usecase.RevertFork,
	anvilManager usecase.AnvilManager,
	networkResolver usecase.NetworkResolver,
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
		ResetRegistry:            resetRegistry,
		ShowConfig:               showConfig,
		SetConfig:                setConfig,
		RemoveConfig:             removeConfig,
		RunScript:                runScript,
		VerifyDeployment:         verifyDeployment,
		ComposeDeployment:        composeDeployment,
		SyncRegistry:             syncRegistry,
		TagDeployment:            tagDeployment,
		RegisterDeployment:       registerDeployment,
		ManageAnvil:              manageAnvil,
		InitProject:              initProject,
		EnterFork:                enterFork,
		ExitFork:                 exitFork,
		RevertFork:               revertFork,
		AnvilManager:             anvilManager,
		NetworkResolver:          networkResolver,
		GenerateRenderer:         generateRenderer,
		ScriptRenderer:           scriptRenderer,
		ComposeRenderer:          composeRenderer,
	}, nil
}

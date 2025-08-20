//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/spf13/viper"
	"github.com/trebuchet-org/treb-cli/internal/adapters"
	"github.com/trebuchet-org/treb-cli/internal/cli/interactive"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ProvideDeploymentSelector provides DeploymentSelector interface from SelectorAdapter
func ProvideDeploymentSelector(adapter *interactive.SelectorAdapter) usecase.DeploymentSelector {
	return adapter
}

// InitApp creates a fully wired App instance with viper configuration
func InitApp(v *viper.Viper, sink usecase.ProgressSink) (*App, error) {
	wire.Build(
		// Configuration
		config.Provider,
		config.ProvideNetworkResolver,
		wire.Bind(new(usecase.NetworkResolver), new(*config.NetworkResolver)),

		config.NewSendersManager,
		wire.Bind(new(usecase.SendersManager), new(*config.SendersManager)),

		// Adapters - now receive RuntimeConfig
		adapters.AllAdapters,
		render.NewGenerateRenderer,

		// Use cases
		usecase.NewListDeployments,
		usecase.NewShowDeployment,
		usecase.NewGenerateDeploymentScript,
		usecase.NewListNetworks,
		usecase.NewPruneRegistry,
		usecase.NewShowConfig,
		usecase.NewSetConfig,
		usecase.NewRemoveConfig,
		usecase.NewRunScript,
		usecase.NewVerifyDeployment,
		usecase.NewOrchestrateDeployment,
		usecase.NewSyncRegistry,
		usecase.NewTagDeployment,
		usecase.NewManageAnvil,
		usecase.NewInitProject,

		// Interface providers
		ProvideDeploymentSelector,

		// App
		NewApp,
	)
	return nil, nil
}

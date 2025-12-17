//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/trebuchet-org/treb-cli/internal/adapters"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/logging"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// InitApp creates a fully wired App instance with viper configuration
func InitApp(v *viper.Viper, cmd *cobra.Command) (*App, error) {
	wire.Build(
		// Configuration
		config.Provider,
		config.ProvideNetworkResolver,
		wire.Bind(new(usecase.NetworkResolver), new(*config.NetworkResolver)),

		config.NewSendersManager,
		wire.Bind(new(usecase.SendersManager), new(*config.SendersManager)),

		render.ProvideIO,

		// Logging
		logging.LoggingSet,

		// Adapters - now receive RuntimeConfig
		adapters.AllAdapters,

		// Renderers
		render.NewScriptRenderer,
		render.NewGenerateRenderer,
		render.NewComposeRenderer,

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
		usecase.NewComposeDeployment,
		usecase.NewSyncRegistry,
		usecase.NewTagDeployment,
		usecase.NewRegisterDeployment,
		usecase.NewManageAnvil,
		usecase.NewInitProject,

		// App
		NewApp,
	)
	return nil, nil
}

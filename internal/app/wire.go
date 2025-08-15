//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/spf13/viper"
	"github.com/trebuchet-org/treb-cli/internal/adapters"
	"github.com/trebuchet-org/treb-cli/internal/adapters/interactive"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ProvideContractResolver provides ContractResolver interface from ResolveContract
func ProvideContractResolver(uc *usecase.ResolveContract) usecase.ContractResolver {
	return uc
}

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
		
		// Adapters - now receive RuntimeConfig
		adapters.AllAdapters,
		
		// Use cases
		usecase.NewListDeployments,
		usecase.NewShowDeployment,
		usecase.NewResolveContract,
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
		ProvideContractResolver,
		ProvideDeploymentSelector,
		
		// App
		NewApp,
	)
	return nil, nil
}
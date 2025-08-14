//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/spf13/viper"
	"github.com/trebuchet-org/treb-cli/internal/adapters"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ProvideContractResolver provides ContractResolver interface from ResolveContract
func ProvideContractResolver(uc *usecase.ResolveContract) usecase.ContractResolver {
	return uc
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
		
		// Interface providers
		ProvideContractResolver,
		
		// App
		NewApp,
	)
	return nil, nil
}
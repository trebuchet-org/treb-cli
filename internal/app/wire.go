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

// InitApp creates a fully wired App instance with viper configuration
func InitApp(v *viper.Viper, sink usecase.ProgressSink) (*App, error) {
	wire.Build(
		// Configuration
		config.Provider,
		
		// Adapters - now receive RuntimeConfig
		adapters.AllAdapters,
		
		// Use cases
		usecase.NewListDeployments,
		usecase.NewShowDeployment,
		
		// App
		NewApp,
	)
	return nil, nil
}
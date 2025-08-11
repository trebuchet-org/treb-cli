//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/adapters"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// InitApp creates a fully wired App instance
func InitApp(cfg Config, sink usecase.ProgressSink) (*App, error) {
	wire.Build(
		// Extract fields from config
		wire.FieldsOf(new(Config), "ProjectRoot"),
		
		// Adapters
		adapters.AllAdapters,
		
		// Use cases
		usecase.NewListDeployments,
		usecase.NewShowDeployment,
		
		// App
		NewApp,
	)
	return nil, nil
}
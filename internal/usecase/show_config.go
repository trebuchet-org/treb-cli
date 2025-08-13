package usecase

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ShowConfigResult contains the result of showing configuration
type ShowConfigResult struct {
	Config     *domain.LocalConfig
	ConfigPath string
	Exists     bool
}

// ShowConfig is a use case for showing configuration
type ShowConfig struct {
	store LocalConfigStore
}

// NewShowConfig creates a new ShowConfig use case
func NewShowConfig(store LocalConfigStore) *ShowConfig {
	return &ShowConfig{
		store: store,
	}
}

// Run executes the show config use case
func (uc *ShowConfig) Run(ctx context.Context) (*ShowConfigResult, error) {
	exists := uc.store.Exists()
	
	config, err := uc.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	return &ShowConfigResult{
		Config:     config,
		ConfigPath: uc.store.GetPath(),
		Exists:     exists,
	}, nil
}
package usecase

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// ShowConfigResult contains the result of showing configuration
type ShowConfigResult struct {
	Config     *config.LocalConfig
	ConfigPath string
	Exists     bool
}

// ShowConfig is a use case for showing configuration
type ShowConfig struct {
	repo LocalConfigRepository
}

// NewShowConfig creates a new ShowConfig use case
func NewShowConfig(repo LocalConfigRepository) *ShowConfig {
	return &ShowConfig{
		repo: repo,
	}
}

// Run executes the show config use case
func (uc *ShowConfig) Run(ctx context.Context) (*ShowConfigResult, error) {
	exists := uc.repo.Exists()

	config, err := uc.repo.Load(ctx)
	if err != nil {
		return nil, err
	}

	return &ShowConfigResult{
		Config:     config,
		ConfigPath: uc.repo.GetPath(),
		Exists:     exists,
	}, nil
}

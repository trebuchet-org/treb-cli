package usecase

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// SenderDisplayInfo represents a sender for display purposes
type SenderDisplayInfo struct {
	Name   string            // Role/sender name
	Type   config.SenderType // Sender type
	Detail string            // Key display detail (env var ref, address, path)
}

// ShowConfigResult contains the result of showing configuration
type ShowConfigResult struct {
	Config       *config.LocalConfig
	ConfigPath   string
	Exists       bool
	ConfigSource string // "treb.toml" or "foundry.toml"
	Senders      []SenderDisplayInfo
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

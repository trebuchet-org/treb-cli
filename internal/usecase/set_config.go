package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// SetConfigParams contains parameters for setting configuration
type SetConfigParams struct {
	Key   string
	Value string
}

// SetConfigResult contains the result of setting configuration
type SetConfigResult struct {
	UpdatedConfig *config.LocalConfig
	ConfigPath    string
	Key           config.ConfigKey
	Value         string
}

// SetConfig is a use case for setting configuration values
type SetConfig struct {
	repo LocalConfigRepository
}

// NewSetConfig creates a new SetConfig use case
func NewSetConfig(repo LocalConfigRepository) *SetConfig {
	return &SetConfig{
		repo: repo,
	}
}

// Run executes the set config use case
func (uc *SetConfig) Run(ctx context.Context, params SetConfigParams) (*SetConfigResult, error) {
	// Normalize key to lowercase
	key := strings.ToLower(params.Key)

	// Validate key
	if !config.IsValidConfigKey(key) {
		validKeys := []string{}
		for _, k := range config.ValidConfigKeys() {
			if k == config.ConfigKeyNamespace {
				validKeys = append(validKeys, string(k)+" (ns)")
			} else {
				validKeys = append(validKeys, string(k))
			}
		}
		return nil, fmt.Errorf("unknown config key: %s\nAvailable keys: %s", params.Key, strings.Join(validKeys, ", "))
	}

	// Normalize the key
	normalizedKey := config.NormalizeConfigKey(key)

	// Load existing config or create new one
	localConfig, err := uc.repo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Set the value based on key
	switch normalizedKey {
	case config.ConfigKeyNamespace:
		localConfig.Namespace = params.Value
	case config.ConfigKeyNetwork:
		localConfig.Network = params.Value
	}

	// Save the updated config
	if err := uc.repo.Save(ctx, localConfig); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &SetConfigResult{
		UpdatedConfig: localConfig,
		ConfigPath:    uc.repo.GetPath(),
		Key:           normalizedKey,
		Value:         params.Value,
	}, nil
}


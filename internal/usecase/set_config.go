package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// SetConfigParams contains parameters for setting configuration
type SetConfigParams struct {
	Key   string
	Value string
}

// SetConfigResult contains the result of setting configuration
type SetConfigResult struct {
	UpdatedConfig *domain.LocalConfig
	ConfigPath    string
	Key           domain.ConfigKey
	Value         string
}

// SetConfig is a use case for setting configuration values
type SetConfig struct {
	store LocalConfigStore
}

// NewSetConfig creates a new SetConfig use case
func NewSetConfig(store LocalConfigStore) *SetConfig {
	return &SetConfig{
		store: store,
	}
}

// Run executes the set config use case
func (uc *SetConfig) Run(ctx context.Context, params SetConfigParams) (*SetConfigResult, error) {
	// Normalize key to lowercase
	key := strings.ToLower(params.Key)
	
	// Validate key
	if !domain.IsValidConfigKey(key) {
		validKeys := []string{}
		for _, k := range domain.ValidConfigKeys() {
			if k == domain.ConfigKeyNamespace {
				validKeys = append(validKeys, string(k)+" (ns)")
			} else {
				validKeys = append(validKeys, string(k))
			}
		}
		return nil, fmt.Errorf("unknown config key: %s\nAvailable keys: %s", params.Key, strings.Join(validKeys, ", "))
	}

	// Normalize the key
	normalizedKey := domain.NormalizeConfigKey(key)

	// Load existing config or create new one
	config, err := uc.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Set the value based on key
	switch normalizedKey {
	case domain.ConfigKeyNamespace:
		config.Namespace = params.Value
	case domain.ConfigKeyNetwork:
		config.Network = params.Value
	}

	// Save the updated config
	if err := uc.store.Save(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &SetConfigResult{
		UpdatedConfig: config,
		ConfigPath:    uc.store.GetPath(),
		Key:           normalizedKey,
		Value:         params.Value,
	}, nil
}
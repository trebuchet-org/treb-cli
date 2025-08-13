package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// RemoveConfigParams contains parameters for removing configuration
type RemoveConfigParams struct {
	Key string
}

// RemoveConfigResult contains the result of removing configuration
type RemoveConfigResult struct {
	UpdatedConfig *domain.LocalConfig
	ConfigPath    string
	Key           domain.ConfigKey
	RemovedValue  string
}

// RemoveConfig is a use case for removing configuration values
type RemoveConfig struct {
	store LocalConfigStore
}

// NewRemoveConfig creates a new RemoveConfig use case
func NewRemoveConfig(store LocalConfigStore) *RemoveConfig {
	return &RemoveConfig{
		store: store,
	}
}

// Run executes the remove config use case
func (uc *RemoveConfig) Run(ctx context.Context, params RemoveConfigParams) (*RemoveConfigResult, error) {
	// Config file must exist to remove values
	if !uc.store.Exists() {
		// Get relative path for error message
		path := uc.store.GetPath()
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, path); err == nil {
				path = relPath
			}
		}
		return nil, fmt.Errorf("no config file found at %s", path)
	}

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

	// Load existing config
	config, err := uc.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Store the old value for the result
	var removedValue string

	// Remove the value based on key
	switch normalizedKey {
	case domain.ConfigKeyNamespace:
		removedValue = config.Namespace
		config.Namespace = "default"
	case domain.ConfigKeyNetwork:
		removedValue = config.Network
		config.Network = ""
	}

	// Save the updated config
	if err := uc.store.Save(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &RemoveConfigResult{
		UpdatedConfig: config,
		ConfigPath:    uc.store.GetPath(),
		Key:           normalizedKey,
		RemovedValue:  removedValue,
	}, nil
}
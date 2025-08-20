package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// RemoveConfigParams contains parameters for removing configuration
type RemoveConfigParams struct {
	Key string
}

// RemoveConfigResult contains the result of removing configuration
type RemoveConfigResult struct {
	UpdatedConfig *config.LocalConfig
	ConfigPath    string
	Key           config.ConfigKey
	RemovedValue  string
}

// RemoveConfig is a use case for removing configuration values
type RemoveConfig struct {
	repo LocalConfigRepository
}

// NewRemoveConfig creates a new RemoveConfig use case
func NewRemoveConfig(repo LocalConfigRepository) *RemoveConfig {
	return &RemoveConfig{
		repo: repo,
	}
}

// Run executes the remove config use case
func (uc *RemoveConfig) Run(ctx context.Context, params RemoveConfigParams) (*RemoveConfigResult, error) {
	// Config file must exist to remove values
	if !uc.repo.Exists() {
		// Get relative path for error message
		path := uc.repo.GetPath()
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

	// Load existing config
	localConfig, err := uc.repo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Store the old value for the result
	var removedValue string

	// Remove the value based on key
	switch normalizedKey {
	case config.ConfigKeyNamespace:
		removedValue = localConfig.Namespace
		localConfig.Namespace = "default"
	case config.ConfigKeyNetwork:
		removedValue = localConfig.Network
		localConfig.Network = ""
	}

	// Save the updated config
	if err := uc.repo.Save(ctx, localConfig); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &RemoveConfigResult{
		UpdatedConfig: localConfig,
		ConfigPath:    uc.repo.GetPath(),
		Key:           normalizedKey,
		RemovedValue:  removedValue,
	}, nil
}


package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// LocalConfigStoreAdapter implements LocalConfigStore using the file system
type LocalConfigStoreAdapter struct {
	configPath string
}

// NewLocalConfigStoreAdapter creates a new LocalConfigStoreAdapter
func NewLocalConfigStoreAdapter(cfg *config.RuntimeConfig) *LocalConfigStoreAdapter {
	return &LocalConfigStoreAdapter{
		configPath: filepath.Join(cfg.DataDir, "config.local.json"),
	}
}

// Exists checks if the config file exists
func (s *LocalConfigStoreAdapter) Exists() bool {
	_, err := os.Stat(s.configPath)
	return !os.IsNotExist(err)
}

// Load reads the configuration from the file
func (s *LocalConfigStoreAdapter) Load(ctx context.Context) (*domain.LocalConfig, error) {
	// If file doesn't exist, return default config
	if !s.Exists() {
		return domain.DefaultLocalConfig(), nil
	}

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config domain.LocalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Fill in any missing fields with defaults
	defaultCfg := domain.DefaultLocalConfig()
	if config.Namespace == "" {
		config.Namespace = defaultCfg.Namespace
	}

	return &config, nil
}

// Save writes the configuration to the file
func (s *LocalConfigStoreAdapter) Save(ctx context.Context, config *domain.LocalConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetPath returns the path to the config file
func (s *LocalConfigStoreAdapter) GetPath() string {
	return s.configPath
}

// Ensure LocalConfigStoreAdapter implements LocalConfigStore
var _ usecase.LocalConfigStore = (*LocalConfigStoreAdapter)(nil)
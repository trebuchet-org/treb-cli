package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the fdeploy configuration
type Config struct {
	Environment string `json:"environment"`
	Network     string `json:"network"`
	Verify      bool   `json:"verify"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Environment: "staging",
		Network:     "alfajores",
		Verify:      false,
	}
}

// Manager handles configuration file operations
type Manager struct {
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(projectRoot string) *Manager {
	return &Manager{
		configPath: filepath.Join(projectRoot, ".fdeploy"),
	}
}

// Load reads the configuration from the .fdeploy file
func (m *Manager) Load() (*Config, error) {
	// If file doesn't exist, return error
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no configuration file found at %s", m.configPath)
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Fill in any missing fields with defaults
	defaultCfg := DefaultConfig()
	if config.Environment == "" {
		config.Environment = defaultCfg.Environment
	}
	if config.Network == "" {
		config.Network = defaultCfg.Network
	}

	return &config, nil
}

// Save writes the configuration to the .fdeploy file
func (m *Manager) Save(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Set updates a specific configuration value
func (m *Manager) Set(key, value string) error {
	config, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "environment", "env":
		if value != "staging" && value != "prod" {
			return fmt.Errorf("invalid environment: %s (must be 'staging' or 'prod')", value)
		}
		config.Environment = value
	case "network":
		config.Network = value
	case "verify":
		switch value {
		case "true", "1", "yes", "on":
			config.Verify = true
		case "false", "0", "no", "off":
			config.Verify = false
		default:
			return fmt.Errorf("invalid verify value: %s (must be true/false)", value)
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return m.Save(config)
}

// Get retrieves a specific configuration value
func (m *Manager) Get(key string) (string, error) {
	config, err := m.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "environment", "env":
		return config.Environment, nil
	case "network":
		return config.Network, nil
	case "verify":
		if config.Verify {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// List returns all configuration values
func (m *Manager) List() (*Config, error) {
	return m.Load()
}

// Exists checks if the config file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.configPath)
	return !os.IsNotExist(err)
}

// GetPath returns the path to the config file
func (m *Manager) GetPath() string {
	return m.configPath
}
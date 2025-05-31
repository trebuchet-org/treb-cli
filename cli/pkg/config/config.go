package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the treb configuration
type Config struct {
	Namespace string `json:"namespace"`
	Network   string `json:"network"`
	Sender    string `json:"sender"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Namespace: "default",
		Network:   "",
		Sender:    "",
	}
}

// Manager handles configuration file operations
type Manager struct {
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(projectRoot string) *Manager {
	return &Manager{
		configPath: filepath.Join(projectRoot, ".treb/config.local.json"),
	}
}

// Load reads the configuration from the .treb file
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
	if config.Namespace == "" {
		config.Namespace = defaultCfg.Namespace
	}

	return &config, nil
}

// Save writes the configuration to the .treb file
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
	case "namespace", "ns":
		config.Namespace = value
	case "network":
		config.Network = value
	case "sender":
		config.Sender = value
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
	case "namespace", "ns":
		return config.Namespace, nil
	case "network":
		return config.Network, nil
	case "sender":
		return config.Sender, nil
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

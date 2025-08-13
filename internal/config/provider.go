package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Provider creates RuntimeConfig for Wire dependency injection
func Provider(v *viper.Viper) (*RuntimeConfig, error) {
	// Get project root from viper
	projectRoot := v.GetString("project_root")
	if projectRoot == "" {
		// Try to find project root
		var err error
		projectRoot, err = FindProjectRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to find project root: %w", err)
		}
	}

	cfg := &RuntimeConfig{
		ProjectRoot:    projectRoot,
		DataDir:        filepath.Join(projectRoot, ".treb"),
		Namespace:      v.GetString("namespace"),
		Debug:          v.GetBool("debug"),
		NonInteractive: v.GetBool("non_interactive"),
		JSON:           v.GetBool("json"),
		Timeout:        v.GetDuration("timeout"),
		DryRun:         v.GetBool("dry_run"),
	}

	// Load foundry config
	foundryConfig, err := loadFoundryConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load foundry config: %w", err)
	}
	cfg.FoundryConfig = foundryConfig

	// Load profile-specific treb config (namespace = profile)
	cfg.TrebConfig = foundryConfig.Profile[cfg.Namespace].Treb

	// Resolve network if specified
	if networkName := v.GetString("network"); networkName != "" {
		networkResolver := NewNetworkResolver(projectRoot, foundryConfig)
		network, err := networkResolver.Resolve(networkName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve network %s: %w", networkName, err)
		}
		cfg.Network = network
	}

	return cfg, nil
}

// FindProjectRoot walks up from current directory to find foundry.toml
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		foundryToml := filepath.Join(dir, "foundry.toml")
		if _, err := os.Stat(foundryToml); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding foundry.toml
			return "", fmt.Errorf("not in a Foundry project (foundry.toml not found)")
		}
		dir = parent
	}
}

// SetupViper creates and configures a viper instance
func SetupViper(projectRoot string) *viper.Viper {
	v := viper.New()

	// Set up config file
	v.SetConfigName("config.local")
	v.SetConfigType("json")
	v.AddConfigPath(filepath.Join(projectRoot, ".treb"))

	// Set up environment variables
	v.SetEnvPrefix("TREB")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Set defaults
	v.SetDefault("namespace", "default")
	v.SetDefault("timeout", "5m")
	v.SetDefault("debug", false)
	v.SetDefault("non_interactive", false)
	v.SetDefault("project_root", projectRoot)

	// Try to read config file (ignore error if not found)
	_ = v.ReadInConfig()

	return v
}

// ProvideNetworkResolver creates a NetworkResolver for Wire dependency injection
func ProvideNetworkResolver(cfg *RuntimeConfig) *NetworkResolver {
	return NewNetworkResolver(cfg.ProjectRoot, cfg.FoundryConfig)
}


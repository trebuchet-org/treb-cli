package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// loadFoundryConfig loads and parses foundry.toml
func loadFoundryConfig(projectRoot string) (*config.FoundryConfig, error) {
	// Load .env files first for variable expansion
	envFiles := []string{
		filepath.Join(projectRoot, ".env"),
		filepath.Join(projectRoot, ".env.local"),
	}

	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err != nil {
				// Log warning but don't fail
				fmt.Fprintf(os.Stderr, "Warning: Failed to load %s: %v\n", envFile, err)
			}
		}
	}

	// Load foundry.toml
	foundryPath := filepath.Join(projectRoot, "foundry.toml")
	var cfg config.FoundryConfig

	if _, err := toml.DecodeFile(foundryPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	// Process RPC endpoints
	for name, url := range cfg.RpcEndpoints {
		cfg.RpcEndpoints[name] = os.ExpandEnv(url)
	}

	// Process etherscan configs
	for _, ethConfig := range cfg.Etherscan {
		ethConfig.URL = os.ExpandEnv(ethConfig.URL)
		ethConfig.Key = os.ExpandEnv(ethConfig.Key)
	}

	return &cfg, nil
}

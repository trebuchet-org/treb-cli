package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the application configuration
type Config struct {
	// ProjectRoot is the root directory of the project
	ProjectRoot string `mapstructure:"project_root"`
	
	// DataDir is the directory for storing application data
	DataDir string `mapstructure:"data_dir"`
	
	// Network is the default network to use
	Network string `mapstructure:"network"`
	
	// Debug enables debug output
	Debug bool `mapstructure:"debug"`
	
	// Timeout for operations
	Timeout time.Duration `mapstructure:"timeout"`
	
	// NonInteractive disables interactive prompts
	NonInteractive bool `mapstructure:"non_interactive"`
}

// LoadConfig loads the configuration from environment and defaults
func LoadConfig() (*Config, error) {
	// Start with defaults
	cfg := &Config{
		ProjectRoot: ".",
		DataDir:     ".treb",
		Timeout:     5 * time.Minute,
	}
	
	// Override with environment variables
	if projectRoot := os.Getenv("TREB_PROJECT_ROOT"); projectRoot != "" {
		cfg.ProjectRoot = projectRoot
	}
	
	if dataDir := os.Getenv("TREB_DATA_DIR"); dataDir != "" {
		cfg.DataDir = dataDir
	}
	
	if network := os.Getenv("TREB_NETWORK"); network != "" {
		cfg.Network = network
	}
	
	if os.Getenv("TREB_DEBUG") == "true" {
		cfg.Debug = true
	}
	
	if os.Getenv("TREB_NON_INTERACTIVE") == "true" {
		cfg.NonInteractive = true
	}
	
	// Resolve paths
	if !filepath.IsAbs(cfg.ProjectRoot) {
		absPath, err := filepath.Abs(cfg.ProjectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project root: %w", err)
		}
		cfg.ProjectRoot = absPath
	}
	
	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check if project root exists
	if _, err := os.Stat(c.ProjectRoot); err != nil {
		return fmt.Errorf("project root does not exist: %s", c.ProjectRoot)
	}
	
	// Check if it's a Foundry project
	foundryToml := filepath.Join(c.ProjectRoot, "foundry.toml")
	if _, err := os.Stat(foundryToml); err != nil {
		return fmt.Errorf("not a Foundry project (foundry.toml not found)")
	}
	
	return nil
}
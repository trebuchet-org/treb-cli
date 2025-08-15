package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// FoundryConfig represents the full foundry.toml configuration
type FoundryConfig struct {
	Profile      map[string]ProfileConfig   `toml:"profile"`
	RpcEndpoints map[string]string          `toml:"rpc_endpoints"`
	Etherscan    map[string]EtherscanConfig `toml:"etherscan,omitempty"`
}

// EtherscanConfig represents Etherscan configuration for a network
type EtherscanConfig struct {
	Key string `toml:"key,omitempty"` // API key for verification
	URL string `toml:"url,omitempty"` // API URL (for custom explorers)
}

// ProfileConfig represents a profile's foundry configuration
type ProfileConfig struct {
	// Foundry settings
	SrcPath       string   `toml:"src,omitempty"`
	OutPath       string   `toml:"out,omitempty"`
	LibPaths      []string `toml:"libs,omitempty"`
	TestPath      string   `toml:"test,omitempty"`
	ScriptPath    string   `toml:"script,omitempty"`
	Remappings    []string `toml:"remappings,omitempty"`
	SolcVersion   string   `toml:"solc_version,omitempty"`
	Optimizer     bool     `toml:"optimizer,omitempty"`
	OptimizerRuns int      `toml:"optimizer_runs,omitempty"`
}

// FoundryManager handles foundry.toml file operations
type FoundryManager struct {
	projectRoot string
	configPath  string
}

// NewFoundryManager creates a new foundry configuration manager
func NewFoundryManager(projectRoot string) *FoundryManager {
	return &FoundryManager{
		projectRoot: projectRoot,
		configPath:  filepath.Join(projectRoot, "foundry.toml"),
	}
}

// Load reads the foundry configuration
func (fm *FoundryManager) Load() (*FoundryConfig, error) {
	// Check if foundry.toml exists
	if _, err := os.Stat(fm.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("foundry.toml not found at %s", fm.configPath)
	}

	// Read the file content
	data, err := os.ReadFile(fm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	// Parse TOML
	var config FoundryConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	return &config, nil
}

// GetRemappings returns the library remapping paths
func (fm *FoundryManager) GetRemappings() []string {
	var paths []string

	// Try to run forge remappings command
	cmd := exec.Command("forge", "remappings")
	cmd.Dir = fm.projectRoot
	output, err := cmd.Output()

	if err == nil {
		// Parse forge remappings output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				to := strings.TrimSpace(parts[1])
				// Remove trailing slash
				to = strings.TrimSuffix(to, "/")
				paths = append(paths, to)
			}
		}
	} else {
		// Fallback to reading from foundry.toml
		config, err := fm.Load()
		if err == nil {
			// Try default profile first
			if profile, ok := config.Profile["default"]; ok {
				paths = append(paths, profile.Remappings...)
			}
			
			// Add lib paths
			if profile, ok := config.Profile["default"]; ok {
				for _, libPath := range profile.LibPaths {
					paths = append(paths, libPath)
				}
			}
		}
	}

	// Always include standard lib paths
	standardPaths := []string{"lib", "node_modules"}
	for _, path := range standardPaths {
		fullPath := filepath.Join(fm.projectRoot, path)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			paths = append(paths, path)
		}
	}

	return paths
}
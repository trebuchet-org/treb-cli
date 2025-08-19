package config

import (
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// RuntimeConfig represents the complete runtime configuration
// This is injected into use cases and contains all resolved settings
type RuntimeConfig struct {
	// Core settings
	ProjectRoot string
	DataDir     string

	// Context settings
	Namespace string          // Maps to foundry profile
	Network   *domain.Network // nil if not specified

	// Execution settings
	Debug          bool
	NonInteractive bool
	JSON           bool // Output in JSON format
	Timeout        time.Duration

	// Command-specific settings (only populated for relevant commands)
	DryRun bool

	// Resolved configurations
	FoundryConfig *FoundryConfig
	TrebConfig    *domain.TrebConfig // Profile-specific treb config
}

// FoundryConfig represents the full foundry.toml configuration
type FoundryConfig struct {
	Profile      map[string]ProfileConfig   `toml:"profile"`
	RpcEndpoints map[string]string          `toml:"rpc_endpoints"`
	Etherscan    map[string]EtherscanConfig `toml:"etherscan,omitempty"`
}

// EtherscanConfig represents Etherscan configuration for a network
// This matches Foundry's expected structure
type EtherscanConfig struct {
	Key string `toml:"key,omitempty"` // API key for verification
	URL string `toml:"url,omitempty"` // API URL (for custom explorers)
}

// ProfileFoundryConfig represents a profile's foundry configuration
type ProfileConfig struct {
	Libraries []string `toml:"libraries,omitempty"`
	// Other foundry settings
	SrcPath       string             `toml:"src,omitempty"`
	OutPath       string             `toml:"out,omitempty"`
	LibPaths      []string           `toml:"libs,omitempty"`
	TestPath      string             `toml:"test,omitempty"`
	ScriptPath    string             `toml:"script,omitempty"`
	Remappings    []string           `toml:"remappings,omitempty"`
	SolcVersion   string             `toml:"solc_version,omitempty"`
	Optimizer     bool               `toml:"optimizer,omitempty"`
	OptimizerRuns int                `toml:"optimizer_runs,omitempty"`
	Treb          *domain.TrebConfig `toml:"treb,omitempty"`
}

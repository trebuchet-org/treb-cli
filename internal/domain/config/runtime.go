package config

import (
	"time"
)

// RuntimeConfig represents the complete runtime configuration
// This is injected into use cases and contains all resolved settings
type RuntimeConfig struct {
	// Core settings
	ProjectRoot string
	DataDir     string

	// Context settings
	Namespace string   // Maps to foundry profile
	Network   *Network // nil if not specified

	// Execution settings
	Debug          bool
	NonInteractive bool
	JSON           bool // Output in JSON format
	Timeout        time.Duration

	// Command-specific settings (only populated for relevant commands)
	DryRun bool
	Slow   bool

	// Config source tracking
	FoundryProfile string // Foundry profile name to use (from treb.toml or namespace)
	ConfigSource   string // "treb.toml" or "foundry.toml"

	// Resolved configurations
	FoundryConfig *FoundryConfig
	TrebConfig    *TrebConfig // Profile-specific treb config
}

// Network represents network configuration
type Network struct {
	ChainID     uint64 `json:"chainId"`
	Name        string `json:"name"`
	RPCURL      string `json:"rpcUrl"`
	ExplorerURL string `json:"explorerUrl,omitempty"`
}

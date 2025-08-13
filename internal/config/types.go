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
	Namespace string         // Maps to foundry profile
	Network   *NetworkConfig // nil if not specified

	// Execution settings
	Debug          bool
	NonInteractive bool
	JSON           bool // Output in JSON format
	Timeout        time.Duration

	// Command-specific settings (only populated for relevant commands)
	DryRun bool

	// Resolved configurations
	FoundryConfig *FoundryConfig
	TrebConfig    *TrebConfig // Profile-specific treb config
}

// NetworkConfig contains resolved network information
type NetworkConfig struct {
	Name       string
	RpcUrl     string
	ChainID    uint64
	Explorer   string
	Configured bool // Whether network was explicitly configured
}

// SenderConfig represents the active sender configuration
type SenderConfig struct {
	Name    string
	Type    string // "private_key", "safe", "ledger"
	Address string

	// Type-specific fields
	PrivateKey     string `json:"-"` // Never log/display
	Safe           string
	Proposer       string
	DerivationPath string
}

// TrebConfig from foundry.toml [profile.*.treb] section
type TrebConfig struct {
	Senders         map[string]SenderConfig
	LibraryDeployer string
}

// FoundryConfig represents parsed foundry.toml
type FoundryConfig struct {
	RpcEndpoints map[string]string
	Etherscan    map[string]EtherscanConfig
	Profiles     map[string]ProfileConfig
}

// EtherscanConfig for block explorer settings
type EtherscanConfig struct {
	URL    string
	APIKey string
}

// ProfileConfig contains profile-specific settings
type ProfileConfig struct {
	Treb TrebConfig `toml:"treb"`
	// Other foundry profile settings can be added here as needed
}


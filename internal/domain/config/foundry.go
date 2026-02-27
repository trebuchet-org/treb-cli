package config

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
	SrcPath       string      `toml:"src,omitempty"`
	OutPath       string      `toml:"out,omitempty"`
	LibPaths      []string    `toml:"libs,omitempty"`
	TestPath      string      `toml:"test,omitempty"`
	ScriptPath    string      `toml:"script,omitempty"`
	Remappings    []string    `toml:"remappings,omitempty"`
	SolcVersion   string      `toml:"solc_version,omitempty"`
	Optimizer     bool        `toml:"optimizer,omitempty"`
	OptimizerRuns int         `toml:"optimizer_runs,omitempty"`
	Treb          *TrebConfig `toml:"treb,omitempty"`
}

// TrebConfig represents treb-specific configuration
type TrebConfig struct {
	Senders map[string]SenderConfig `json:"senders" toml:"senders"`
}

type SenderType string

var (
	SenderTypeLedger     SenderType = "ledger"
	SenderTypeTrezor     SenderType = "trezor"
	SenderTypeSafe       SenderType = "safe"
	SenderTypePrivateKey SenderType = "private_key"
	SenderTypeOZGovernor SenderType = "oz_governor"
)

// SenderConfig represents a sender configuration
type SenderConfig struct {
	Type           SenderType `toml:"type"`
	Address        string     `toml:"address,omitempty"`
	PrivateKey     string     `toml:"private_key,omitempty"` //nolint:gosec // holds env var reference, not a literal secret
	Safe           string     `toml:"safe,omitempty"`
	Signer         string     `toml:"signer,omitempty"`          // For Safe senders
	DerivationPath string     `toml:"derivation_path,omitempty"` // For Ledger senders
	Governor       string     `toml:"governor,omitempty"`        // For OZ Governor senders
	Timelock       string     `toml:"timelock,omitempty"`        // For OZ Governor senders (optional)
	Proposer       string     `toml:"proposer,omitempty"`        // For OZ Governor senders
}

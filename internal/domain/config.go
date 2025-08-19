package domain

// LocalConfig represents the local treb configuration
type LocalConfig struct {
	Namespace string `json:"namespace"`
	Network   string `json:"network"`
}

// ConfigKey represents a configuration key
type ConfigKey string

const (
	ConfigKeyNamespace ConfigKey = "namespace"
	ConfigKeyNetwork   ConfigKey = "network"
)

// DefaultLocalConfig returns the default local configuration
func DefaultLocalConfig() *LocalConfig {
	return &LocalConfig{
		Namespace: "default",
		Network:   "",
	}
}

// ValidConfigKeys returns all valid configuration keys
func ValidConfigKeys() []ConfigKey {
	return []ConfigKey{
		ConfigKeyNamespace,
		ConfigKeyNetwork,
	}
}

// IsValidConfigKey checks if a key is valid
func IsValidConfigKey(key string) bool {
	for _, validKey := range ValidConfigKeys() {
		if string(validKey) == key || (key == "ns" && validKey == ConfigKeyNamespace) {
			return true
		}
	}
	return false
}

// NormalizeConfigKey normalizes a config key (e.g., "ns" -> "namespace")
func NormalizeConfigKey(key string) ConfigKey {
	if key == "ns" {
		return ConfigKeyNamespace
	}
	return ConfigKey(key)
}

// Network represents network configuration
type Network struct {
	ChainID     uint64 `json:"chainId"`
	Name        string `json:"name"`
	RPCURL      string `json:"rpcUrl"`
	ExplorerURL string `json:"explorerUrl,omitempty"`
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
)

// SenderConfig represents a sender configuration
type SenderConfig struct {
	Type           SenderType `toml:"type"`
	Address        string     `toml:"address,omitempty"`
	PrivateKey     string     `toml:"private_key,omitempty"`
	Safe           string     `toml:"safe,omitempty"`
	Signer         string     `toml:"signer,omitempty"`          // For Safe senders
	DerivationPath string     `toml:"derivation_path,omitempty"` // For Ledger senders
}

type SenderScriptConfig struct {
	UseLedger       bool
	UseTrezor       bool
	DerivationPaths []string
	EncodedConfig   string
	Senders         []string
}

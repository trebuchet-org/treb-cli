package config

// LocalConfig represents the local treb configuration
type LocalConfig struct {
	Namespace string `json:"namespace"`
	Network   string `json:"network"`
	ForkSetup string `json:"forkSetup,omitempty"`
}

// ConfigKey represents a configuration key
type ConfigKey string

const (
	ConfigKeyNamespace ConfigKey = "namespace"
	ConfigKeyNetwork   ConfigKey = "network"
	ConfigKeyForkSetup ConfigKey = "fork.setup"
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
		ConfigKeyForkSetup,
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

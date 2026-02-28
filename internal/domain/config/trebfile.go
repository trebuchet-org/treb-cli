package config

// TrebFileConfig represents the full treb.toml configuration file
type TrebFileConfig struct {
	Slow *bool                      `toml:"slow,omitempty"`
	Ns   map[string]NamespaceConfig `toml:"ns"`
}

// NamespaceConfig represents a [ns.<name>] section in treb.toml
type NamespaceConfig struct {
	Profile string                  `toml:"profile,omitempty"`
	Slow    *bool                   `toml:"slow,omitempty"`
	Senders map[string]SenderConfig `toml:"senders"`
}

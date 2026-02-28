package config

// AccountConfig represents a named signing entity in [accounts.*] sections.
// It has the same fields as SenderConfig but is defined at the top level,
// decoupled from any specific namespace.
type AccountConfig struct {
	Type           SenderType `toml:"type"`
	Address        string     `toml:"address,omitempty"`
	PrivateKey     string     `toml:"private_key,omitempty"` //nolint:gosec // holds env var reference, not a literal secret
	Safe           string     `toml:"safe,omitempty"`
	Signer         string     `toml:"signer,omitempty"`          // For Safe accounts: references another account name
	DerivationPath string     `toml:"derivation_path,omitempty"` // For Ledger/Trezor accounts
	Governor       string     `toml:"governor,omitempty"`        // For OZ Governor accounts
	Timelock       string     `toml:"timelock,omitempty"`        // For OZ Governor accounts (optional)
	Proposer       string     `toml:"proposer,omitempty"`        // For OZ Governor accounts: references another account name
}

// NamespaceRoles represents a [namespace.*] section in treb.toml v2.
// Profile maps to a foundry.toml profile, and Roles maps role names to account names.
type NamespaceRoles struct {
	Profile string            `toml:"profile,omitempty"`
	Roles   map[string]string `toml:"-"` // Populated manually from remaining TOML keys
}

// TrebFileConfigV2 represents the new treb.toml format with separate accounts and namespaces.
type TrebFileConfigV2 struct {
	Accounts  map[string]AccountConfig  `toml:"accounts"`
	Namespace map[string]NamespaceRoles `toml:"namespace"`
	Fork      ForkConfig                `toml:"fork"`
}

// ForkConfig represents the [fork] section in treb.toml v2.
type ForkConfig struct {
	Setup string `toml:"setup,omitempty"`
}

// ResolvedNamespace holds the fully-resolved configuration for a namespace
// after walking the dot-based hierarchy and resolving role→account mappings.
type ResolvedNamespace struct {
	Profile  string                   // Resolved foundry profile name
	Accounts map[string]AccountConfig // role name → resolved AccountConfig
}

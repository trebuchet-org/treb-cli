package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// TrebConfigFormat represents the detected format of the treb configuration.
type TrebConfigFormat int

const (
	TrebConfigFormatNone TrebConfigFormat = iota // No treb.toml or unrecognized format
	TrebConfigFormatV1                           // Old [ns.*] format
	TrebConfigFormatV2                           // New [accounts.*] / [namespace.*] format
)

// detectTrebConfigFormat checks if treb.toml exists and determines which format version it uses.
func detectTrebConfigFormat(projectRoot string) (TrebConfigFormat, error) {
	trebPath := filepath.Join(projectRoot, "treb.toml")

	if _, err := os.Stat(trebPath); os.IsNotExist(err) {
		return TrebConfigFormatNone, nil
	}

	var raw map[string]interface{}
	if _, err := toml.DecodeFile(trebPath, &raw); err != nil {
		return TrebConfigFormatNone, fmt.Errorf("failed to parse treb.toml: %w", err)
	}

	// Check for v2 sections first (accounts/namespace take priority)
	if _, ok := raw["accounts"]; ok {
		return TrebConfigFormatV2, nil
	}
	if _, ok := raw["namespace"]; ok {
		return TrebConfigFormatV2, nil
	}

	// Check for v1 sections
	if _, ok := raw["ns"]; ok {
		return TrebConfigFormatV1, nil
	}

	// File exists but has neither format (might be empty or only [fork])
	return TrebConfigFormatNone, nil
}

// trebFileV2Raw is a helper for initial TOML parsing of v2 format.
// Namespace uses map[string]string because all values (profile + roles) are flat strings.
type trebFileV2Raw struct {
	Accounts  map[string]config.AccountConfig `toml:"accounts"`
	Namespace map[string]map[string]string    `toml:"namespace"`
	Fork      config.ForkConfig               `toml:"fork"`
}

// loadTrebConfigV2 loads and parses treb.toml in the v2 format with [accounts.*] and [namespace.*] sections.
// Returns (nil, nil) if treb.toml doesn't exist or doesn't have v2 format sections.
func loadTrebConfigV2(projectRoot string) (*config.TrebFileConfigV2, error) {
	trebPath := filepath.Join(projectRoot, "treb.toml")

	if _, err := os.Stat(trebPath); os.IsNotExist(err) {
		return nil, nil
	}

	var raw trebFileV2Raw
	if _, err := toml.DecodeFile(trebPath, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse treb.toml: %w", err)
	}

	// If no accounts and no namespace sections, this isn't v2 format
	if len(raw.Accounts) == 0 && len(raw.Namespace) == 0 {
		return nil, nil
	}

	cfg := &config.TrebFileConfigV2{
		Accounts:  raw.Accounts,
		Namespace: make(map[string]config.NamespaceRoles),
		Fork:      raw.Fork,
	}

	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]config.AccountConfig)
	}

	// Parse namespace sections: "profile" is a reserved key, all others become role→account mappings
	for nsName, nsMap := range raw.Namespace {
		ns := config.NamespaceRoles{
			Roles: make(map[string]string),
		}
		for k, v := range nsMap {
			if k == "profile" {
				ns.Profile = v
			} else {
				ns.Roles[k] = v
			}
		}
		cfg.Namespace[nsName] = ns
	}

	// Expand environment variables in all account config string fields
	for name, acct := range cfg.Accounts {
		acct.PrivateKey = os.ExpandEnv(acct.PrivateKey)
		acct.Safe = os.ExpandEnv(acct.Safe)
		acct.Address = os.ExpandEnv(acct.Address)
		acct.Signer = os.ExpandEnv(acct.Signer)
		acct.DerivationPath = os.ExpandEnv(acct.DerivationPath)
		acct.Governor = os.ExpandEnv(acct.Governor)
		acct.Timelock = os.ExpandEnv(acct.Timelock)
		acct.Proposer = os.ExpandEnv(acct.Proposer)
		cfg.Accounts[name] = acct
	}

	return cfg, nil
}

// ResolveNamespace resolves a namespace's full configuration by walking up the
// dot-separated hierarchy and overlaying roles and profile at each level.
// For example, resolving "production.ntt" walks: default → production → production.ntt.
// At each level, explicitly set values override inherited ones.
func ResolveNamespace(cfg *config.TrebFileConfigV2, namespaceName string) (*config.ResolvedNamespace, error) {
	// Build the ancestry chain: always start with "default", then each prefix segment
	chain := buildNamespaceChain(namespaceName)

	// Accumulate profile and roles by walking the chain
	profile := ""
	roles := make(map[string]string)

	for _, ancestor := range chain {
		ns, exists := cfg.Namespace[ancestor]
		if !exists {
			continue
		}
		if ns.Profile != "" {
			profile = ns.Profile
		}
		for role, account := range ns.Roles {
			roles[role] = account
		}
	}

	// Validate that all role values reference existing account names
	for role, accountName := range roles {
		if _, exists := cfg.Accounts[accountName]; !exists {
			return nil, fmt.Errorf("namespace %q role %q references unknown account %q", namespaceName, role, accountName)
		}
	}

	// Build resolved accounts map: role name → AccountConfig
	accounts := make(map[string]config.AccountConfig, len(roles))
	for role, accountName := range roles {
		accounts[role] = cfg.Accounts[accountName]
	}

	return &config.ResolvedNamespace{
		Profile:  profile,
		Accounts: accounts,
	}, nil
}

// buildNamespaceChain returns the ordered list of namespace names to resolve,
// starting from "default" and adding each dot-separated prefix.
// For "production.ntt.v2" it returns: ["default", "production", "production.ntt", "production.ntt.v2"]
// For "default" it returns just: ["default"]
func buildNamespaceChain(namespaceName string) []string {
	if namespaceName == "default" {
		return []string{"default"}
	}

	chain := []string{"default"}
	parts := strings.Split(namespaceName, ".")
	for i := range parts {
		chain = append(chain, strings.Join(parts[:i+1], "."))
	}
	return chain
}

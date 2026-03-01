package config

import (
	"fmt"
	"io"
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

// DetectTrebConfigFormat checks if treb.toml exists and determines which format version it uses.
func DetectTrebConfigFormat(projectRoot string) (TrebConfigFormat, error) {
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

	// Treat [fork] section as V2 so fork.setup is loaded from treb.toml
	if _, ok := raw["fork"]; ok {
		return TrebConfigFormatV2, nil
	}

	// File exists but has neither format (empty or unrecognized)
	return TrebConfigFormatNone, nil
}

// trebFileV2Raw is a helper for initial TOML parsing of v2 format.
// Namespace sections decode directly into NamespaceRoles structs with profile and senders sub-table.
type trebFileV2Raw struct {
	Accounts  map[string]config.AccountConfig  `toml:"accounts"`
	Namespace map[string]config.NamespaceRoles `toml:"namespace"`
	Fork      config.ForkConfig                `toml:"fork"`
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

	// If no accounts, no namespace, and no fork config, this isn't v2 format
	if len(raw.Accounts) == 0 && len(raw.Namespace) == 0 && raw.Fork.Setup == "" {
		return nil, nil
	}

	cfg := &config.TrebFileConfigV2{
		Accounts:  raw.Accounts,
		Namespace: raw.Namespace,
		Fork:      raw.Fork,
	}

	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]config.AccountConfig)
	}

	if cfg.Namespace == nil {
		cfg.Namespace = make(map[string]config.NamespaceRoles)
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
// Roles that reference unknown accounts are skipped with a warning to warnWriter.
// Pass nil for warnWriter to use os.Stderr.
func ResolveNamespace(cfg *config.TrebFileConfigV2, namespaceName string, warnWriter ...io.Writer) (*config.ResolvedNamespace, error) {
	w := resolveWarnWriter(warnWriter)

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
		for role, account := range ns.Senders {
			roles[role] = account
		}
	}

	// Build resolved accounts map, skipping roles that reference unknown accounts
	accounts := make(map[string]config.AccountConfig, len(roles))
	for role, accountName := range roles {
		if _, exists := cfg.Accounts[accountName]; !exists {
			fmt.Fprintf(w, "Warning: namespace %q role %q references unknown account %q — skipping\n", namespaceName, role, accountName)
			continue
		}
		accounts[role] = cfg.Accounts[accountName]
	}

	return &config.ResolvedNamespace{
		Profile:  profile,
		Accounts: accounts,
	}, nil
}

// ResolvedNamespaceToTrebConfig converts a ResolvedNamespace (accounts + role mappings)
// into the existing TrebConfig with Senders map that the rest of the codebase expects.
// Each role in the resolved namespace becomes a SenderConfig entry keyed by role name.
// Cross-references (Safe signer, OzGovernor proposer) that reference unknown accounts
// are skipped with a warning to warnWriter. Pass nil for warnWriter to use os.Stderr.
func ResolvedNamespaceToTrebConfig(resolved *config.ResolvedNamespace, accounts map[string]config.AccountConfig, warnWriter ...io.Writer) (*config.TrebConfig, error) {
	w := resolveWarnWriter(warnWriter)
	senders := make(map[string]config.SenderConfig, len(resolved.Accounts))

	for roleName, acct := range resolved.Accounts {
		sender := config.SenderConfig(acct)

		// Check Safe signer cross-reference
		if acct.Type == config.SenderTypeSafe && acct.Signer != "" {
			if _, exists := accounts[acct.Signer]; !exists {
				fmt.Fprintf(w, "Warning: role %q references safe with unknown signer account %q — skipping\n", roleName, acct.Signer)
				continue
			}
		}

		// Check OzGovernor proposer cross-reference
		if acct.Type == config.SenderTypeOZGovernor && acct.Proposer != "" {
			if _, exists := accounts[acct.Proposer]; !exists {
				fmt.Fprintf(w, "Warning: role %q references oz_governor with unknown proposer account %q — skipping\n", roleName, acct.Proposer)
				continue
			}
		}

		senders[roleName] = sender
	}

	return &config.TrebConfig{Senders: senders}, nil
}

// resolveWarnWriter returns the first writer from the variadic args, or os.Stderr if none provided.
func resolveWarnWriter(writers []io.Writer) io.Writer {
	if len(writers) > 0 && writers[0] != nil {
		return writers[0]
	}
	return os.Stderr
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

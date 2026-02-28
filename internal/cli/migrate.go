package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	internalconfig "github.com/trebuchet-org/treb-cli/internal/config"
	domainconfig "github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// NewMigrateCmd creates the migrate command that converts foundry.toml
// legacy sender configs to the new treb.toml v2 accounts/namespace format.
func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate config to new treb.toml accounts/namespace format",
		Long: `Migrate treb sender configuration from foundry.toml [profile.*.treb.*] sections
into the new treb.toml format with [accounts.*] and [namespace.*] sections.

This command will:
1. Read all [profile.*.treb.*] sections from foundry.toml
2. Deduplicate identical sender configs into shared accounts
3. Map profile names to namespaces with role→account mappings
4. Show a preview of the generated treb.toml
5. Ask for confirmation before writing

Examples:
  # Interactive migration (shows preview, asks for confirmation)
  treb migrate

  # Non-interactive migration (writes without prompts)
  treb migrate --non-interactive`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			return runMigrate(app.Config)
		},
	}

	return cmd
}

// runMigrate performs the config migration from foundry.toml to treb.toml v2 format.
func runMigrate(cfg *domainconfig.RuntimeConfig) error {
	trebTomlPath := filepath.Join(cfg.ProjectRoot, "treb.toml")
	foundryTomlPath := filepath.Join(cfg.ProjectRoot, "foundry.toml")

	// Check if treb.toml already exists in v2 format
	format, err := internalconfig.DetectTrebConfigFormat(cfg.ProjectRoot)
	if err != nil {
		return err
	}
	if format == internalconfig.TrebConfigFormatV2 {
		return fmt.Errorf("treb.toml already uses the new accounts/namespace format — nothing to migrate")
	}

	// Extract treb profiles from foundry.toml
	profiles := extractTrebProfiles(cfg.FoundryConfig)
	if len(profiles) == 0 {
		fmt.Println("No treb config found in foundry.toml — nothing to migrate.")
		return nil
	}

	// Build namespace→senders map for deduplication
	namespaceSenders := make(map[string]map[string]domainconfig.SenderConfig, len(profiles))
	for _, p := range profiles {
		namespaceSenders[p.Name] = p.Senders
	}

	// Deduplicate senders across profiles
	accounts, namespaceMappings := internalconfig.DeduplicateSenders(namespaceSenders)

	// Build namespace info (profile name for each namespace)
	namespaces := make(map[string]nsInfo, len(profiles))
	for _, p := range profiles {
		namespaces[p.Name] = nsInfo{
			profile: p.Name,
			roles:   namespaceMappings[p.Name],
		}
	}

	// Generate v2 treb.toml content
	content := generateTrebTomlV2(accounts, namespaces)

	// Check if treb.toml already exists (v1 or other)
	if _, err := os.Stat(trebTomlPath); err == nil {
		if cfg.NonInteractive {
			fmt.Fprintln(os.Stderr, "Warning: treb.toml already exists and will be overwritten.")
		} else {
			yellow := color.New(color.FgYellow)
			yellow.Fprintln(os.Stderr, "Warning: treb.toml already exists.")
			if !confirmPrompt("Overwrite existing treb.toml?") {
				fmt.Println("Migration cancelled.")
				return nil
			}
		}
	}

	// Show preview
	if !cfg.NonInteractive {
		fmt.Println("Generated treb.toml:")
		fmt.Println()
		fmt.Println(content)

		if !confirmPrompt("Write this to treb.toml?") {
			fmt.Println("Migration cancelled.")
			return nil
		}
	}

	// Write treb.toml
	if err := os.WriteFile(trebTomlPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write treb.toml: %w", err)
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Printf("✓ treb.toml written successfully\n")

	// Offer to clean up foundry.toml
	cleanedUp := false
	if !cfg.NonInteractive {
		fmt.Println()
		if confirmPrompt("Remove [profile.*.treb.*] sections from foundry.toml?") {
			if err := cleanupFoundryToml(foundryTomlPath); err != nil {
				return fmt.Errorf("failed to clean up foundry.toml: %w", err)
			}
			green.Printf("✓ foundry.toml cleaned up\n")
			cleanedUp = true
		} else {
			fmt.Println("Skipped foundry.toml cleanup — you can remove [profile.*.treb.*] sections manually.")
		}
	}

	// Print next steps
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review the generated treb.toml")
	if !cleanedUp {
		fmt.Println("  2. Remove [profile.*.treb.*] sections from foundry.toml")
		fmt.Println("  3. Run `treb config show` to verify your config is loaded correctly")
	} else {
		fmt.Println("  2. Run `treb config show` to verify your config is loaded correctly")
	}

	return nil
}

// nsInfo holds namespace info for TOML generation.
type nsInfo struct {
	profile string
	roles   map[string]string
}

// generateTrebTomlV2 generates well-formatted treb.toml content in v2 format
// with [accounts.*] and [namespace.*] sections.
func generateTrebTomlV2(
	accounts map[string]domainconfig.AccountConfig,
	namespaces map[string]nsInfo,
) string {
	var b strings.Builder

	b.WriteString("# treb.toml — Treb configuration\n")
	b.WriteString("#\n")
	b.WriteString("# Accounts define signing entities (wallets, hardware wallets, multisigs).\n")
	b.WriteString("# Namespaces map roles to accounts for different environments.\n")
	b.WriteString("#\n")
	b.WriteString("# Migrated from foundry.toml [profile.*.treb.*] sections.\n")

	// Write accounts sorted by name
	accountNames := sortedKeys(accounts)
	for _, name := range accountNames {
		acct := accounts[name]
		b.WriteString("\n")
		fmt.Fprintf(&b, "[accounts.%s]\n", tomlKey(name))
		writeAccountConfig(&b, acct)
	}

	// Write namespaces sorted by name (default first)
	nsNames := sortedKeys(namespaces)
	sort.SliceStable(nsNames, func(i, j int) bool {
		if nsNames[i] == "default" {
			return true
		}
		if nsNames[j] == "default" {
			return false
		}
		return false // preserve existing alphabetical order
	})

	for _, nsName := range nsNames {
		ns := namespaces[nsName]
		b.WriteString("\n")
		fmt.Fprintf(&b, "[namespace.%s]\n", tomlKey(nsName))
		fmt.Fprintf(&b, "profile = %q\n", ns.profile)

		// Write role mappings sorted by name
		roleNames := sortedKeys(ns.roles)
		for _, role := range roleNames {
			fmt.Fprintf(&b, "%s = %q\n", role, ns.roles[role])
		}
	}

	return b.String()
}

// writeAccountConfig writes an AccountConfig's fields to the builder.
func writeAccountConfig(b *strings.Builder, a domainconfig.AccountConfig) {
	fmt.Fprintf(b, "type = %q\n", string(a.Type))

	if a.Address != "" {
		fmt.Fprintf(b, "address = %q\n", a.Address)
	}
	if a.PrivateKey != "" {
		fmt.Fprintf(b, "private_key = %q\n", a.PrivateKey)
	}
	if a.Safe != "" {
		fmt.Fprintf(b, "safe = %q\n", a.Safe)
	}
	if a.Signer != "" {
		fmt.Fprintf(b, "signer = %q\n", a.Signer)
	}
	if a.DerivationPath != "" {
		fmt.Fprintf(b, "derivation_path = %q\n", a.DerivationPath)
	}
	if a.Governor != "" {
		fmt.Fprintf(b, "governor = %q\n", a.Governor)
	}
	if a.Timelock != "" {
		fmt.Fprintf(b, "timelock = %q\n", a.Timelock)
	}
	if a.Proposer != "" {
		fmt.Fprintf(b, "proposer = %q\n", a.Proposer)
	}
}

// tomlKey quotes a key if it contains dots to prevent TOML nested table interpretation.
// e.g. "production.ntt" → `"production.ntt"` so [namespace."production.ntt"] is a single key.
func tomlKey(key string) string {
	if strings.Contains(key, ".") {
		return `"` + key + `"`
	}
	return key
}

// sortedKeys returns the sorted keys of a map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

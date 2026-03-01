package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
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

	// Load raw foundry config (without env var expansion) so we preserve ${VAR}
	// references in the migrated output instead of leaking actual secrets.
	rawFoundryConfig, err := internalconfig.LoadFoundryConfigRaw(cfg.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to load foundry config: %w", err)
	}

	// Extract treb profiles from raw foundry.toml
	profiles := extractTrebProfiles(rawFoundryConfig)
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

	// Interactive account naming: prompt user to name each deduplicated account
	if !cfg.NonInteractive {
		renamed, err := interactiveAccountNaming(accounts, namespaceMappings)
		if err != nil {
			return err
		}
		accounts = renamed
	}

	// Build namespace info (profile name for each namespace)
	namespaces := make(map[string]nsInfo, len(profiles))
	for _, p := range profiles {
		namespaces[p.Name] = nsInfo{
			profile: p.Name,
			roles:   namespaceMappings[p.Name],
		}
	}

	// Namespace pruning: offer to remove namespaces with zero deployments
	if !cfg.NonInteractive {
		deploymentCounts, err := countDeploymentsPerNamespace(cfg.ProjectRoot)
		if err != nil {
			return err
		}
		if deploymentCounts != nil {
			if err := pruneEmptyNamespaces(namespaces, deploymentCounts); err != nil {
				return err
			}
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

		// Write role mappings under [namespace.<name>.senders] sub-table
		if len(ns.roles) > 0 {
			b.WriteString("\n")
			fmt.Fprintf(&b, "[namespace.%s.senders]\n", tomlKey(nsName))
			roleNames := sortedKeys(ns.roles)
			for _, role := range roleNames {
				fmt.Fprintf(&b, "%s = %q\n", role, ns.roles[role])
			}
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

// trebProfile holds a foundry profile name and its treb sender config.
type trebProfile struct {
	Name    string
	Senders map[string]domainconfig.SenderConfig
}

// extractTrebProfiles extracts foundry profiles that have treb sender configs.
func extractTrebProfiles(fc *domainconfig.FoundryConfig) []trebProfile {
	if fc == nil {
		return nil
	}

	var profiles []trebProfile
	for name, profile := range fc.Profile {
		if profile.Treb != nil && len(profile.Treb.Senders) > 0 {
			profiles = append(profiles, trebProfile{
				Name:    name,
				Senders: profile.Treb.Senders,
			})
		}
	}

	// Sort by name for deterministic output (default first)
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Name == "default" {
			return true
		}
		if profiles[j].Name == "default" {
			return false
		}
		return profiles[i].Name < profiles[j].Name
	})

	return profiles
}

// cleanupFoundryToml reads foundry.toml and removes [profile.*.treb.*] sections.
func cleanupFoundryToml(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	cleaned := removeTrebFromFoundryToml(string(data))
	return os.WriteFile(path, []byte(cleaned), 0644)
}

// trebSectionHeaderRe matches [profile.<name>.treb] and [profile.<name>.treb.senders.<sender>] headers.
var trebSectionHeaderRe = regexp.MustCompile(`^\[profile\.[^]]+\.treb(?:\.[^]]+)?\]\s*$`)

// anySectionHeaderRe matches any TOML section header like [something] or [a.b.c].
var anySectionHeaderRe = regexp.MustCompile(`^\[.+\]\s*$`)

// removeTrebFromFoundryToml removes [profile.*.treb.*] sections from foundry.toml content
// using a line-based approach to preserve user formatting and comments.
func removeTrebFromFoundryToml(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTrebSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if anySectionHeaderRe.MatchString(trimmed) {
			if trebSectionHeaderRe.MatchString(trimmed) {
				inTrebSection = true
				continue
			}
			inTrebSection = false
		}

		if inTrebSection {
			continue
		}

		result = append(result, line)
	}

	// Clean up excess trailing blank lines (collapse to single trailing newline)
	output := strings.Join(result, "\n")
	for strings.HasSuffix(output, "\n\n") {
		output = strings.TrimSuffix(output, "\n")
	}
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return output
}

// confirmPrompt asks the user a yes/no question and returns their choice.
func confirmPrompt(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()
	return err == nil
}

// validAccountNameRe matches valid TOML bare keys: alphanumeric, hyphens, and underscores.
var validAccountNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validateAccountName checks whether a proposed account name is valid.
// Returns an error message string if invalid, or empty string if valid.
func validateAccountName(name string, usedNames map[string]bool) string {
	if name == "" {
		return "name cannot be empty"
	}
	if !validAccountNameRe.MatchString(name) {
		return "name must contain only letters, digits, hyphens, and underscores"
	}
	if usedNames[name] {
		return fmt.Sprintf("name %q is already taken", name)
	}
	return ""
}

// formatAccountSummary returns a short human-readable description of an account.
func formatAccountSummary(acct domainconfig.AccountConfig) string {
	switch acct.Type {
	case domainconfig.SenderTypePrivateKey:
		return fmt.Sprintf("private_key (%s)", acct.PrivateKey)
	case domainconfig.SenderTypeSafe:
		return fmt.Sprintf("safe (%s)", acct.Safe)
	case domainconfig.SenderTypeLedger:
		if acct.Address != "" {
			return fmt.Sprintf("ledger (%s)", acct.Address)
		}
		return fmt.Sprintf("ledger (path: %s)", acct.DerivationPath)
	case domainconfig.SenderTypeTrezor:
		if acct.Address != "" {
			return fmt.Sprintf("trezor (%s)", acct.Address)
		}
		return fmt.Sprintf("trezor (path: %s)", acct.DerivationPath)
	case domainconfig.SenderTypeOZGovernor:
		return fmt.Sprintf("oz_governor (%s)", acct.Governor)
	default:
		return string(acct.Type)
	}
}

// countDeploymentsPerNamespace reads .treb/deployments.json and returns a map
// of namespace → deployment count. Returns (nil, nil) if the file doesn't exist.
func countDeploymentsPerNamespace(projectRoot string) (map[string]int, error) {
	deploymentsPath := filepath.Join(projectRoot, ".treb", "deployments.json")
	data, err := os.ReadFile(deploymentsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read deployments.json: %w", err)
	}

	// Parse just enough to extract namespace from each deployment entry.
	var deployments map[string]struct {
		Namespace string `json:"namespace"`
	}
	if err := json.Unmarshal(data, &deployments); err != nil {
		return nil, fmt.Errorf("failed to parse deployments.json: %w", err)
	}

	counts := make(map[string]int)
	for _, d := range deployments {
		counts[d.Namespace]++
	}
	return counts, nil
}

// pruneEmptyNamespaces prompts the user to remove namespaces with zero deployments.
// It modifies the namespaces map in place, removing declined namespaces.
// Returns an error only if the user cancels (Ctrl+C).
func pruneEmptyNamespaces(
	namespaces map[string]nsInfo,
	deploymentCounts map[string]int,
) error {
	nsNames := sortedKeys(namespaces)
	for _, name := range nsNames {
		count := deploymentCounts[name]
		if count > 0 {
			continue
		}
		label := fmt.Sprintf("Namespace %q has no deployments. Keep it?", name)
		prompt := promptui.Prompt{
			Label:     label,
			IsConfirm: true,
		}
		_, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				return fmt.Errorf("migration cancelled")
			}
			// User declined (entered "n" or just pressed Enter for default N)
			delete(namespaces, name)
		}
	}
	return nil
}

// interactiveAccountNaming prompts the user to name each deduplicated account.
// It returns the renamed accounts map. The namespaceMappings are updated in-place
// to reference the new names.
func interactiveAccountNaming(
	accounts map[string]domainconfig.AccountConfig,
	namespaceMappings map[string]map[string]string,
) (map[string]domainconfig.AccountConfig, error) {
	names := sortedKeys(accounts)
	if len(names) == 0 {
		return accounts, nil
	}

	fmt.Println()
	fmt.Printf("Found %d deduplicated account(s). Name each one (press Enter to keep default):\n", len(names))
	fmt.Println()

	usedNames := make(map[string]bool)
	// Map old name → new name for renaming
	renames := make(map[string]string, len(names))

	for _, oldName := range names {
		acct := accounts[oldName]
		summary := formatAccountSummary(acct)

		var newName string
		for {
			prompt := promptui.Prompt{
				Label:   fmt.Sprintf("  %s — name", summary),
				Default: oldName,
				Validate: func(input string) error {
					if msg := validateAccountName(input, usedNames); msg != "" {
						return fmt.Errorf("%s", msg)
					}
					return nil
				},
			}

			result, err := prompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					return nil, fmt.Errorf("migration cancelled")
				}
				// Validation error — promptui re-prompts automatically,
				// but if Run() returns an unexpected error, retry.
				continue
			}
			newName = result
			break
		}

		usedNames[newName] = true
		renames[oldName] = newName
	}

	// Build renamed accounts map
	renamed := make(map[string]domainconfig.AccountConfig, len(accounts))
	for oldName, acct := range accounts {
		renamed[renames[oldName]] = acct
	}

	// Update signer/proposer cross-references in accounts
	for name, acct := range renamed {
		updated := false
		if acct.Signer != "" {
			if newName, ok := renames[acct.Signer]; ok {
				acct.Signer = newName
				updated = true
			}
		}
		if acct.Proposer != "" {
			if newName, ok := renames[acct.Proposer]; ok {
				acct.Proposer = newName
				updated = true
			}
		}
		if updated {
			renamed[name] = acct
		}
	}

	// Update namespace mappings to use new names
	for _, roles := range namespaceMappings {
		for role, oldName := range roles {
			if newName, ok := renames[oldName]; ok {
				roles[role] = newName
			}
		}
	}

	return renamed, nil
}

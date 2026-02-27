package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	domainconfig "github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// NewMigrateConfigCmd creates the migrate-config command
func NewMigrateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-config",
		Short: "Migrate treb config from foundry.toml to treb.toml",
		Long: `Migrate treb sender configuration from foundry.toml [profile.*.treb.*] sections
into a dedicated treb.toml file with [ns.<namespace>] structure.

This command will:
1. Read all [profile.*.treb.*] sections from foundry.toml
2. Convert each profile to a [ns.<name>] namespace in treb.toml
3. Show a preview of the generated treb.toml
4. Ask for confirmation before writing

Examples:
  # Interactive migration (shows preview, asks for confirmation)
  treb migrate-config

  # Non-interactive migration (writes without prompts)
  treb migrate-config --non-interactive`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			return runMigrateConfig(app.Config)
		},
	}

	return cmd
}

// runMigrateConfig performs the config migration from foundry.toml to treb.toml.
func runMigrateConfig(cfg *domainconfig.RuntimeConfig) error {
	// Check if there's any treb config in foundry.toml to migrate
	profiles := extractTrebProfiles(cfg.FoundryConfig)
	if len(profiles) == 0 {
		fmt.Println("No treb config found in foundry.toml — nothing to migrate.")
		return nil
	}

	// Generate treb.toml content
	content := generateTrebToml(profiles)

	trebTomlPath := filepath.Join(cfg.ProjectRoot, "treb.toml")
	foundryTomlPath := filepath.Join(cfg.ProjectRoot, "foundry.toml")

	// Check if treb.toml already exists
	if _, err := os.Stat(trebTomlPath); err == nil {
		if cfg.NonInteractive {
			// In non-interactive mode, overwrite
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

// cleanupFoundryToml reads foundry.toml and removes [profile.*.treb.*] sections.
func cleanupFoundryToml(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	cleaned := removeTrebFromFoundryToml(string(data))
	return os.WriteFile(path, []byte(cleaned), 0644)
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

// generateTrebToml generates well-formatted treb.toml content from foundry profiles.
func generateTrebToml(profiles []trebProfile) string {
	var b strings.Builder

	b.WriteString("# treb.toml — Treb sender configuration\n")
	b.WriteString("#\n")
	b.WriteString("# Each [ns.<name>] section defines a namespace with sender configs.\n")
	b.WriteString("# The optional 'profile' field maps to a foundry.toml profile (defaults to namespace name).\n")
	b.WriteString("#\n")
	b.WriteString("# Migrated from foundry.toml [profile.*.treb.*] sections.\n")

	for i, p := range profiles {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("\n")

		// Namespace header
		fmt.Fprintf(&b, "[ns.%s]\n", p.Name)
		// Always set profile explicitly so it's clear where it maps
		fmt.Fprintf(&b, "profile = %q\n", p.Name)

		// Sort sender names for deterministic output
		senderNames := make([]string, 0, len(p.Senders))
		for name := range p.Senders {
			senderNames = append(senderNames, name)
		}
		sort.Strings(senderNames)

		for _, senderName := range senderNames {
			sender := p.Senders[senderName]
			b.WriteString("\n")
			fmt.Fprintf(&b, "[ns.%s.senders.%s]\n", p.Name, senderName)
			writeSenderConfig(&b, sender)
		}
	}

	return b.String()
}

// writeSenderConfig writes a sender config's fields to the builder.
func writeSenderConfig(b *strings.Builder, s domainconfig.SenderConfig) {
	fmt.Fprintf(b, "type = %q\n", string(s.Type))

	if s.Address != "" {
		fmt.Fprintf(b, "address = %q\n", s.Address)
	}
	if s.PrivateKey != "" {
		fmt.Fprintf(b, "private_key = %q\n", s.PrivateKey)
	}
	if s.Safe != "" {
		fmt.Fprintf(b, "safe = %q\n", s.Safe)
	}
	if s.Signer != "" {
		fmt.Fprintf(b, "signer = %q\n", s.Signer)
	}
	if s.DerivationPath != "" {
		fmt.Fprintf(b, "derivation_path = %q\n", s.DerivationPath)
	}
	if s.Governor != "" {
		fmt.Fprintf(b, "governor = %q\n", s.Governor)
	}
	if s.Timelock != "" {
		fmt.Fprintf(b, "timelock = %q\n", s.Timelock)
	}
	if s.Proposer != "" {
		fmt.Fprintf(b, "proposer = %q\n", s.Proposer)
	}
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
			// Skip key-value lines, comments, and blank lines within treb sections
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

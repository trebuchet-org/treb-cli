package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage treb local config",
	Long: `Manage treb local config stored in .treb/config.local.json

The config defines default values for namespace and network
that are used when these flags are not explicitly provided.

Available subcommands:
  config           Show current config
  config set       Set a config value
  config remove    Remove a config value

When run without subcommands, displays the current config.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := configShow(config.NewManager(".")); err != nil {
			checkError(err)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a config value in the .treb file.
Available keys: namespace (ns), network, sender

Examples:
  treb config set namespace production
  treb config set network sepolia`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := configSet(args[0], args[1]); err != nil {
			checkError(err)
		}
	},
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <key>",
	Short: "Remove a config value",
	Long: `Remove a config value from the .treb file.
Removing namespace reverts it to 'default'.
Removing network or sender makes them unspecified (required as flags).

Examples:
  treb config remove namespace
  treb config remove network`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := configRemove(args[0]); err != nil {
			checkError(err)
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configRemoveCmd)
}

// configShow displays all config values
func configShow(manager *config.Manager) error {
	if !manager.Exists() {
		fmt.Printf("❌ No .treb/config.local.json file found\n")
		fmt.Printf("⚠️  Without config, commands require explicit --namespace and --network flags\n")
		return nil
	}

	cfg, err := manager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("📋 Current config:")

	// Show namespace (always has a value)
	fmt.Printf("Namespace: %s\n", cfg.Namespace)

	// Show network (may be empty)
	if cfg.Network != "" {
		fmt.Printf("Network:   %s\n", cfg.Network)
	} else {
		fmt.Printf("Network:   %s\n", "(not set)")
	}

	fmt.Printf("\n📁 config file: %s\n", manager.GetPath())

	return nil
}

// configSet sets a config value
func configSet(key, value string) error {
	manager := config.NewManager(".")

	// Load existing config or create new one
	var cfg *config.Config
	if manager.Exists() {
		var err error
		cfg, err = manager.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		cfg = config.DefaultConfig()
	}

	// Normalize key to lowercase
	key = strings.ToLower(key)

	// Set the value based on key
	switch key {
	case "namespace", "ns":
		cfg.Namespace = value
		fmt.Printf("✅ Set namespace to: %s\n", value)
	case "network":
		cfg.Network = value
		fmt.Printf("✅ Set network to: %s\n", value)
	default:
		return fmt.Errorf("unknown config key: %s\nAvailable keys: namespace, network", key)
	}

	// Save the updated config
	if err := manager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("📁 config saved to: %s\n", manager.GetPath())
	return nil
}

// configRemove removes a config value
func configRemove(key string) error {
	manager := config.NewManager(".")

	// config file must exist to remove values
	if !manager.Exists() {
		return fmt.Errorf("no config file found at %s", manager.GetPath())
	}

	cfg, err := manager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Normalize key to lowercase
	key = strings.ToLower(key)

	// Remove the value based on key
	switch key {
	case "namespace", "ns":
		cfg.Namespace = "default"
		fmt.Printf("✅ Reset namespace to: default\n")
	case "network":
		cfg.Network = ""
		fmt.Printf("✅ Removed network from config (will be required as flag)\n")
	default:
		return fmt.Errorf("unknown config key: %s\nAvailable keys: namespace (ns), network, sender", key)
	}

	// Save the updated config
	if err := manager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("📁 config saved to: %s\n", manager.GetPath())
	return nil
}

// GetconfigDefaults returns config values for command defaults
// Returns empty strings if no config file exists or values are not set
func GetconfigDefaults() (namespace, network string, hasconfig bool) {
	manager := config.NewManager(".")

	// Check if config file exists
	if !manager.Exists() {
		return "default", "", false
	}

	cfg, err := manager.Load()
	if err != nil {
		// config file exists but can't be loaded - return defaults
		return "default", "", false
	}

	return cfg.Namespace, cfg.Network, true
}

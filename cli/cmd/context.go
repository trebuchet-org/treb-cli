package cmd

import (
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/spf13/cobra"
)

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage treb context settings",
	Long: `Manage treb context settings stored in .treb file.

The context defines default values for namespace, network, and sender
that are used when these flags are not explicitly provided.

Available subcommands:
  context           Show current context
  context set       Set a context value
  context remove    Remove a context value

When run without subcommands, displays the current context.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := contextShow(config.NewManager(".")); err != nil {
			checkError(err)
		}
	},
}

var contextSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a context value",
	Long: `Set a context value in the .treb file.
Available keys: namespace (ns), network, sender

Examples:
  treb context set namespace production
  treb context set ns test
  treb context set network sepolia
  treb context set sender safe`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := contextSet(args[0], args[1]); err != nil {
			checkError(err)
		}
	},
}

var contextRemoveCmd = &cobra.Command{
	Use:   "remove <key>",
	Short: "Remove a context value",
	Long: `Remove a context value from the .treb file.
Removing namespace reverts it to 'default'.
Removing network or sender makes them unspecified (required as flags).

Examples:
  treb context remove namespace
  treb context remove network
  treb context remove sender`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := contextRemove(args[0]); err != nil {
			checkError(err)
		}
	},
}

func init() {
	contextCmd.AddCommand(contextSetCmd)
	contextCmd.AddCommand(contextRemoveCmd)
}

// contextShow displays all context values
func contextShow(manager *config.Manager) error {
	if !manager.Exists() {
		fmt.Println("📋 No Context Configuration")
		fmt.Println("===========================")
		fmt.Printf("❌ No .treb file found\n")
		fmt.Printf("💡 Context will be created when you use 'treb context set'\n")
		fmt.Printf("⚠️  Without context, commands require explicit --namespace, --network, and --sender flags\n")
		return nil
	}

	cfg, err := manager.Load()
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	fmt.Println("📋 Current Context:")
	fmt.Println("===================")
	
	// Show namespace (always has a value)
	fmt.Printf("Namespace: %s\n", cfg.Namespace)
	
	// Show network (may be empty)
	if cfg.Network != "" {
		fmt.Printf("Network:   %s\n", cfg.Network)
	} else {
		fmt.Printf("Network:   %s\n", "(not set)")
	}
	
	// Show sender (may be empty)
	if cfg.Sender != "" {
		fmt.Printf("Sender:    %s\n", cfg.Sender)
	} else {
		fmt.Printf("Sender:    %s\n", "(not set)")
	}
	
	fmt.Printf("\n📁 Context file: %s\n", manager.GetPath())

	return nil
}

// contextSet sets a context value
func contextSet(key, value string) error {
	manager := config.NewManager(".")
	
	// Load existing config or create new one
	var cfg *config.Config
	if manager.Exists() {
		var err error
		cfg, err = manager.Load()
		if err != nil {
			return fmt.Errorf("failed to load context: %w", err)
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
	case "sender":
		cfg.Sender = value
		fmt.Printf("✅ Set sender to: %s\n", value)
	default:
		return fmt.Errorf("unknown context key: %s\nAvailable keys: namespace (ns), network, sender", key)
	}
	
	// Save the updated config
	if err := manager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}
	
	fmt.Printf("📁 Context saved to: %s\n", manager.GetPath())
	return nil
}

// contextRemove removes a context value
func contextRemove(key string) error {
	manager := config.NewManager(".")
	
	// Context file must exist to remove values
	if !manager.Exists() {
		return fmt.Errorf("no context file found at %s", manager.GetPath())
	}
	
	cfg, err := manager.Load()
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
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
		fmt.Printf("✅ Removed network from context (will be required as flag)\n")
	case "sender":
		cfg.Sender = ""
		fmt.Printf("✅ Removed sender from context (will be required as flag)\n")
	default:
		return fmt.Errorf("unknown context key: %s\nAvailable keys: namespace (ns), network, sender", key)
	}
	
	// Save the updated config
	if err := manager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}
	
	fmt.Printf("📁 Context saved to: %s\n", manager.GetPath())
	return nil
}

// GetContextDefaults returns context values for command defaults
// Returns empty strings if no context file exists or values are not set
func GetContextDefaults() (namespace, network, sender string, hasContext bool) {
	manager := config.NewManager(".")
	
	// Check if context file exists
	if !manager.Exists() {
		return "default", "", "", false
	}
	
	cfg, err := manager.Load()
	if err != nil {
		// Context file exists but can't be loaded - return defaults
		return "default", "", "", false
	}
	
	return cfg.Namespace, cfg.Network, cfg.Sender, true
}
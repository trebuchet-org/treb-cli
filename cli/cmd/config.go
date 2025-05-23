package cmd

import (
	"fmt"

	"github.com/bogdan/fdeploy/cli/pkg/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage fdeploy configuration",
	Long: `Manage fdeploy configuration settings stored in .fdeploy file.

The configuration file is git-ignored and stores project-specific defaults
for environment, network, and verification settings.

Available configuration keys:
  environment (env)  - Deployment environment: staging, prod
  network           - Network name from foundry.toml
  verify            - Auto-verify contracts: true, false

Examples:
  fdeploy config list                    # Show all configuration
  fdeploy config get network             # Get network setting  
  fdeploy config set env prod            # Set environment to prod
  fdeploy config set network mainnet     # Set default network
  fdeploy config set verify true         # Enable auto-verification`,
	Args: cobra.MinimumNArgs(1),
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	manager := config.NewManager(".")
	
	switch args[0] {
	case "list", "ls":
		return configList(manager)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("get requires a key argument")
		}
		return configGet(manager, args[1])
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("set requires key and value arguments")
		}
		return configSet(manager, args[1], args[2])
	case "init":
		return configInit(manager)
	case "path":
		return configPath(manager)
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

// configList displays all configuration values
func configList(manager *config.Manager) error {
	if !manager.Exists() {
		fmt.Println("📋 No Configuration Found")
		fmt.Println("=========================")
		fmt.Printf("❌ No .fdeploy file found\n")
		fmt.Printf("💡 Run 'fdeploy config init' to create a config file\n")
		fmt.Printf("⚠️  Without configuration, deploy/predict commands require --env and --network flags\n")
		return nil
	}

	cfg, err := manager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("📋 Current Configuration:")
	fmt.Println("========================")
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("Network:     %s\n", cfg.Network)
	fmt.Printf("Verify:      %t\n", cfg.Verify)
	fmt.Printf("\n📁 Config file: %s\n", manager.GetPath())

	return nil
}

// configGet retrieves a specific configuration value
func configGet(manager *config.Manager, key string) error {
	value, err := manager.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get config value: %w", err)
	}

	fmt.Printf("%s\n", value)
	return nil
}

// configSet updates a configuration value
func configSet(manager *config.Manager, key, value string) error {
	if err := manager.Set(key, value); err != nil {
		return fmt.Errorf("failed to set config value: %w", err)
	}

	fmt.Printf("✅ Set %s = %s\n", key, value)
	
	// Show current config after change
	fmt.Println()
	return configList(manager)
}

// configInit creates a new .fdeploy configuration file
func configInit(manager *config.Manager) error {
	if manager.Exists() {
		fmt.Printf("⚠️  Configuration file already exists at %s\n", manager.GetPath())
		fmt.Printf("Current configuration:\n\n")
		return configList(manager)
	}

	// Create with default values
	defaultCfg := config.DefaultConfig()
	if err := manager.Save(defaultCfg); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("✅ Created configuration file: %s\n", manager.GetPath())
	fmt.Printf("🔒 This file is git-ignored and stores project-specific defaults\n\n")
	
	return configList(manager)
}

// configPath shows the path to the configuration file
func configPath(manager *config.Manager) error {
	fmt.Printf("%s\n", manager.GetPath())
	return nil
}

// GetConfiguredDefaults returns configuration values for command defaults
// Returns empty strings if no config file exists, indicating flags should be required
func GetConfiguredDefaults() (env, network string, verify bool, hasConfig bool) {
	manager := config.NewManager(".")
	
	// Check if config file exists
	if !manager.Exists() {
		return "", "", false, false
	}
	
	cfg, err := manager.Load()
	if err != nil {
		// Config file exists but can't be loaded - return empty to require flags
		return "", "", false, false
	}
	
	return cfg.Environment, cfg.Network, cfg.Verify, true
}
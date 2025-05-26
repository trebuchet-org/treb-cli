package cmd

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage treb configuration",
	Long: `Manage treb configuration stored in .treb file.

Available subcommands:
  config          Show current configuration
  config check    Validate configuration
  config set      Set a configuration value
  config init     Initialize configuration file

When run without subcommands, displays the current configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := configList(config.NewManager(".")); err != nil {
			checkError(err)
		}
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate configuration",
	Long: `Check that all required configuration is present and valid.
Verifies RPC endpoints, API keys, and deployer settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := configCheck(); err != nil {
			checkError(err)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value in the .treb file.
Available keys: environment, network, verify

Examples:
  treb config set environment production
  treb config set network sepolia
  treb config set verify true`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := configSet(args[0], args[1]); err != nil {
			checkError(err)
		}
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new .treb configuration file with default values.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := configInit(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	configCmd.AddCommand(configCheckCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
}

func configCheck() error {
	// Load and validate deploy configuration
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load deploy configuration: %w", err)
	}

	fmt.Println("Configuration Check:")
	fmt.Println("===================")

	// Check default environment
	env := "default"
	if err := deployConfig.Validate(env); err != nil {
		fmt.Printf("❌ Default environment validation failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ Default environment configuration is valid\n")

	// Show environment info
	envConfig, err := deployConfig.GetEnvironmentConfig(env)
	if err != nil {
		return fmt.Errorf("failed to get environment config: %w", err)
	}

	fmt.Printf("   Deployer Type: %s\n", envConfig.Deployer.Type)
	if envConfig.Deployer.Type == "safe" {
		fmt.Printf("   Safe Address: %s\n", envConfig.Deployer.Safe)
	}

	return nil
}

// configList displays all configuration values
func configList(manager *config.Manager) error {
	if !manager.Exists() {
		fmt.Println("📋 No Configuration Found")
		fmt.Println("=========================")
		fmt.Printf("❌ No .treb file found\n")
		fmt.Printf("💡 Run 'treb config init' to create a config file\n")
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

// configSet sets a configuration value
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
		cfg = &config.Config{}
	}
	
	// Set the value based on key
	switch key {
	case "environment", "env":
		cfg.Environment = value
		fmt.Printf("✅ Set environment to: %s\n", value)
	case "network":
		cfg.Network = value
		fmt.Printf("✅ Set network to: %s\n", value)
	case "verify":
		switch value {
		case "true", "yes", "1":
			cfg.Verify = true
			fmt.Printf("✅ Set verify to: true\n")
		case "false", "no", "0":
			cfg.Verify = false
			fmt.Printf("✅ Set verify to: false\n")
		default:
			return fmt.Errorf("invalid value for verify: %s (use true/false)", value)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s\nAvailable keys: environment, network, verify", key)
	}
	
	// Save the updated config
	if err := manager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	fmt.Printf("📁 Config saved to: %s\n", manager.GetPath())
	return nil
}

// configInit initializes a new configuration file
func configInit() error {
	manager := config.NewManager(".")
	
	// Check if config already exists
	if manager.Exists() {
		return fmt.Errorf(".treb file already exists at %s", manager.GetPath())
	}
	
	// Create default config
	cfg := &config.Config{
		Environment: "default",
		Network:     "",
		Verify:      false,
	}
	
	// Save the config
	if err := manager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	fmt.Printf("✅ Initialized configuration file: %s\n", manager.GetPath())
	fmt.Println("\nDefault configuration:")
	fmt.Printf("  Environment: %s\n", cfg.Environment)
	fmt.Printf("  Network:     %s (not set)\n", cfg.Network)
	fmt.Printf("  Verify:      %t\n", cfg.Verify)
	fmt.Println("\nUse 'treb config set <key> <value>' to modify these values")
	
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
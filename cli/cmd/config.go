package cmd

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long: `Display resolved configuration from environment and .treb file.
Shows environment, network, and deployer settings.`,
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

func init() {
	configCmd.AddCommand(configCheckCmd)
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
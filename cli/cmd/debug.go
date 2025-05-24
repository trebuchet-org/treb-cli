package cmd

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug utilities",
	Long:  `Debug utilities for troubleshooting treb configuration and environment.`,
}

var debugConfigCmd = &cobra.Command{
	Use:   "config [environment]",
	Short: "Show resolved deployment configuration",
	Long:  `Show the resolved deployment configuration for a specific environment, including environment variables that would be passed to forge scripts.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		env := args[0]
		if err := showDeployConfig(env); err != nil {
			checkError(err)
		}
	},
}

func init() {
	debugCmd.AddCommand(debugConfigCmd)
}

func showDeployConfig(env string) error {
	// Load deploy configuration
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load deploy configuration: %w", err)
	}

	// Validate configuration
	if err := deployConfig.Validate(env); err != nil {
		return fmt.Errorf("invalid deploy configuration for environment '%s': %w", env, err)
	}

	// Generate environment variables
	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}

	fmt.Printf("🔧 Deploy Configuration for '%s' environment:\n\n", env)

	// Show the config structure
	envConfig, _ := deployConfig.GetEnvironmentConfig(env)
	fmt.Printf("Deployer Type: %s\n", envConfig.Deployer.Type)
	
	if envConfig.Deployer.Type == "safe" {
		fmt.Printf("Safe Address: %s\n", envConfig.Deployer.Safe)
		if envConfig.Deployer.Proposer != nil {
			fmt.Printf("Proposer Type: %s\n", envConfig.Deployer.Proposer.Type)
		}
	}

	fmt.Printf("\n📋 Environment Variables (passed to forge scripts):\n")
	for key, value := range envVars {
		// Mask private keys for security
		if key == "DEPLOYER_PRIVATE_KEY" || key == "PROPOSER_PRIVATE_KEY" {
			if len(value) > 10 {
				fmt.Printf("  %s=%s...%s\n", key, value[:6], value[len(value)-4:])
			} else {
				fmt.Printf("  %s=***\n", key)
			}
		} else {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}

	return nil
}
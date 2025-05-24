package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
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

var debugFixCompilerCmd = &cobra.Command{
	Use:   "fix-compiler-version",
	Short: "Fix compiler versions in deployment registry",
	Long: `Fix compiler versions in the deployment registry by reading the actual compiler version from foundry artifacts.

This command iterates through all deployments and updates the compiler version metadata
by reading from the contract artifacts in the out/ directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := fixCompilerVersions(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	debugCmd.AddCommand(debugConfigCmd)
	debugCmd.AddCommand(debugFixCompilerCmd)
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

func fixCompilerVersions() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Get all deployments
	allDeployments := registryManager.GetAllDeployments()
	
	if len(allDeployments) == 0 {
		color.New(color.FgYellow).Println("No deployments found in registry.")
		return nil
	}

	color.New(color.FgCyan, color.Bold).Printf("Fixing compiler versions for %d deployments...\n\n", len(allDeployments))

	fixedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, deployment := range allDeployments {
		// Get current compiler version
		currentVersion := deployment.Entry.Metadata.Compiler
		
		// Get the correct version from artifact
		correctVersion := registryManager.GetCompilerVersionFromArtifact(deployment.Entry.ContractName)
		
		if correctVersion == "" {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - No artifact found\n", 
				deployment.NetworkName, deployment.Entry.Environment, deployment.Entry.GetDisplayName())
			errorCount++
			continue
		}

		if currentVersion == correctVersion {
			color.New(color.FgGreen).Printf("  ✓ %s/%s/%s - Already correct (%s)\n", 
				deployment.NetworkName, deployment.Entry.Environment, deployment.Entry.GetDisplayName(), currentVersion)
			skippedCount++
			continue
		}

		// Update the compiler version
		deployment.Entry.Metadata.Compiler = correctVersion
		
		// Save the updated deployment
		chainIDUint, err := strconv.ParseUint(deployment.ChainID, 10, 64)
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - Invalid chain ID: %s\n", 
				deployment.NetworkName, deployment.Entry.Environment, deployment.Entry.GetDisplayName(), deployment.ChainID)
			errorCount++
			continue
		}
		
		key := strings.ToLower(deployment.Address.Hex())
		err = registryManager.UpdateDeploymentByAddress(key, deployment.Entry, chainIDUint)
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - Failed to save: %v\n", 
				deployment.NetworkName, deployment.Entry.Environment, deployment.Entry.GetDisplayName(), err)
			errorCount++
			continue
		}

		color.New(color.FgCyan).Printf("  → %s/%s/%s - Updated: %s → %s\n", 
			deployment.NetworkName, deployment.Entry.Environment, deployment.Entry.GetDisplayName(), 
			currentVersion, correctVersion)
		fixedCount++
	}

	fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
	color.New(color.FgGreen, color.Bold).Printf("✓ Fixed: %d\n", fixedCount)
	color.New(color.FgYellow, color.Bold).Printf("⚪ Skipped (already correct): %d\n", skippedCount)
	if errorCount > 0 {
		color.New(color.FgRed, color.Bold).Printf("✗ Errors: %d\n", errorCount)
	}
	
	if fixedCount > 0 {
		fmt.Println("\nRegistry has been updated with correct compiler versions.")
	}

	return nil
}
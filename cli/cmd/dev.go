//go:build dev
// +build dev

package cmd

import (
	"fmt"
	"os"
	// "strconv" // Used in commented v1 registry functions
	"strings"

	// "github.com/fatih/color" // Used in commented v1 registry functions
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	// "github.com/trebuchet-org/treb-cli/cli/pkg/registry" // v1 registry removed
	// "github.com/trebuchet-org/treb-cli/cli/pkg/types" // Used in commented v1 registry functions
	"github.com/trebuchet-org/treb-cli/cli/pkg/dev"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development utilities",
	Long:  `Development utilities for troubleshooting treb configuration and environment.`,
}

var debugConfigCmd = &cobra.Command{
	Use:   "config [namespace] [sender]",
	Short: "Show resolved deployment configuration",
	Long:  `Show the resolved deployment configuration for a specific namespace and sender, including environment variables that would be passed to forge scripts.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		sender := args[1]
		if err := showDeployConfig(namespace, sender); err != nil {
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
		fmt.Println("This command is obsolete - it was for v1 registry which has been removed")
		// if err := fixCompilerVersions(); err != nil {
		// 	checkError(err)
		// }
	},
}

var debugFixScriptPathCmd = &cobra.Command{
	Use:   "fix-script-path",
	Short: "Fix script paths in deployment registry",
	Long: `Fix deployment script paths in the deployment registry retroactively.

This command iterates through all deployments and adds the script_path metadata
field based on the contract name and type (implementation or proxy).`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("This command is obsolete - it was for v1 registry which has been removed")
		// if err := fixScriptPaths(); err != nil {
		// 	checkError(err)
		// }
	},
}

var devAnvilCmd = &cobra.Command{
	Use:   "anvil",
	Short: "Manage local anvil node",
	Long:  `Manage a local anvil node for testing deployments with CreateX factory automatically deployed.`,
}

var debugAnvilStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start local anvil node",
	Long:  `Start a local anvil node with CreateX factory deployed. Fails if already running.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := dev.StartAnvil(); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop local anvil node",
	Long:  `Stop the local anvil node if running.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := dev.StopAnvil(); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart local anvil node",
	Long:  `Restart the local anvil node with CreateX factory deployed.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := dev.RestartAnvil(); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show anvil logs",
	Long:  `Show logs from the local anvil node.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := dev.ShowAnvilLogs(); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show anvil status",
	Long:  `Show status of the local anvil node.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := dev.ShowAnvilStatus(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	devCmd.AddCommand(debugConfigCmd)
	devCmd.AddCommand(debugFixCompilerCmd)
	devCmd.AddCommand(debugFixScriptPathCmd)
	
	// Add anvil command with subcommands
	devAnvilCmd.AddCommand(debugAnvilStartCmd)
	devAnvilCmd.AddCommand(debugAnvilStopCmd)
	devAnvilCmd.AddCommand(debugAnvilRestartCmd)
	devAnvilCmd.AddCommand(debugAnvilLogsCmd)
	devAnvilCmd.AddCommand(debugAnvilStatusCmd)
	devCmd.AddCommand(devAnvilCmd)
}

func showDeployConfig(namespace, sender string) error {
	// Load deploy configuration
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load deploy configuration: %w", err)
	}

	// Validate sender configuration
	if err := deployConfig.ValidateSender("treb", sender); err != nil {
		return fmt.Errorf("invalid sender configuration '%s': %w", sender, err)
	}

	// Generate environment variables
	envVars, err := deployConfig.GenerateEnvVars(namespace)
	if err != nil {
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}
	
	// Generate sender environment variables
	senderEnvVars, err := deployConfig.GenerateSenderEnvVars("treb", sender)
	if err != nil {
		return fmt.Errorf("failed to generate sender environment variables: %w", err)
	}
	
	// Merge sender env vars
	for k, v := range senderEnvVars {
		envVars[k] = v
	}

	fmt.Printf("🔧 Deploy Configuration for namespace '%s' with sender '%s':\n\n", namespace, sender)

	// Show the sender config structure
	senderConfig, _ := deployConfig.GetSender("treb", sender)
	fmt.Printf("Sender Type: %s\n", senderConfig.Type)

	if senderConfig.Type == "safe" {
		fmt.Printf("Safe Address: %s\n", senderConfig.Safe)
		fmt.Printf("Signer: %s\n", senderConfig.Signer)
	} else if senderConfig.Type == "ledger" {
		fmt.Printf("Derivation Path: %s\n", senderConfig.DerivationPath)
	}

	fmt.Printf("\n📋 Environment Variables (passed to forge scripts):\n")
	for key, value := range envVars {
		// Mask private keys for security
		if key == "SENDER_PRIVATE_KEY" {
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

// OBSOLETE: This function was for v1 registry which has been removed
/*
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

		// Get the correct version from artifact, using contract_path for proxies
		var correctVersion string
		// TODO: Implement artifact-based compiler version checking
		// This needs to be updated to work with the new ContractInfo structure
		correctVersion = "unknown"

		if correctVersion == "" {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - No artifact found\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName())
			errorCount++
			continue
		}

		if currentVersion == correctVersion {
			color.New(color.FgGreen).Printf("  ✓ %s/%s/%s - Already correct (%s)\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(), currentVersion)
			skippedCount++
			continue
		}

		// Update the compiler version
		deployment.Entry.Metadata.Compiler = correctVersion

		// Save the updated deployment
		chainIDUint, err := strconv.ParseUint(deployment.ChainID, 10, 64)
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - Invalid chain ID: %s\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(), deployment.ChainID)
			errorCount++
			continue
		}

		err = registryManager.UpdateDeployment(chainIDUint, deployment.Entry)
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - Failed to save: %v\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(), err)
			errorCount++
			continue
		}

		color.New(color.FgCyan).Printf("  → %s/%s/%s - Updated: %s → %s\n",
			deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(),
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
*/

// OBSOLETE: This function was for v1 registry which has been removed
/*
func fixScriptPaths() error {
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

	color.New(color.FgCyan, color.Bold).Printf("Fixing script paths for %d deployments...\n\n", len(allDeployments))

	fixedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, deployment := range allDeployments {
		// Check if script path already exists
		if deployment.Entry.Metadata.ScriptPath != "" {
			color.New(color.FgGreen).Printf("  ✓ %s/%s/%s - Already has script path (%s)\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(),
				deployment.Entry.Metadata.ScriptPath)
			skippedCount++
			continue
		}

		// Determine the script path based on contract name
		var scriptPath string
		if deployment.Entry.Type == types.ProxyDeployment {
			// For proxies, the contract name is already the proxy name
			scriptPath = fmt.Sprintf("script/deploy/Deploy%s.s.sol", deployment.Entry.ContractName)
		} else {
			// For implementations
			scriptPath = fmt.Sprintf("script/deploy/Deploy%s.s.sol", deployment.Entry.ContractName)
		}

		// Update the script path
		deployment.Entry.Metadata.ScriptPath = scriptPath

		// Save the updated deployment
		chainIDUint, err := strconv.ParseUint(deployment.ChainID, 10, 64)
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - Invalid chain ID: %s\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(), deployment.ChainID)
			errorCount++
			continue
		}

		err = registryManager.UpdateDeployment(chainIDUint, deployment.Entry)
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s/%s/%s - Failed to save: %v\n",
				deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(), err)
			errorCount++
			continue
		}

		color.New(color.FgCyan).Printf("  → %s/%s/%s - Added script path: %s\n",
			deployment.NetworkName, deployment.Entry.Namespace, deployment.Entry.GetDisplayName(),
			scriptPath)
		fixedCount++
	}

	fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
	color.New(color.FgGreen, color.Bold).Printf("✓ Fixed: %d\n", fixedCount)
	color.New(color.FgYellow, color.Bold).Printf("⚪ Skipped (already has script path): %d\n", skippedCount)
	if errorCount > 0 {
		color.New(color.FgRed, color.Bold).Printf("✗ Errors: %d\n", errorCount)
	}

	if fixedCount > 0 {
		fmt.Println("\nRegistry has been updated with script paths.")
	}

	return nil
}
*/

// isValidContractPath checks if a contract path points to a file that exists on disk
func isValidContractPath(contractPath string) bool {
	if contractPath == "" {
		return false
	}

	// Extract file path from contract path (format: ./path/to/Contract.sol:Contract)
	parts := strings.Split(contractPath, ":")
	if len(parts) != 2 {
		return false
	}

	filePath := strings.TrimPrefix(parts[0], "./")

	// Check if file exists
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// extractContractNameFromPath extracts contract name from contract path
// E.g., "./lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol:TransparentUpgradeableProxy" -> "TransparentUpgradeableProxy"
func extractContractNameFromPath(contractPath string) string {
	// Contract path format: ./path/to/Contract.sol:ContractName
	parts := strings.Split(contractPath, ":")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

//go:build dev
// +build dev

package cmd

import (
	"fmt"
	"os"

	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/display"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/runner"
	"github.com/trebuchet-org/treb-cli/cli/pkg/submodule"
)

var runCmd = &cobra.Command{
	Use:   "run <script-file>",
	Short: "Run a Foundry script with treb infrastructure",
	Long: `Run a Foundry script with automatic sender configuration and event tracking.

This command executes Foundry scripts while:
- Automatically configuring senders based on your treb configuration
- Parsing deployment events from script execution
- Recording deployments in the registry
- Supporting multiple sender types (private key, Safe, hardware wallet)
- Automatically parsing and validating script parameters from natspec

Script Parameters:
Scripts can define parameters using custom natspec tags:
  /**
   * @custom:env {string} label Label for the deployment
   * @custom:env {address} owner Owner address for the contract
   * @custom:env {string:optional} description Optional description
   */
  function run() public { ... }

Supported parameter types:
- Base types: string, address, uint256, int256, bytes32, bytes
- Meta types: sender (sender ID), deployment (contract reference), artifact (contract name)

Examples:
  # Run a deployment script
  treb run script/deploy/DeployCounter.s.sol

  # Run with parameters via --env flags
  treb run script/deploy/DeployCounter.s.sol --env label=v2 --env owner=0x123...

  # Run with deployment reference
  treb run script/deploy/DeployProxy.s.sol --env implementation=Counter:v1

  # Run with dry-run to see what would happen
  treb run script/deploy/DeployCounter.s.sol --dry-run

  # Run with debug output
  treb run script/deploy/DeployCounter.s.sol --debug

  # Run with specific network and profile
  treb run script/deploy/DeployCounter.s.sol --network sepolia --profile production`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptPath := args[0]

		// Check and update treb-sol if needed (unless disabled)
		skipTrebSolUpdate, _ := cmd.Flags().GetBool("skip-treb-sol-update")
		if !skipTrebSolUpdate {
			trebSolManager := submodule.NewTrebSolManager(".")
			if trebSolManager.IsTrebSolInstalled() {
				// Check for updates in a non-blocking way
				if err := trebSolManager.CheckAndUpdate(false); err != nil {
					// This should not happen as CheckAndUpdate handles errors gracefully
					// but if it does, we just continue with the existing version
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to check treb-sol updates: %v\n", err)
				}
			}
		}

		// Get flags
		network, _ := cmd.Flags().GetString("network")
		namespace, _ := cmd.Flags().GetString("namespace")
		envVars, _ := cmd.Flags().GetStringSlice("env")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		debug, _ := cmd.Flags().GetBool("debug")
		debugJSON, _ := cmd.Flags().GetBool("debug-json")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Parse environment variables
		parsedEnvVars := make(map[string]string)
		for _, envVar := range envVars {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				checkError(fmt.Errorf("invalid env var format: %s (expected KEY=VALUE)", envVar))
			}
			parsedEnvVars[parts[0]] = parts[1]
		}

		// Create script runner
		scriptRunner, err := runner.NewScriptRunner(".", !IsNonInteractive())
		if err != nil {
			checkError(err)
		}

		// Create run configuration
		runConfig := &runner.RunConfig{
			ScriptPath:     scriptPath,
			Network:        network,
			Namespace:      namespace,
			EnvVars:        parsedEnvVars,
			DryRun:         dryRun,
			Debug:          debug,
			DebugJSON:      debugJSON,
			Verbose:        verbose,
			NonInteractive: IsNonInteractive(),
			WorkDir:        ".",
		}

		// Run the script
		result, err := scriptRunner.Run(runConfig)
		if err != nil {
			checkError(err)
		}

		if !result.Success {
			os.Exit(1)
		}

		// In debug mode, the raw output is already saved
		if debug || debugJSON {
			fmt.Printf("\nDebug output saved to: debug-output.json\n")
			fmt.Printf("Raw output size: %d bytes\n", len(result.RawOutput))
		}

		display.PrintSuccessMessage("Script execution completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Network flag
	runCmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")

	runCmd.Flags().String("namespace", "", "Namespace to use (defaults to current context namespace) [also sets foundry profile]")

	// Environment variables flag
	runCmd.Flags().StringSliceP("env", "e", []string{}, "Set environment variables for the script (format: KEY=VALUE, can be used multiple times)")

	// Dry run flag
	runCmd.Flags().Bool("dry-run", false, "Perform a dry run without broadcasting transactions")

	// Debug flags
	runCmd.Flags().Bool("debug", false, "Enable debug mode (shows forge output and saves to file)")
	runCmd.Flags().Bool("debug-json", false, "Enable JSON debug mode (shows raw JSON output)")
	runCmd.Flags().BoolP("verbose", "v", false, "Show extra detailed information for events and transactions")

	// Submodule management flags
	runCmd.Flags().Bool("skip-treb-sol-update", false, "Skip automatic treb-sol update check")
}

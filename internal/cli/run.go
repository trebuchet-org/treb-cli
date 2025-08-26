package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewRunCmd creates the run command using the new architecture
func NewRunCmd() *cobra.Command {
	var (
		envVars   []string
		dryRun    bool
		debug     bool
		debugJSON bool
	)

	cmd := &cobra.Command{
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
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context (v2 usecase wiring)
			deploymentScriptRef := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			if app.Config.Network == nil {
				return fmt.Errorf("no active network set in config, --network flag is required")
			}

			// Parse environment variables (KEY=VALUE)
			parsedEnvVars := make(map[string]string)
			for _, envVar := range envVars {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid env var format: %s (expected key=value)", envVar)
				}
				parsedEnvVars[parts[0]] = parts[1]
			}

			params := usecase.RunScriptParams{
				ScriptRef:  deploymentScriptRef,
				Parameters: parsedEnvVars,
				DryRun:     dryRun,
				Debug:      debug,
				DebugJSON:  debugJSON,
			}
			result, err := app.RunScript.Run(cmd.Context(), params)
			if err != nil {
				return err
			}
			if result.Error != nil {
				return result.Error
			}
			if !result.Success {
				return fmt.Errorf("script execution failed")
			}

			if err := app.ScriptRenderer.RenderExecution(result); err != nil {
				return err
			}

			// Final success line like v1
			fmt.Printf("\x1b[32m✓ Script execution completed successfully\x1b[0m\n")
			return nil
		},
	}

	// Flags
	cmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")
	cmd.Flags().StringP("namespace", "s", "", "Namespace to use (defaults to current context namespace) [also sets foundry profile]")
	cmd.Flags().StringSliceVarP(&envVars, "env", "e", []string{}, "Set environment variables for the script (format: KEY=VALUE, can be used multiple times)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without broadcasting transactions")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode (shows forge output and saves to file)")
	cmd.Flags().BoolVar(&debugJSON, "debug-json", false, "Enable JSON debug mode (shows raw JSON output)")
	cmd.Flags().BoolP("verbose", "v", false, "Show extra detailed information for events and transactions")

	return cmd
}

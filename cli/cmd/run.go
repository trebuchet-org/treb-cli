package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	netpkg "github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script"
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

		resolver := resolvers.NewContext(".", !IsNonInteractive())
		scriptContract, err := resolver.ResolveContract(scriptPath, contracts.ScriptFilter())
		if err != nil {
			checkError(fmt.Errorf("failed to resolve contract: %w", err))
		}

		// Get flags
		network, _ := cmd.Flags().GetString("network")
		profile, _ := cmd.Flags().GetString("profile")
		namespace, _ := cmd.Flags().GetString("namespace")
		envVars, _ := cmd.Flags().GetStringSlice("env")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		debug, _ := cmd.Flags().GetBool("debug")
		debugJSON, _ := cmd.Flags().GetBool("debug-json")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Default network
		if network == "" {
			network = os.Getenv("DEPLOYMENT_NETWORK")
			if network == "" {
				network = "local"
			}
		}

		// Default namespace from context if not provided
		if namespace == "" {
			// Try to load from context
			configManager := config.NewManager(".")
			if cfg, err := configManager.Load(); err == nil {
				namespace = cfg.Namespace
			}
			// Ensure we have a default namespace if still empty
			if namespace == "" {
				namespace = "default"
			}
		}

		// Parse environment variables
		parsedEnvVars := make(map[string]string)
		for _, envVar := range envVars {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				checkError(fmt.Errorf("invalid env var format: %s (expected KEY=VALUE)", envVar))
			}
			parsedEnvVars[parts[0]] = parts[1]
		}

		// Resolve network info to get chain ID from RPC
		networkResolver := netpkg.NewResolver(".")
		networkInfo, err := networkResolver.ResolveNetwork(network)
		if err != nil {
			checkError(fmt.Errorf("failed to resolve network: %w", err))
		}

		// Load treb config for parameter resolution
		fullConfig, err := config.LoadTrebConfig(".")
		if err != nil {
			checkError(fmt.Errorf("failed to load treb config: %w", err))
		}

		trebConfig, err := fullConfig.GetProfileTrebConfig(profile)
		if err != nil {
			checkError(fmt.Errorf("failed to get profile config: %w", err))
		}

		// Parse script parameters from natspec if artifact is available
		if scriptContract.Artifact != nil {
			paramParser := script.NewParameterParser()
			params, err := paramParser.ParseFromArtifact(scriptContract.Artifact)
			if err != nil {
				checkError(fmt.Errorf("failed to parse script parameters: %w", err))
			}

			if len(params) > 0 {
				// Create parameter resolver
				paramResolver, err := script.NewParameterResolver(".", trebConfig, namespace, network, networkInfo.ChainID, !IsNonInteractive())
				if err != nil {
					checkError(fmt.Errorf("failed to create parameter resolver: %w", err))
				}

				// Resolve all parameters
				resolvedEnvVars, err := paramResolver.ResolveAll(params, parsedEnvVars)
				if err != nil {
					if IsNonInteractive() {
						checkError(fmt.Errorf("parameter resolution failed: %w", err))
					}
					// In interactive mode, we'll prompt for missing values
				}

				// Ensure we have a valid map even if resolution had errors
				if resolvedEnvVars == nil {
					resolvedEnvVars = make(map[string]string)
				}

				// Check for missing required parameters
				var missingRequired []script.Parameter
				for _, param := range params {
					if !param.Optional && resolvedEnvVars[param.Name] == "" {
						missingRequired = append(missingRequired, param)
					}
				}

				// Handle missing parameters
				if len(missingRequired) > 0 {
					if IsNonInteractive() {
						var missingNames []string
						for _, p := range missingRequired {
							missingNames = append(missingNames, p.Name)
						}
						checkError(fmt.Errorf("missing required parameters: %s", strings.Join(missingNames, ", ")))
					} else {
						// Interactive mode: prompt for missing values
						fmt.Println("The script supports the following parameters:")
						for _, p := range params {
							var status, nameColor string
							if resolvedEnvVars[p.Name] != "" {
								// Present - green
								status = color.GreenString("✓")
								nameColor = color.GreenString(p.Name)
							} else if p.Optional {
								// Optional and missing - yellow
								status = color.YellowString("○")
								nameColor = color.YellowString(p.Name)
							} else {
								// Required and missing - red
								status = color.RedString("✗")
								nameColor = color.RedString(p.Name)
							}
							fmt.Printf("  %s %s (%s): %s\n", status, nameColor, p.Type, p.Description)
						}
						fmt.Println("\nMissing one or more required parameters.")

						prompter := script.NewParameterPrompter(paramResolver)
						promptedVars, err := prompter.PromptForMissingParameters(params, resolvedEnvVars)
						if err != nil {
							checkError(fmt.Errorf("failed to prompt for parameters: %w", err))
						}

						// Re-resolve with prompted values
						resolvedEnvVars, err = paramResolver.ResolveAll(params, promptedVars)
						if err != nil {
							checkError(fmt.Errorf("failed to resolve prompted parameters: %w", err))
						}
					}
				}

				// Update parsedEnvVars with resolved values
				for k, v := range resolvedEnvVars {
					if v != "" {
						parsedEnvVars[k] = v
					}
				}
			}
		}

		// Create script executor
		executor := script.NewExecutor(".", networkInfo)

		// Initialize indexer for contract identification
		indexer, err := contracts.GetGlobalIndexer(".")
		if err != nil {
			fmt.Printf("Warning: Could not initialize contract indexer: %v\n", err)
			indexer = nil
		}

		// Run the script
		script.PrintDeploymentBanner(fmt.Sprintf("Running script: %s", filepath.Base(scriptContract.Path)), network, profile)

		// Debug mode always implies dry run to prevent Safe transaction creation
		if debug || debugJSON {
			dryRun = true
			fmt.Println("Mode: Debug (dry run, no broadcast)")
		} else if dryRun {
			fmt.Println("Mode: Dry run (no broadcast)")
		}

		opts := script.RunOptions{
			ScriptPath: scriptContract.Path,
			Network:    network,
			Profile:    profile,
			Namespace:  namespace,
			EnvVars:    parsedEnvVars,
			DryRun:     dryRun,
			Debug:      debug || debugJSON,
			DebugJSON:  debugJSON,
		}

		result, err := executor.Run(opts)
		checkError(err)

		if !result.Success {
			script.PrintErrorMessage("Script execution failed")
			os.Exit(1)
		}

		// In debug mode, the raw output is already saved
		if debug || debugJSON {
			fmt.Printf("\nDebug output saved to: debug-output.json\n")
			fmt.Printf("Raw output size: %d bytes\n", len(result.RawOutput))
		}

		// Report all parsed events using enhanced display
		if len(result.AllEvents) > 0 || len(result.Logs) > 0 {
			// Use enhanced display system for better formatting and phase tracking
			enhancedDisplay := script.NewEnhancedEventDisplay(indexer)
			enhancedDisplay.SetVerbose(verbose)

			// Load sender configs to improve address display
			if senderConfigs, err := script.BuildSenderConfigs(trebConfig); err == nil {
				enhancedDisplay.SetSenderConfigs(senderConfigs)
			}

			// Enable registry-based ABI resolution for better transaction decoding
			if manager, err := registry.NewManager("."); err == nil {
				enhancedDisplay.SetRegistryResolver(manager, networkInfo.ChainID)
			}

			// Display logs first
			enhancedDisplay.DisplayLogs(result.Logs)

			// Then display events
			enhancedDisplay.ProcessEvents(result.AllEvents)

			// Update registry if not dry run
			if !dryRun {
				// Create script updater and build registry update
				scriptUpdater := registry.NewScriptUpdater(indexer)
				registryUpdate := scriptUpdater.BuildRegistryUpdate(
					result.AllEvents,
					namespace,
					networkInfo.ChainID,
					network,
					scriptPath,
				)

				// Enrich with broadcast data if available
				if result.BroadcastPath != "" {
					enricher := registry.NewBroadcastEnricher()
					if err := enricher.EnrichFromBroadcastFile(registryUpdate, result.BroadcastPath); err != nil {
						script.PrintWarningMessage(fmt.Sprintf("Failed to enrich from broadcast: %v", err))
					}
				}

				// Apply registry update
				manager, err := registry.NewManager(".")
				if err != nil {
					script.PrintErrorMessage(fmt.Sprintf("Failed to create registry manager: %v", err))
				} else if err := registryUpdate.Apply(manager); err != nil {
					script.PrintErrorMessage(fmt.Sprintf("Failed to apply registry update: %v", err))
				} else {
					script.PrintSuccessMessage(fmt.Sprintf("Updated registry for %s network in namespace %s", network, namespace))
				}
			}

			// Track and report proxy relationships
			proxyTracker := script.NewProxyTracker()
			proxyTracker.ProcessEvents(result.AllEvents)
			proxyTracker.PrintProxyRelationships()
		} else if !dryRun {
			script.PrintWarningMessage("No events detected")
		}

		// Report broadcast file if found
		if result.BroadcastPath != "" {
			fmt.Printf("\nBroadcast file: %s\n", result.BroadcastPath)
		}

		script.PrintSuccessMessage("Script execution completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Network flag
	runCmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")

	// Profile flag
	runCmd.Flags().StringP("profile", "p", "default", "Configuration profile to use")

	// Namespace flag
	runCmd.Flags().String("namespace", "", "Namespace to use (defaults to current context namespace)")

	// Environment variables flag
	runCmd.Flags().StringSlice("env", []string{}, "Set environment variables for the script (format: KEY=VALUE, can be used multiple times)")

	// Dry run flag
	runCmd.Flags().Bool("dry-run", false, "Perform a dry run without broadcasting transactions")

	// Debug flags
	runCmd.Flags().Bool("debug", false, "Enable debug mode (shows forge output and saves to file)")
	runCmd.Flags().Bool("debug-json", false, "Enable JSON debug mode (shows raw JSON output)")
	runCmd.Flags().BoolP("verbose", "v", false, "Show extra detailed information for events and transactions")
}

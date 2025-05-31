package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	netpkg "github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
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

Examples:
  # Run a deployment script
  treb run script/deploy/DeployCounter.s.sol

  # Run with dry-run to see what would happen
  treb run script/deploy/DeployCounter.s.sol --dry-run

  # Run with debug output
  treb run script/deploy/DeployCounter.s.sol --debug

  # Run with specific network and profile
  treb run script/deploy/DeployCounter.s.sol --network sepolia --profile production`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptPath := args[0]

		// Check if script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			checkError(fmt.Errorf("script file not found: %s", scriptPath))
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

		// Create script executor
		executor := script.NewExecutor(".", networkInfo)

		// Initialize indexer for contract identification
		indexer, err := contracts.GetGlobalIndexer(".")
		if err != nil {
			fmt.Printf("Warning: Could not initialize contract indexer: %v\n", err)
			indexer = nil
		}

		// Run the script
		script.PrintDeploymentBanner(fmt.Sprintf("Running script: %s", filepath.Base(scriptPath)), network, profile)
		
		// Debug mode always implies dry run to prevent Safe transaction creation
		if debug || debugJSON {
			dryRun = true
			fmt.Println("Mode: Debug (dry run, no broadcast)")
		} else if dryRun {
			fmt.Println("Mode: Dry run (no broadcast)")
		}

		opts := script.RunOptions{
			ScriptPath: scriptPath,
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
		if len(result.AllEvents) > 0 {
			// Use enhanced display system for better formatting and phase tracking
			enhancedDisplay := script.NewEnhancedEventDisplay(indexer)
			enhancedDisplay.SetVerbose(verbose)
			
			// Load sender configs to improve address display
			if trebConfig, err := config.LoadTrebConfig("."); err == nil {
				if profileConfig, err := trebConfig.GetProfileTrebConfig(profile); err == nil {
					if senderConfigs, err := script.BuildSenderConfigs(profileConfig); err == nil {
						enhancedDisplay.SetSenderConfigs(senderConfigs)
					}
				}
			}
			
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

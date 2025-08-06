package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/orchestration"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/display"
)

var orchestrateCmd = &cobra.Command{
	Use:   "orchestrate <orchestration-file>",
	Short: "Execute orchestrated deployments from a YAML configuration",
	Long: `Execute multiple deployment scripts in dependency order based on a YAML configuration file.

The orchestration file defines components, their deployment scripts, dependencies, and environment variables.
Treb will build a dependency graph and execute scripts in the correct order.

Example orchestration file (deploy.yaml):
  group: Mento Protocol
  components:
    Broker:
      script: DeployBroker
    Tokens:
      script: DeployTokens
      deps: 
        - Broker
    Reserve:
      script: DeployReserve
      deps: 
        - Tokens
      env:
        INITIAL_BALANCE: "1000000"
    SortedOracles:
      script: DeploySortedOracles
      deps:
        - Reserve

This will execute: Broker → Tokens → Reserve → SortedOracles`,
	Example: `  # Execute orchestration from YAML file
  treb orchestrate deploy.yaml

  # Execute with specific network and dry run
  treb orchestrate deploy.yaml --network sepolia --dry-run

  # Execute with debug output
  treb orchestrate deploy.yaml --debug --verbose`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		orchestrationFile := args[0]

		// Get flags
		network, _ := cmd.Flags().GetString("network")
		namespace, _ := cmd.Flags().GetString("namespace") 
		profile, _ := cmd.Flags().GetString("profile")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		debug, _ := cmd.Flags().GetBool("debug")
		debugJSON, _ := cmd.Flags().GetBool("debug-json")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Default network and namespace resolution (same as run command)
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

		// Parse orchestration file
		parser := orchestration.NewParser()
		config, err := parser.ParseFile(orchestrationFile)
		if err != nil {
			display.PrintErrorMessage(fmt.Sprintf("Failed to parse orchestration file: %v", err))
			os.Exit(1)
		}

		// Create execution plan
		plan, err := parser.CreateExecutionPlan(config)
		if err != nil {
			display.PrintErrorMessage(fmt.Sprintf("Failed to create execution plan: %v", err))
			os.Exit(1)
		}

		// Create executor configuration
		// Orchestration should always be non-interactive to avoid hanging on ambiguous script names
		executorConfig := &orchestration.ExecutorConfig{
			Network:        network,
			Namespace:      namespace,
			Profile:        profile,
			DryRun:         dryRun,
			Debug:          debug,
			DebugJSON:      debugJSON,
			Verbose:        verbose,
			NonInteractive: true,
		}

		// Create and run executor
		executor, err := orchestration.NewExecutor(executorConfig)
		if err != nil {
			display.PrintErrorMessage(fmt.Sprintf("Failed to create orchestration executor: %v", err))
			os.Exit(1)
		}

		if err := executor.Execute(plan); err != nil {
			display.PrintErrorMessage(fmt.Sprintf("Orchestration failed: %v", err))
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(orchestrateCmd)

	// Network and namespace flags
	orchestrateCmd.Flags().StringP("network", "n", "", "Network to deploy to (local, sepolia, mainnet, etc.)")
	orchestrateCmd.Flags().String("namespace", "", "Deployment namespace (default, staging, production) [also sets foundry profile]")

	// Execution flags
	orchestrateCmd.Flags().Bool("dry-run", false, "Perform a dry run without broadcasting transactions")
	orchestrateCmd.Flags().Bool("debug", false, "Enable debug mode (shows forge output and saves to file)")
	orchestrateCmd.Flags().Bool("debug-json", false, "Enable JSON debug mode (shows raw JSON output)")
	orchestrateCmd.Flags().BoolP("verbose", "v", false, "Show extra detailed information for events and transactions")
}

package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewComposeCmd creates the orchestrate command using the new architecture
func NewComposeCmd() *cobra.Command {
	var (
		network        string
		namespace      string
		profile        string
		dryRun         bool
		debug          bool
		debugJSON      bool
		verbose        bool
		nonInteractive bool
	)

	cmd := &cobra.Command{
		Use:   "compose <compose-file>",
		Short: "Execute orchestrated deployments from a YAML configuration",
		Long: `Execute multiple deployment scripts in dependency order based on a YAML configuration file.

The composes file defines components, their deployment scripts, dependencies, and environment variables.
Treb will build a dependency graph and execute scripts in the correct order.

Example compose file (protocol.yaml):
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
  treb compose deploy.yaml

  # Execute with specific network and dry run
  treb compose deploy.yaml --network sepolia --dry-run

  # Execute with debug output
  treb compose deploy.yaml --debug --verbose`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			orchestrationFile := args[0]

			// Default network and namespace resolution
			if network == "" {
				network = os.Getenv("DEPLOYMENT_NETWORK")
				if network == "" {
					network = "local"
				}
			}

			// Default namespace
			if namespace == "" {
				namespace = "default"
			}

			// Create compose parameters
			params := usecase.ComposeParams{
				ConfigPath:     orchestrationFile,
				Network:        network,
				Namespace:      namespace,
				Profile:        profile,
				DryRun:         dryRun,
				Debug:          debug,
				DebugJSON:      debugJSON,
				Verbose:        verbose,
				NonInteractive: true, // Orchestration should always be non-interactive
			}

			ctx := cmd.Context()

			// Execute orchestration
			result, err := app.ComposeDeployment.Execute(ctx, params)
			if err != nil {
				return err
			}

			// Render the results
			renderer := render.NewComposeRenderer(cmd.OutOrStdout())
			return renderer.RenderComposeResult(result)
		},
	}

	// Network and namespace flags
	cmd.Flags().StringVarP(&network, "network", "n", "", "Network to deploy to (local, sepolia, mainnet, etc.)")
	cmd.Flags().StringVarP(&namespace, "namespace", "s", "", "Deployment namespace (default, staging, production) [also sets foundry profile]")
	cmd.Flags().StringVar(&profile, "profile", "", "Foundry profile to use (overrides namespace-based profile)")

	// Execution flags
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without broadcasting transactions")
	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode (shows forge output and saves to file)")
	cmd.Flags().BoolVar(&debugJSON, "debug-json", false, "Enable JSON debug mode (shows raw JSON output)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show extra detailed information for events and transactions")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Disable interactive prompts (always non-interactive for orchestration)")

	return cmd
}

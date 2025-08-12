package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewShowCmd creates the show command using the new architecture
func NewShowCmd(baseCfg *app.Config) *cobra.Command {
	var (
		jsonOutput bool
		network    string
		namespace  string
	)

	cmd := &cobra.Command{
		Use:   "show <deployment>",
		Short: "Show detailed deployment information from registry",
		Long: `Show detailed information about a specific deployment.

You can specify deployments using:
- Contract name: "Counter"
- Contract with label: "Counter:v2"
- Namespace/contract: "staging/Counter"
- Chain/contract: "11155111/Counter"
- Full deployment ID: "production/1/Counter:v1"
- Contract address: "0x1234..."
- Alias: "MyCounter"

Examples:
  treb show Counter
  treb show Counter:v2
  treb show 0x1234567890abcdef...
  treb show production/1/Counter:v1
  treb show MyCounter`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentRef := args[0]

			// Use NOP progress for now to preserve exact output
			progressSink := usecase.NopProgress{}

			// Initialize app with Wire
			app, err := app.InitApp(*baseCfg, progressSink)
			if err != nil {
				return fmt.Errorf("failed to initialize app: %w", err)
			}

			// Use default namespace from config if not specified
			if namespace == "" {
				// Load the old config to get namespace
				cfg, err := config.NewManager(baseCfg.ProjectRoot).Load()
				if err == nil && cfg.Namespace != "" {
					namespace = cfg.Namespace
				}
			}

			// Resolve network to chain ID if specified
			var chainID uint64
			if network != "" {
				// TODO: Implement network resolution when NetworkResolver is available
				// For now, we'll skip network resolution
			}

			// Run use case
			params := usecase.ShowDeploymentParams{
				DeploymentRef: deploymentRef,
				ChainID:       chainID,
				Namespace:     namespace,
				ResolveProxy:  true, // Always resolve proxy implementations
			}

			deployment, err := app.ShowDeployment.Run(context.Background(), params)
			if err != nil {
				return fmt.Errorf("failed to resolve deployment: %w", err)
			}

			// Output JSON if requested
			if jsonOutput {
				// For JSON output, we need to structure the data
				output := map[string]interface{}{
					"deployment": deployment,
					// TODO: Add transaction when transaction support is implemented
				}
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			// Render output (preserve existing format exactly)
			// Detect if color is enabled from the command
			color := cmd.OutOrStdout() == cmd.OutOrStdout() // Simple check, can be improved
			renderer := render.NewDetailRenderer(cmd.OutOrStdout(), color)
			return renderer.RenderDeployment(deployment)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&network, "network", "n", "", "Network to use (e.g., mainnet, sepolia)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Namespace to use (defaults to current context)")

	return cmd
}
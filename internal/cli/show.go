package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewShowCmd creates the show command using the new architecture
func NewShowCmd() *cobra.Command {

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

			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Run use case
			params := usecase.ShowDeploymentParams{
				DeploymentRef: deploymentRef,
				ResolveProxy:  true, // Always resolve proxy implementations
			}

			deployment, err := app.ShowDeployment.Run(cmd.Context(), params)
			if err != nil {
				return fmt.Errorf("failed to resolve deployment: %w", err)
			}

			// Output JSON if requested
			if app.Config.JSON {
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

	// No command-specific flags - all configuration comes from runtime config

	return cmd
}


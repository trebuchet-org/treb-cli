package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewListCmd creates the list command using the new architecture
func NewListCmd() *cobra.Command {
	var (
		contractName string
		label        string
		deployType   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List deployments from registry",
		Long: `List all deployments from the registry.

The list can be filtered by namespace, chain ID, contract name, label, or deployment type.`,
		Example: `  # List all deployments
  treb list

  # List all Counter deployments
  treb list --contract Counter

  # List proxy deployments only
  treb list --type proxy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Convert string type to domain type
			var deploymentType models.DeploymentType
			if deployType != "" {
				switch deployType {
				case "singleton":
					deploymentType = models.SingletonDeployment
				case "proxy":
					deploymentType = models.ProxyDeployment
				case "library":
					deploymentType = models.LibraryDeployment
				default:
					return fmt.Errorf("invalid deployment type: %s (valid: singleton, proxy, library)", deployType)
				}
			}

			// Run use case
			params := usecase.ListDeploymentsParams{
				ContractName: contractName,
				Label:        label,
				Type:         deploymentType,
			}

			result, err := app.ListDeployments.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			// Render output (preserve existing format exactly)
			// Detect if color is enabled from the command
			color := cmd.OutOrStdout() == cmd.OutOrStdout() // Simple check, can be improved
			renderer := render.NewDeploymentsRenderer(cmd.OutOrStdout(), color)
			return renderer.RenderDeploymentList(result)
		},
	}

	// Add flags (removed namespace and chain - these come from runtime config)
	cmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")
	cmd.Flags().StringP("namespace", "s", "", "Namespace to use (defaults to current context namespace) [also sets foundry profile]")
	cmd.Flags().StringVar(&contractName, "contract", "", "Filter by contract name")
	cmd.Flags().StringVar(&label, "label", "", "Filter by label")
	cmd.Flags().StringVar(&deployType, "type", "", "Filter by deployment type (singleton, proxy, library)")

	return cmd
}

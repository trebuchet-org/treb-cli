package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewListCmd creates the list command using the new architecture
func NewListCmd(baseCfg *app.Config) *cobra.Command {
	var (
		namespace    string
		chainID      uint64
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

  # List deployments in production namespace
  treb list --namespace production

  # List deployments on mainnet
  treb list --chain 1

  # List all Counter deployments
  treb list --contract Counter

  # List proxy deployments only
  treb list --type proxy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use NOP progress for now to preserve exact output
			progressSink := usecase.NopProgress{}

			// Initialize app with Wire
			app, err := app.InitApp(*baseCfg, progressSink)
			if err != nil {
				return fmt.Errorf("failed to initialize app: %w", err)
			}

			// Use default namespace from config if not specified AND no other filters are set
			if namespace == "" && chainID == 0 && contractName == "" && label == "" && deployType == "" {
				// Load the old config to get namespace
				cfg, err := config.NewManager(baseCfg.ProjectRoot).Load()
				if err == nil && cfg.Namespace != "" {
					namespace = cfg.Namespace
				}
			}

			// Convert string type to domain type
			var deploymentType domain.DeploymentType
			if deployType != "" {
				switch deployType {
				case "singleton":
					deploymentType = domain.SingletonDeployment
				case "proxy":
					deploymentType = domain.ProxyDeployment
				case "library":
					deploymentType = domain.LibraryDeployment
				default:
					return fmt.Errorf("invalid deployment type: %s (valid: singleton, proxy, library)", deployType)
				}
			}

			// Run use case
			params := usecase.ListDeploymentsParams{
				Namespace:    namespace,
				ChainID:      chainID,
				ContractName: contractName,
				Label:        label,
				Type:         deploymentType,
			}

			result, err := app.ListDeployments.Run(context.Background(), params)
			if err != nil {
				return err
			}

			// Render output (preserve existing format exactly)
			// Detect if color is enabled from the command
			color := cmd.OutOrStdout() == cmd.OutOrStdout() // Simple check, can be improved
			renderer := render.NewTableRenderer(cmd.OutOrStdout(), color)
			return renderer.RenderDeploymentList(result)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&namespace, "namespace", "", "Filter by namespace")
	cmd.Flags().Uint64Var(&chainID, "chain", 0, "Filter by chain ID")
	cmd.Flags().StringVar(&contractName, "contract", "", "Filter by contract name")
	cmd.Flags().StringVar(&label, "label", "", "Filter by label")
	cmd.Flags().StringVar(&deployType, "type", "", "Filter by deployment type (singleton, proxy, library)")

	return cmd
}
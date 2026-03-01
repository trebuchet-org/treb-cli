package cli

import (
	"encoding/json"
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
		forkOnly     bool
		noFork       bool
		jsonOutput   bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List deployments from registry",
		Long: `List all deployments from the registry.

The list can be filtered by namespace, chain ID, contract name, label, or deployment type.

In fork mode, deployments added during the fork are marked with [fork].
Use --fork to show only fork-added deployments, or --no-fork to exclude them.`,
		Example: `  # List all deployments
  treb list

  # List all Counter deployments
  treb list --contract Counter

  # List proxy deployments only
  treb list --type proxy

  # List only fork-added deployments
  treb list --fork

  # List only pre-fork deployments
  treb list --no-fork`,
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
				ForkOnly:     forkOnly,
				NoFork:       noFork,
			}

			result, err := app.ListDeployments.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			// JSON output
			if jsonOutput {
				return renderListJSON(result)
			}

			// Render output (preserve existing format exactly)
			// Detect if color is enabled from the command
			color := cmd.OutOrStdout() == cmd.OutOrStdout() // Simple check, can be improved
			renderer := render.NewDeploymentsRenderer(cmd.OutOrStdout(), color)
			return renderer.RenderDeploymentList(result)
		},
	}

	// Namespace and network flags are bound to viper automatically via SetupViper
	cmd.Flags().StringP("namespace", "s", "", "Namespace to use (defaults to current context namespace) [also sets foundry profile]")
	cmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")
	cmd.Flags().StringVar(&contractName, "contract", "", "Filter by contract name")
	cmd.Flags().StringVar(&label, "label", "", "Filter by label")
	cmd.Flags().StringVar(&deployType, "type", "", "Filter by deployment type (singleton, proxy, library)")
	cmd.Flags().BoolVar(&forkOnly, "fork", false, "Show only fork-added deployments")
	cmd.Flags().BoolVar(&noFork, "no-fork", false, "Show only pre-fork deployments")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// listJSONEntry represents a deployment in JSON output
type listJSONEntry struct {
	ID           string `json:"id"`
	ContractName string `json:"contractName"`
	Address      string `json:"address"`
	Namespace    string `json:"namespace"`
	ChainID      uint64 `json:"chainId"`
	Label        string `json:"label,omitempty"`
	Type         string `json:"type"`
	Fork         bool   `json:"fork,omitempty"`
}

// listJSONOutput wraps the JSON output with optional namespace discovery data
type listJSONOutput struct {
	Deployments     []listJSONEntry `json:"deployments"`
	OtherNamespaces map[string]int  `json:"otherNamespaces,omitempty"`
}

// renderListJSON outputs deployments as JSON
func renderListJSON(result *usecase.DeploymentListResult) error {
	entries := make([]listJSONEntry, 0, len(result.Deployments))
	for _, dep := range result.Deployments {
		entry := listJSONEntry{
			ID:           dep.ID,
			ContractName: dep.ContractName,
			Address:      dep.Address,
			Namespace:    dep.Namespace,
			ChainID:      dep.ChainID,
			Label:        dep.Label,
			Type:         string(dep.Type),
		}
		if result.ForkDeploymentIDs != nil && result.ForkDeploymentIDs[dep.ID] {
			entry.Fork = true
		}
		entries = append(entries, entry)
	}

	output := listJSONOutput{
		Deployments: entries,
	}

	// Include other namespaces only when current namespace is empty and others exist
	if len(result.Deployments) == 0 && len(result.OtherNamespaces) > 0 {
		output.OtherNamespaces = result.OtherNamespaces
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

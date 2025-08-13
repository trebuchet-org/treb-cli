package cli

import (
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewNetworksCmd creates the networks command using the new architecture
func NewNetworksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "networks",
		Short: "List available networks from foundry.toml",
		Long: `List all networks configured in the [rpc_endpoints] section of foundry.toml.

This command shows all available networks and attempts to fetch their chain IDs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Run use case
			params := usecase.ListNetworksParams{}
			result, err := app.ListNetworks.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			// Render output
			color := cmd.OutOrStdout() == cmd.OutOrStdout() // Simple check, can be improved
			renderer := render.NewNetworksRenderer(cmd.OutOrStdout(), color)
			return renderer.RenderNetworksList(result)
		},
	}

	return cmd
}
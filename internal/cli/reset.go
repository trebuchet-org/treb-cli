package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewResetCmd creates the reset command
func NewResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset registry entries for the current namespace and network",
		Long: `Reset registry entries for the current namespace and network.

This command deletes all deployments, transactions, and safe transactions
matching the current namespace and network from the registry. This is useful
for cleaning up and starting fresh on a given namespace/network combination.

The namespace and network are determined from the current configuration context
(set via 'treb config set' or --namespace/--network flags).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			if app.Config.Network == nil {
				return fmt.Errorf("network must be set (use 'treb config set network <name>' or --network flag)")
			}

			renderer := render.NewResetRenderer(cmd.OutOrStdout())

			// First, collect items to reset (dry run)
			result, err := app.ResetRegistry.Run(cmd.Context(), usecase.ResetRegistryParams{
				DryRun: true,
			})
			if err != nil {
				return err
			}

			// If no items to reset, we're done
			if !result.Changeset.HasChanges() {
				renderer.RenderNothing()
				return nil
			}

			// Show what will be deleted
			del := result.Changeset.Delete
			renderer.RenderItemsToReset(
				app.Config.Namespace,
				app.Config.Network.Name,
				app.Config.Network.ChainID,
				len(del.Deployments),
				len(del.Transactions),
				len(del.SafeTransactions),
			)

			// Handle confirmation
			if !app.Config.NonInteractive {
				fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to reset the registry for namespace '%s' on network '%s'? This cannot be undone. [y/N]: ",
					app.Config.Namespace,
					app.Config.Network.Name,
				)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					renderer.RenderCancelled()
					return nil
				}

				if strings.ToLower(strings.TrimSpace(response)) != "y" {
					renderer.RenderCancelled()
					return nil
				}
			} else {
				renderer.RenderNonInteractive()
			}

			// Execute the actual reset
			result, err = app.ResetRegistry.Run(cmd.Context(), usecase.ResetRegistryParams{
				DryRun: false,
			})
			if err != nil {
				return err
			}

			renderer.RenderSuccess(result.Changeset.Delete.Count())

			return nil
		},
	}

	return cmd
}

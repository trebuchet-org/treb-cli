package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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

			// First, collect items to reset (dry run)
			result, err := app.ResetRegistry.Run(cmd.Context(), usecase.ResetRegistryParams{
				DryRun: true,
			})
			if err != nil {
				return err
			}

			// If no items to reset, we're done
			if !result.Changeset.HasChanges() {
				fmt.Fprintln(cmd.OutOrStdout(), "Nothing to reset. No registry entries found for the current namespace and network.")
				return nil
			}

			// Show what will be deleted
			del := result.Changeset.Delete
			fmt.Fprintf(cmd.OutOrStdout(), "Found %d items to reset for namespace '%s' on network '%s' (chain %d):\n\n",
				del.Count(),
				app.Config.Namespace,
				app.Config.Network.Name,
				app.Config.Network.ChainID,
			)

			if len(del.Deployments) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  Deployments:        %d\n", len(del.Deployments))
			}
			if len(del.Transactions) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  Transactions:       %d\n", len(del.Transactions))
			}
			if len(del.SafeTransactions) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  Safe Transactions:  %d\n", len(del.SafeTransactions))
			}
			fmt.Fprintln(cmd.OutOrStdout())

			// Handle confirmation
			if !app.Config.NonInteractive {
				fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to reset the registry for namespace '%s' on network '%s'? This cannot be undone. [y/N]: ",
					app.Config.Namespace,
					app.Config.Network.Name,
				)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "Reset cancelled.")
					return nil
				}

				if strings.ToLower(strings.TrimSpace(response)) != "y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Reset cancelled.")
					return nil
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Running in non-interactive mode. Proceeding with reset...")
			}

			// Execute the actual reset
			result, err = app.ResetRegistry.Run(cmd.Context(), usecase.ResetRegistryParams{
				DryRun: false,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Successfully reset %d items from the registry.\n", result.Changeset.Delete.Count())

			return nil
		},
	}

	return cmd
}

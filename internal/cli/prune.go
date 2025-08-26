package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewPruneCmd creates the prune command using the new architecture
func NewPruneCmd() *cobra.Command {
	var (
		includePending bool
		network        string
	)

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune registry entries that no longer exist on-chain",
		Long: `Prune registry entries that no longer exist on-chain.

This command checks all deployments, transactions, and safe transactions
against the blockchain and removes entries that no longer exist. This is
useful for cleaning up after test deployments on local or virtual networks.

By default, pending items (queued safe transactions, simulated transactions)
are preserved. Use --include-pending to also prune these items.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Network is required
			if network == "" {
				return fmt.Errorf("--network flag is required")
			}

			// First, collect items to prune (dry run)
			collectParams := usecase.PruneRegistryParams{
				NetworkName:    network,
				IncludePending: includePending,
				DryRun:         true,
			}

			result, err := app.PruneRegistry.Run(cmd.Context(), collectParams)
			if err != nil {
				return err
			}

			// Render items to be pruned
			renderer := render.NewPruneRenderer(cmd.OutOrStdout())
			if err := renderer.RenderItemsToPrune(result.Changeset.Delete); err != nil {
				return err
			}

			// If no items to prune, we're done
			if result.Changeset.Count() == 0 {
				return nil
			}

			// Handle confirmation
			if !app.Config.NonInteractive {
				fmt.Fprint(cmd.OutOrStdout(), "⚠️  Are you sure you want to prune these items? This cannot be undone. [y/N]: ")
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					// Treat error as "no" response
					fmt.Fprintln(cmd.OutOrStdout(), "❌ Prune cancelled.")
					return nil
				}

				if strings.ToLower(strings.TrimSpace(response)) != "y" {
					fmt.Fprintln(cmd.OutOrStdout(), "❌ Prune cancelled.")
					return nil
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "⚠️  Running in non-interactive mode. Proceeding with prune...")
			}

			// Now execute the actual prune
			executeParams := usecase.PruneRegistryParams{
				NetworkName:    network,
				IncludePending: includePending,
				DryRun:         false,
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\n🔧 Pruning registry entries...")

			_, err = app.PruneRegistry.Run(cmd.Context(), executeParams)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✅ Successfully pruned %d items from the registry.\n", result.Changeset.Count())

			return nil
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&includePending, "include-pending", false, "Also prune pending items (queued safe txs, simulated txs)")
	cmd.Flags().StringVar(&network, "network", "", "Network to verify against (required)")

	// Mark network as required
	if err := cmd.MarkFlagRequired("network"); err != nil {
		// This should not happen, but handle it gracefully
		panic(fmt.Sprintf("failed to mark network flag as required: %v", err))
	}

	return cmd
}

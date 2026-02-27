package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewSyncCmd creates the sync command using the new architecture
func NewSyncCmd() *cobra.Command {
	var (
		clean bool
		debug bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync registry with on-chain state",
		Long: `Update deployment registry with latest on-chain information.
Checks pending Safe transactions and updates their execution status.

This command will:
- Check all pending Safe transactions for execution status
- Update transaction records when Safe txs are executed
- Update deployment status based on transaction status
- Clean up orphaned records if --clean is specified`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Block sync in fork mode
			if active, _ := isForkActiveForCurrentNetwork(cmd.Context(), app); active {
				return fmt.Errorf("cannot sync with a fork")
			}

			// Create sync options
			options := usecase.SyncOptions{
				Clean: clean,
				Debug: debug,
			}

			ctx := cmd.Context()

			// Execute sync
			result, err := app.SyncRegistry.Sync(ctx, options)
			if err != nil {
				return err
			}

			// Render the results
			renderer := render.NewSyncRenderer(cmd.OutOrStdout())
			return renderer.RenderSyncResult(result)
		},
	}

	cmd.Flags().BoolVar(&clean, "clean", false, "Remove invalid entries while syncing")
	cmd.Flags().BoolVar(&debug, "debug", false, "Show debug information during sync")

	return cmd
}

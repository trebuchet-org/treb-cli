package render

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// SyncRenderer handles rendering of sync results
type SyncRenderer struct {
	out io.Writer
}

// NewSyncRenderer creates a new sync renderer
func NewSyncRenderer(out io.Writer) *SyncRenderer {
	return &SyncRenderer{
		out: out,
	}
}

// RenderSyncResult renders the result of sync operation
func (r *SyncRenderer) RenderSyncResult(result *usecase.SyncResult) error {
	fmt.Fprintln(r.out, "Syncing registry...")

	// Show Safe transaction sync results
	if result.PendingSafeTxsChecked > 0 {
		fmt.Fprintf(r.out, "\nSafe Transactions:\n")
		fmt.Fprintf(r.out, "  • Checked: %d\n", result.PendingSafeTxsChecked)

		if result.SafeTxsExecuted > 0 {
			color.New(color.FgGreen).Fprintf(r.out, "  • Executed: %d\n", result.SafeTxsExecuted)
		}

		if result.TransactionsUpdated > 0 {
			fmt.Fprintf(r.out, "  • Transactions updated: %d\n", result.TransactionsUpdated)
		}

		if result.DeploymentsUpdated > 0 {
			fmt.Fprintf(r.out, "  • Deployments updated: %d\n", result.DeploymentsUpdated)
		}
	} else {
		fmt.Fprintln(r.out, "No pending Safe transactions found")
	}

	// Show cleanup results if any
	if result.InvalidEntriesRemoved > 0 {
		fmt.Fprintf(r.out, "\nCleanup:\n")
		fmt.Fprintf(r.out, "  • Invalid entries removed: %d\n", result.InvalidEntriesRemoved)
	}

	// Show errors if any
	if len(result.Errors) > 0 {
		color.New(color.FgYellow).Fprintf(r.out, "\nWarnings:\n")
		for _, err := range result.Errors {
			fmt.Fprintf(r.out, "  • %s\n", err)
		}
	}

	// Show final status
	fmt.Fprintln(r.out)
	if len(result.Errors) == 0 {
		color.New(color.FgGreen).Fprintln(r.out, "✓ Registry synced successfully")
	} else {
		color.New(color.FgGreen).Fprintln(r.out, "✓ Registry sync completed with warnings")
	}

	return nil
}

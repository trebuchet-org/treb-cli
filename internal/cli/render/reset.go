package render

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

// ResetRenderer renders reset-related output
type ResetRenderer struct {
	out io.Writer
}

// NewResetRenderer creates a new reset renderer
func NewResetRenderer(out io.Writer) *ResetRenderer {
	return &ResetRenderer{
		out: out,
	}
}

// RenderNothing renders the message when there are no items to reset
func (r *ResetRenderer) RenderNothing() {
	fmt.Fprintln(r.out, "Nothing to reset. No registry entries found for the current namespace and network.")
}

// RenderItemsToReset renders the summary of items that will be reset
func (r *ResetRenderer) RenderItemsToReset(namespace, network string, chainID uint64, deployments, transactions, safeTransactions int) {
	total := deployments + transactions + safeTransactions
	fmt.Fprintf(r.out, "Found %s items to reset for namespace '%s' on network '%s' (chain %d):\n\n",
		bold.Sprintf("%d", total),
		cyan.Sprint(namespace),
		cyan.Sprint(network),
		chainID,
	)

	if deployments > 0 {
		fmt.Fprintf(r.out, "  Deployments:        %s\n", bold.Sprintf("%d", deployments))
	}
	if transactions > 0 {
		fmt.Fprintf(r.out, "  Transactions:       %s\n", bold.Sprintf("%d", transactions))
	}
	if safeTransactions > 0 {
		fmt.Fprintf(r.out, "  Safe Transactions:  %s\n", bold.Sprintf("%d", safeTransactions))
	}
	fmt.Fprintln(r.out)
}

// RenderNonInteractive renders the non-interactive mode notice
func (r *ResetRenderer) RenderNonInteractive() {
	fmt.Fprintln(r.out, "Running in non-interactive mode. Proceeding with reset...")
}

// RenderCancelled renders the cancellation message
func (r *ResetRenderer) RenderCancelled() {
	fmt.Fprintln(r.out, "Reset cancelled.")
}

// RenderSuccess renders the success message after reset
func (r *ResetRenderer) RenderSuccess(count int) {
	fmt.Fprintln(r.out, color.New(color.FgGreen).Sprintf("✓ Successfully reset %d items from the registry.", count))
}

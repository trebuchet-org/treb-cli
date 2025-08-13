package render

import (
	"fmt"
	"io"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// PruneRenderer renders prune-related output
type PruneRenderer struct {
	out io.Writer
}

// NewPruneRenderer creates a new prune renderer
func NewPruneRenderer(out io.Writer) *PruneRenderer {
	return &PruneRenderer{
		out: out,
	}
}

// RenderItemsToPrune renders the items that will be pruned
func (r *PruneRenderer) RenderItemsToPrune(items *domain.ItemsToPrune, totalItems int) error {
	// Check if there's anything to prune
	if totalItems == 0 {
		fmt.Fprintln(r.out, "✅ All registry entries are valid. Nothing to prune.")
		return nil
	}

	// Display header
	fmt.Fprintln(r.out, "🔍 Checking registry entries against on-chain state...")
	fmt.Fprintf(r.out, "\n🗑️  Found %d items to prune:\n\n", totalItems)

	// Display deployments to prune
	if len(items.Deployments) > 0 {
		fmt.Fprintf(r.out, "Deployments (%d):\n", len(items.Deployments))
		for _, dep := range items.Deployments {
			fmt.Fprintf(r.out, "  - %s at %s (reason: %s)\n", dep.ID, dep.Address, dep.Reason)
		}
		fmt.Fprintln(r.out)
	}

	// Display transactions to prune
	if len(items.Transactions) > 0 {
		fmt.Fprintf(r.out, "Transactions (%d):\n", len(items.Transactions))
		for _, tx := range items.Transactions {
			status := string(tx.Status)
			if tx.Hash != "" {
				fmt.Fprintf(r.out, "  - %s [%s] (reason: %s)\n", tx.ID, status, tx.Reason)
			} else {
				fmt.Fprintf(r.out, "  - %s [%s] (reason: %s)\n", tx.ID, status, tx.Reason)
			}
		}
		fmt.Fprintln(r.out)
	}

	// Display safe transactions to prune
	if len(items.SafeTransactions) > 0 {
		fmt.Fprintf(r.out, "Safe Transactions (%d):\n", len(items.SafeTransactions))
		for _, safeTx := range items.SafeTransactions {
			// Show short safe address
			shortSafeAddr := safeTx.SafeAddress
			if len(shortSafeAddr) > 10 {
				shortSafeAddr = safeTx.SafeAddress[0:10] + "..."
			}
			fmt.Fprintf(r.out, "  - %s on Safe %s [%s] (reason: %s)\n",
				safeTx.SafeTxHash,
				shortSafeAddr,
				safeTx.Status,
				safeTx.Reason)
		}
		fmt.Fprintln(r.out)
	}

	return nil
}
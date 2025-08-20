package render

import (
	"fmt"
	"io"

	"github.com/trebuchet-org/treb-cli/internal/domain/models"
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
func (r *PruneRenderer) RenderItemsToPrune(changeset models.ChangesetModels) error {
	// Check if there's anything to prune
	if changeset.Count() == 0 {
		fmt.Fprintln(r.out, "✅ All registry entries are valid. Nothing to prune.")
		return nil
	}

	// Display header
	fmt.Fprintln(r.out, "🔍 Checking registry entries against on-chain state...")
	fmt.Fprintf(r.out, "\n🗑️  Found %d items to prune:\n\n", changeset.Count())

	// Display deployments to prune
	if len(changeset.Deployments) > 0 {
		fmt.Fprintf(r.out, "Deployments (%d):\n", len(changeset.Deployments))
		for _, dep := range changeset.Deployments {
			fmt.Fprintf(r.out, "  - %s at %s (reason: %s)\n", dep.ID, dep.Address, changeset.Metadata.Reasons[dep.ID])
		}
		fmt.Fprintln(r.out)
	}

	// Display transactions to prune
	if len(changeset.Transactions) > 0 {
		fmt.Fprintf(r.out, "Transactions (%d):\n", len(changeset.Transactions))
		for _, tx := range changeset.Transactions {
			status := string(tx.Status)
			reason := changeset.Metadata.Reasons[tx.ID]
			if tx.Hash != "" {
				fmt.Fprintf(r.out, "  - %s [%s] (reason: %s)\n", tx.ID, status, reason)
			} else {
				fmt.Fprintf(r.out, "  - %s [%s] (reason: %s)\n", tx.ID, status, reason)
			}
		}
		fmt.Fprintln(r.out)
	}

	// Display safe transactions to prune
	if len(changeset.SafeTransactions) > 0 {
		fmt.Fprintf(r.out, "Safe Transactions (%d):\n", len(changeset.SafeTransactions))
		for _, safeTx := range changeset.SafeTransactions {
			// Show short safe address
			shortSafeAddr := safeTx.SafeAddress
			if len(shortSafeAddr) > 10 {
				shortSafeAddr = safeTx.SafeAddress[0:10] + "..."
			}
			reason := changeset.Metadata.Reasons[safeTx.SafeTxHash]
			fmt.Fprintf(r.out, "  - %s on Safe %s [%s] (reason: %s)\n",
				safeTx.SafeTxHash,
				shortSafeAddr,
				safeTx.Status,
				reason)
		}
		fmt.Fprintln(r.out)
	}

	return nil
}

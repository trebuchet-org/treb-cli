package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

var (
	includePending bool
	pruneNetwork   string
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune registry entries that no longer exist on-chain",
	Long: `Prune registry entries that no longer exist on-chain.

This command checks all deployments, transactions, and safe transactions
against the blockchain and removes entries that no longer exist. This is
useful for cleaning up after test deployments on local or virtual networks.

By default, pending items (queued safe transactions, simulated transactions)
are preserved. Use --include-pending to also prune these items.`,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().BoolVar(&includePending, "include-pending", false, "Also prune pending items (queued safe txs, simulated txs)")
	pruneCmd.Flags().StringVar(&pruneNetwork, "network", "", "Network to verify against (required)")
	if err := pruneCmd.MarkFlagRequired("network"); err != nil {
		panic(fmt.Sprintf("failed to mark flag as required: %v", err))
	}

	// Set command group
	pruneCmd.GroupID = "management"

	// Register command
	rootCmd.AddCommand(pruneCmd)
}

func runPrune(cmd *cobra.Command, args []string) error {
	// Use centralized network resolver
	networkResolver, err := network.NewResolver(".")
	if err != nil {
		return fmt.Errorf("failed to create network resolver: %w", err)
	}
	networkInfo, err := networkResolver.ResolveNetwork(pruneNetwork)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Create registry manager
	manager, err := registry.NewManager(".")
	if err != nil {
		return fmt.Errorf("failed to create registry manager: %w", err)
	}

	// Create pruner with network config
	pruner := manager.NewPruner(networkInfo.RpcUrl, networkInfo.ChainID)

	// Connect to RPC
	if err := pruner.Connect(networkInfo.RpcUrl); err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}

	// Collect items to prune
	fmt.Println("🔍 Checking registry entries against on-chain state...")
	itemsToPrune, err := pruner.CollectItemsToPrune(includePending)
	if err != nil {
		return fmt.Errorf("failed to collect items to prune: %w", err)
	}

	// Check if there's anything to prune
	totalItems := len(itemsToPrune.Deployments) + len(itemsToPrune.Transactions) + len(itemsToPrune.SafeTransactions)
	if totalItems == 0 {
		fmt.Println("✅ All registry entries are valid. Nothing to prune.")
		return nil
	}

	// Display items to be pruned
	fmt.Printf("\n🗑️  Found %d items to prune:\n\n", totalItems)

	if len(itemsToPrune.Deployments) > 0 {
		fmt.Printf("Deployments (%d):\n", len(itemsToPrune.Deployments))
		for _, dep := range itemsToPrune.Deployments {
			fmt.Printf("  - %s at %s (reason: %s)\n", dep.ID, dep.Address, dep.Reason)
		}
		fmt.Println()
	}

	if len(itemsToPrune.Transactions) > 0 {
		fmt.Printf("Transactions (%d):\n", len(itemsToPrune.Transactions))
		for _, tx := range itemsToPrune.Transactions {
			status := string(tx.Status)
			if tx.Hash != "" {
				fmt.Printf("  - %s [%s] (reason: %s)\n", tx.ID, status, tx.Reason)
			} else {
				fmt.Printf("  - %s [%s] (reason: %s)\n", tx.ID, status, tx.Reason)
			}
		}
		fmt.Println()
	}

	if len(itemsToPrune.SafeTransactions) > 0 {
		fmt.Printf("Safe Transactions (%d):\n", len(itemsToPrune.SafeTransactions))
		for _, safeTx := range itemsToPrune.SafeTransactions {
			fmt.Printf("  - %s on Safe %s [%s] (reason: %s)\n",
				safeTx.SafeTxHash,
				safeTx.SafeAddress[0:10]+"...",
				safeTx.Status,
				safeTx.Reason)
		}
		fmt.Println()
	}

	// Confirmation prompt
	if !IsNonInteractive() {
		fmt.Print("⚠️  Are you sure you want to prune these items? This cannot be undone. [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Treat error as "no" response
			fmt.Println("❌ Prune cancelled.")
			return nil
		}

		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("❌ Prune cancelled.")
			return nil
		}
	} else {
		fmt.Println("⚠️  Running in non-interactive mode. Proceeding with prune...")
	}

	// Execute prune
	fmt.Println("\n🔧 Pruning registry entries...")
	if err := pruner.ExecutePrune(itemsToPrune); err != nil {
		return fmt.Errorf("failed to prune items: %w", err)
	}

	fmt.Printf("✅ Successfully pruned %d items from the registry.\n", totalItems)
	return nil
}

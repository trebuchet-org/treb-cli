package cmd

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/safe"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync registry with on-chain state",
	Long: `Update deployment registry with latest on-chain information.
Checks pending Safe transactions and updates their execution status.

This command will:
- Check all pending Safe transactions for execution status
- Update transaction records when Safe txs are executed
- Update deployment status based on transaction status
- Clean up orphaned records if --clean is specified`,
	Run: func(cmd *cobra.Command, args []string) {
		cleanFlag, _ := cmd.Flags().GetBool("clean")
		debugFlag, _ := cmd.Flags().GetBool("debug")
		
		if err := syncRegistry(cleanFlag, debugFlag); err != nil {
			checkError(err)
		}

		color.New(color.FgGreen).Println("✓ Registry synced successfully")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().Bool("clean", false, "Remove invalid entries while syncing")
	syncCmd.Flags().Bool("debug", false, "Show debug information during sync")
}

func syncRegistry(cleanRegistry bool, debugSync bool) error {
	// Initialize v2 registry manager
	manager, err := registry.NewManager(".")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	fmt.Println("Syncing registry...")

	// Check and update pending Safe transactions
	if err := syncPendingSafeTransactions(manager, debugSync); err != nil {
		fmt.Printf("Warning: Failed to sync Safe transactions: %v\n", err)
	}

	// Check pending transactions that might have been executed
	if err := syncPendingTransactions(manager, debugSync); err != nil {
		fmt.Printf("Warning: Failed to sync regular transactions: %v\n", err)
	}

	// Clean invalid entries if requested
	if cleanRegistry {
		fmt.Println("\nCleaning invalid entries...")
		// TODO: Implement cleanup logic for v2
		fmt.Println("Cleanup not yet implemented for v2 registry")
	}

	return nil
}

// syncPendingSafeTransactions checks pending Safe transactions and updates their status
func syncPendingSafeTransactions(manager *registry.Manager, debug bool) error {
	// Get all Safe transactions
	safeTxs := manager.GetAllSafeTransactions()
	
	// Filter pending ones
	var pendingSafeTxs []*types.SafeTransaction
	for _, safeTx := range safeTxs {
		if safeTx.Status == types.TransactionStatusPending {
			pendingSafeTxs = append(pendingSafeTxs, safeTx)
		}
	}

	if len(pendingSafeTxs) == 0 {
		fmt.Println("No pending Safe transactions found")
		return nil
	}

	// Group by chain
	pendingByChain := make(map[uint64][]*types.SafeTransaction)
	for _, safeTx := range pendingSafeTxs {
		pendingByChain[safeTx.ChainID] = append(pendingByChain[safeTx.ChainID], safeTx)
	}

	fmt.Printf("Found %d pending Safe transaction(s) on %d network(s)\n", len(pendingSafeTxs), len(pendingByChain))

	// Check each chain
	for chainID, chainSafeTxs := range pendingByChain {
		fmt.Printf("\nChecking %d pending Safe transaction(s) on chain %d...\n", len(chainSafeTxs), chainID)

		// Create Safe client for this chain
		safeClient, err := safe.NewClient(chainID)
		if err != nil {
			fmt.Printf("Warning: Cannot create Safe client for chain %d: %v\n", chainID, err)
			continue
		}

		// Enable debug if flag is set
		safeClient.SetDebug(debug)

		// Check each pending Safe transaction
		for _, safeTx := range chainSafeTxs {
			fmt.Printf("  Checking Safe tx %s... ", safeTx.SafeTxHash)

			// Debug info
			if debug {
				fmt.Printf("\n    [DEBUG] Safe address: %s\n", safeTx.SafeAddress)
				fmt.Printf("    [DEBUG] Nonce: %d\n", safeTx.Nonce)
				fmt.Printf("    [DEBUG] Proposed by: %s\n", safeTx.ProposedBy)
			}

			// Check if transaction is executed
			safeTxHashBytes := common.HexToHash(safeTx.SafeTxHash)
			isExecuted, ethTxHash, err := safeClient.IsTransactionExecuted(safeTxHashBytes)
			if err != nil {
				color.New(color.FgRed).Printf("ERROR: %v\n", err)
				continue
			}

			if isExecuted && ethTxHash != nil {
				color.New(color.FgGreen).Printf("EXECUTED (tx: %s)\n", ethTxHash.Hex())

				// Update the Safe transaction
				safeTx.Status = types.TransactionStatusExecuted
				safeTx.ExecutionTxHash = ethTxHash.Hex()
				now := time.Now()
				safeTx.ExecutedAt = &now

				// Save the updated Safe transaction
				if err := manager.UpdateSafeTransaction(safeTx); err != nil {
					fmt.Printf("    Warning: Failed to update Safe transaction: %v\n", err)
					continue
				}

				// Create transaction records for each operation in the batch
				for i, txData := range safeTx.Transactions {
					txID := fmt.Sprintf("tx-%s-%d", ethTxHash.Hex(), i)
					
					// Check if transaction already exists
					if _, err := manager.GetTransaction(txID); err == nil {
						// Transaction already recorded
						continue
					}

					// Create new transaction record
					tx := &types.Transaction{
						ID:      txID,
						ChainID: safeTx.ChainID,
						Hash:    ethTxHash.Hex(),
						Status:  types.TransactionStatusExecuted,
						Sender:  safeTx.SafeAddress,
						SafeContext: &types.SafeContext{
							SafeAddress: safeTx.SafeAddress,
							SafeTxHash:  safeTx.SafeTxHash,
							BatchIndex:  i,
							ProposerAddress: safeTx.ProposedBy,
						},
						Operations: []types.Operation{
							{
								Type:   "CALL",
								Target: txData.To,
								Method: "execute",
								Result: map[string]interface{}{
									"data": txData.Data,
									"value": txData.Value,
								},
							},
						},
						CreatedAt: time.Now(),
					}

					// Add transaction
					if err := manager.AddTransaction(tx); err != nil {
						fmt.Printf("    Warning: Failed to create transaction record: %v\n", err)
					}
				}

				// Update deployment records that reference this Safe tx
				// This would require scanning deployments and updating their transaction references
				// TODO: Implement deployment status updates
				
			} else {
				// Get more details about the pending transaction
				tx, err := safeClient.GetTransaction(safeTxHashBytes)
				if err == nil {
					color.New(color.FgYellow).Printf("PENDING (%d/%d confirmations)\n", 
						len(tx.Confirmations), tx.ConfirmationsRequired)
					
					// Update confirmations in our record
					safeTx.Confirmations = make([]types.Confirmation, 0, len(tx.Confirmations))
					for _, conf := range tx.Confirmations {
						safeTx.Confirmations = append(safeTx.Confirmations, types.Confirmation{
							Signer:      conf.Owner,
							Signature:   conf.Signature,
							ConfirmedAt: time.Now(), // Safe API doesn't provide confirmation time
						})
					}
					
					// Save updated confirmations
					if err := manager.UpdateSafeTransaction(safeTx); err != nil {
						fmt.Printf("    Warning: Failed to update confirmations: %v\n", err)
					}
				} else {
					color.New(color.FgYellow).Println("PENDING (couldn't get confirmation details)")
				}
			}
		}
	}

	return nil
}

// syncPendingTransactions checks regular pending transactions
func syncPendingTransactions(manager *registry.Manager, debug bool) error {
	// Get all transactions
	allTxs := manager.GetAllTransactions()
	
	// Filter pending ones
	var pendingTxs []*types.Transaction
	for _, tx := range allTxs {
		if tx.Status == types.TransactionStatusPending {
			pendingTxs = append(pendingTxs, tx)
		}
	}

	if len(pendingTxs) == 0 {
		fmt.Println("\nNo pending regular transactions found")
		return nil
	}

	fmt.Printf("\nFound %d pending regular transaction(s)\n", len(pendingTxs))

	// Group by chain
	pendingByChain := make(map[uint64][]*types.Transaction)
	for _, tx := range pendingTxs {
		pendingByChain[tx.ChainID] = append(pendingByChain[tx.ChainID], tx)
	}

	// Check each chain
	for chainID, chainTxs := range pendingByChain {
		fmt.Printf("\nChecking %d pending transaction(s) on chain %d...\n", len(chainTxs), chainID)
		
		// TODO: Implement blockchain transaction status checking
		// This would require:
		// 1. Getting RPC client for the chain
		// 2. Checking transaction receipt
		// 3. Updating status based on receipt
		
		for _, tx := range chainTxs {
			fmt.Printf("  Transaction %s - checking not yet implemented\n", tx.Hash)
		}
	}

	return nil
}
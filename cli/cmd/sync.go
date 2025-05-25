package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/safe"
)

var (
	cleanRegistry bool
	debugSync     bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync registry with on-chain state",
	Long: `Update deployment registry with latest on-chain information.
Checks pending Safe transactions and updates execution status.

Options:
  --clean  Remove invalid entries while syncing`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := syncRegistry(); err != nil {
			checkError(err)
		}

		fmt.Println("Registry synced successfully")
	},
}

func init() {
	syncCmd.Flags().BoolVar(&cleanRegistry, "clean", false, "Remove invalid entries while syncing")
	syncCmd.Flags().BoolVar(&debugSync, "debug", false, "Show debug information during sync")
}

func syncRegistry() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	fmt.Println("Syncing registry...")

	// Check and update pending Safe transactions
	if err := syncPendingSafeTransactions(registryManager); err != nil {
		fmt.Printf("Warning: Failed to sync Safe transactions: %v\n", err)
	}

	// Clean invalid entries if requested
	if cleanRegistry {
		fmt.Println("\nCleaning invalid entries...")
		cleaned := registryManager.CleanInvalidEntries()

		if cleaned > 0 {
			fmt.Printf("Removed %d invalid entries\n", cleaned)
		} else {
			fmt.Println("No invalid entries found")
		}
	}

	return registryManager.Save()
}

// syncPendingSafeTransactions checks pending Safe transactions and updates their status
func syncPendingSafeTransactions(registryManager *registry.Manager) error {
	deployments := registryManager.GetAllDeployments()

	// Group pending deployments by chain ID
	pendingByChain := make(map[uint64][]*registry.DeploymentInfo)

	for _, deployment := range deployments {
		if deployment.Entry.Deployment.Status == "pending_safe" && deployment.Entry.Deployment.SafeTxHash != nil {
			chainID, err := strconv.ParseUint(deployment.ChainID, 10, 64)
			if err != nil {
				fmt.Printf("Warning: Invalid chain ID %s for deployment %s\n", deployment.ChainID, deployment.Address.Hex())
				continue
			}
			pendingByChain[chainID] = append(pendingByChain[chainID], deployment)
		}
	}

	if len(pendingByChain) == 0 {
		fmt.Println("No pending Safe transactions found")
		return nil
	}

	fmt.Printf("Found pending Safe transactions on %d network(s)\n", len(pendingByChain))

	// Check each chain
	for chainID, pendingDeployments := range pendingByChain {
		fmt.Printf("\nChecking %d pending transaction(s) on chain %d...\n", len(pendingDeployments), chainID)

		// Create Safe client for this chain
		safeClient, err := safe.NewClient(chainID)
		if err != nil {
			fmt.Printf("Warning: Cannot create Safe client for chain %d: %v\n", chainID, err)
			continue
		}

		// Enable debug if flag is set
		safeClient.SetDebug(debugSync)

		// Check each pending deployment
		for _, deployment := range pendingDeployments {
			safeTxHash := *deployment.Entry.Deployment.SafeTxHash
			fmt.Printf("  Checking Safe tx %s for %s... \n", safeTxHash.Hex(), deployment.Entry.GetDisplayName())

			// Debug info
			if debugSync {
				fmt.Printf("    [DEBUG] Deployment address: %s\n", deployment.Address.Hex())
				fmt.Printf("    [DEBUG] Safe address: %s\n", deployment.Entry.Deployment.SafeAddress)
				fmt.Printf("    [DEBUG] Environment: %s\n", deployment.Entry.Environment)
			}

			// Check if transaction is executed
			isExecuted, ethTxHash, err := safeClient.IsTransactionExecuted(safeTxHash)
			if err != nil {
				fmt.Printf("    ERROR: %v\n", err)

				// Provide helpful context for common errors
				if strings.Contains(err.Error(), "transaction not found") {
					fmt.Printf("    HINT: This might happen if:\n")
					fmt.Printf("      - The Safe transaction was never created (check if Safe address is correct)\n")
					fmt.Printf("      - The transaction is on a different network\n")
					fmt.Printf("      - The Safe Transaction Service hasn't indexed it yet (try again later)\n")

					if deployment.Entry.Deployment.SafeAddress == "" || deployment.Entry.Deployment.SafeAddress == "0x0000000000000000000000000000000000000000" {
						fmt.Printf("      - WARNING: Safe address is missing! This deployment needs to be re-executed.\n")
					}
				}
				continue
			}

			if isExecuted && ethTxHash != nil {
				fmt.Printf("EXECUTED (tx: %s)\n", ethTxHash.Hex())

				// Update the deployment entry
				deployment.Entry.Deployment.Status = "deployed"
				deployment.Entry.Deployment.TxHash = ethTxHash

				// Update in registry
				chainID, _ := strconv.ParseUint(deployment.ChainID, 10, 64)
				if err := registryManager.UpdateDeployment(chainID, deployment.Entry); err != nil {
					fmt.Printf("    Warning: Failed to update registry: %v\n", err)
				}
			} else {
				// Get more details about the pending transaction
				tx, err := safeClient.GetTransaction(safeTxHash)
				if err == nil {
					fmt.Printf("    PENDING (%d/%d confirmations)\n", len(tx.Confirmations), tx.ConfirmationsRequired)
				} else {
					fmt.Printf("    PENDING (couldn't get confirmation details)\n")
				}
			}
		}
	}

	return nil
}

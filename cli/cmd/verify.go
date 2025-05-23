package cmd

import (
	"fmt"

	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/bogdan/fdeploy/cli/internal/verification"
	"github.com/spf13/cobra"
)

var (
	pending bool
	chainID uint64
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify contracts on block explorers",
	Long: `Verify contracts on block explorers and update registry status.

This command can verify individual contracts or process all pending
verifications in the registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := verifyContracts(); err != nil {
			checkError(err)
		}
		
		fmt.Println("✅ Verification completed")
	},
}

func init() {
	verifyCmd.Flags().BoolVar(&pending, "pending", false, "Verify all pending contracts")
	verifyCmd.Flags().Uint64Var(&chainID, "chain-id", 0, "Chain ID to verify contracts on")
}

func verifyContracts() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Initialize verification manager
	verificationManager := verification.NewManager(nil, registryManager)

	if pending {
		// Verify all pending contracts
		if chainID == 0 {
			return fmt.Errorf("chain-id is required when using --pending")
		}
		
		return verificationManager.VerifyPendingContracts(chainID)
	}

	// TODO: Implement individual contract verification
	fmt.Println("Individual contract verification not yet implemented")
	return nil
}
package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/internal/verification"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

var (
	pendingFlag bool
	forceFlag   bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify [contract|address]",
	Short: "Verify contracts on block explorers",
	Long: `Verify contracts on block explorers (Etherscan and Sourcify) and update registry status.

Examples:
  treb verify Counter               # Verify specific contract
  treb verify 0x1234...            # Verify by address
  treb verify --pending             # Verify all pending contracts
  treb verify Counter --force       # Re-verify even if already verified`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runVerify(args); err != nil {
			checkError(err)
		}
	},
}

func init() {
	verifyCmd.Flags().BoolVar(&pendingFlag, "pending", false, "Verify all pending contracts")
	verifyCmd.Flags().BoolVar(&forceFlag, "force", false, "Re-verify even if already verified")
}

func runVerify(args []string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Initialize network resolver
	networkResolver := network.NewResolver(".")

	// Initialize verification manager
	verificationManager := verification.NewManager(registryManager, networkResolver)

	if pendingFlag {
		// Verify all pending contracts
		return verifyPendingContracts(verificationManager, registryManager)
	}

	if len(args) == 0 {
		return fmt.Errorf("please provide a contract identifier or use --pending flag")
	}

	// Verify specific contract
	identifier := args[0]
	return verifySpecificContract(identifier, verificationManager, registryManager)
}

func verifyPendingContracts(verificationManager *verification.Manager, registryManager *registry.Manager) error {
	allDeployments := registryManager.GetAllDeployments()
	
	// Filter pending verifications
	var pendingDeployments []*registry.DeploymentInfo
	for _, deployment := range allDeployments {
		if deployment.Entry.Verification.Status == "pending" || 
		   (forceFlag && deployment.Entry.Verification.Status != "verified") {
			pendingDeployments = append(pendingDeployments, deployment)
		}
	}

	if len(pendingDeployments) == 0 {
		color.New(color.FgYellow).Println("No pending verifications found.")
		return nil
	}

	color.New(color.FgCyan, color.Bold).Printf("Found %d contracts to verify:\n", len(pendingDeployments))
	
	successCount := 0
	for _, deployment := range pendingDeployments {
		displayName := deployment.Entry.GetDisplayName()
		fmt.Printf("  • %s/%s/%s\n", deployment.NetworkName, deployment.Entry.Environment, displayName)
		
		err := verificationManager.VerifyContract(deployment)
		if err != nil {
			color.New(color.FgRed).Printf("    ✗ Failed: %v\n", err)
		} else {
			color.New(color.FgGreen).Printf("    ✓ Verified\n")
			successCount++
		}
	}

	fmt.Printf("\nVerification complete: %d/%d successful\n", successCount, len(pendingDeployments))
	return nil
}

func verifySpecificContract(identifier string, verificationManager *verification.Manager, registryManager *registry.Manager) error {
	// Use shared picker to find deployment
	deployment, err := pickDeployment(identifier, registryManager)
	if err != nil {
		return err
	}

	// Check if already verified
	if deployment.Entry.Verification.Status == "verified" && !forceFlag {
		color.New(color.FgYellow).Printf("Contract %s is already verified. Use --force to re-verify.\n", 
			deployment.Entry.GetDisplayName())
		return nil
	}

	displayName := deployment.Entry.GetDisplayName()
	color.New(color.FgCyan, color.Bold).Printf("Verifying %s/%s/%s...\n", 
		deployment.NetworkName, deployment.Entry.Environment, displayName)

	err = verificationManager.VerifyContract(deployment)
	if err != nil {
		color.New(color.FgRed).Printf("✗ Verification failed: %v\n", err)
		return err
	}

	color.New(color.FgGreen).Println("✓ Verification completed successfully!")
	
	// Show verification status
	showVerificationStatus(deployment)
	return nil
}

func showVerificationStatus(deployment *registry.DeploymentInfo) {
	if deployment.Entry.Verification.Verifiers == nil {
		return
	}

	fmt.Println("\nVerification Status:")
	for verifier, status := range deployment.Entry.Verification.Verifiers {
		switch status.Status {
		case "verified":
			color.New(color.FgGreen).Printf("  %s: ✓ Verified", strings.Title(verifier))
			if status.URL != "" {
				fmt.Printf(" - %s", status.URL)
			}
			fmt.Println()
		case "failed":
			color.New(color.FgRed).Printf("  %s: ✗ Failed", strings.Title(verifier))
			if status.Reason != "" {
				fmt.Printf(" - %s", status.Reason)
			}
			fmt.Println()
		case "pending":
			color.New(color.FgYellow).Printf("  %s: ⏳ Pending\n", strings.Title(verifier))
		}
	}
}
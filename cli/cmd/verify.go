package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/cli/pkg/verification"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	allFlag           bool
	forceFlag         bool
	contractPathFlag  string
	debugFlag         bool
	verifyNetworkFlag string
	verifyNamespaceFlag string
)

var verifyV1Cmd = &cobra.Command{
	Use:   "verify-v1 [contract|address]",
	Short: "Verify contracts on block explorers (legacy)",
	Long: `Verify contracts on block explorers (Etherscan and Sourcify) and update v1 registry status.

Examples:
  treb verify-v1 Counter               # Verify specific contract
  treb verify-v1 0x1234...            # Verify by address
  treb verify-v1 --all                 # Verify all unverified contracts (pending/failed)
  treb verify-v1 --all --force         # Re-verify all contracts including verified ones
  treb verify-v1 Counter --force       # Re-verify even if already verified
  treb verify-v1 Counter --network sepolia --namespace staging  # Verify with filters
  treb verify-v1 CounterProxy --contract-path "./src/Counter.sol:Counter"  # Verify with manual contract path`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runVerify(args); err != nil {
			checkError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(verifyV1Cmd)
	verifyV1Cmd.Flags().BoolVar(&allFlag, "all", false, "Verify all unverified contracts (pending/failed)")
	verifyV1Cmd.Flags().BoolVar(&forceFlag, "force", false, "Re-verify even if already verified")
	verifyV1Cmd.Flags().StringVar(&contractPathFlag, "contract-path", "", "Manual contract path (e.g., ./src/Contract.sol:Contract)")
	verifyV1Cmd.Flags().BoolVar(&debugFlag, "debug", false, "Show debug information including forge verify commands")
	verifyV1Cmd.Flags().StringVar(&verifyNetworkFlag, "network", "", "Filter by network name")
	verifyV1Cmd.Flags().StringVarP(&verifyNamespaceFlag, "namespace", "n", "", "Filter by namespace")
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

	if allFlag {
		// Verify all unverified contracts (pending/failed) or all contracts if force is used
		return verifyAllContracts(verificationManager, registryManager)
	}

	if len(args) == 0 {
		return fmt.Errorf("please provide a contract identifier or use --all flag")
	}

	// Verify specific contract
	identifier := args[0]
	return verifySpecificContract(identifier, verificationManager, registryManager, verifyNetworkFlag, verifyNamespaceFlag)
}

func verifyAllContracts(verificationManager *verification.Manager, registryManager *registry.Manager) error {
	allDeployments := registryManager.GetAllDeployments()

	// Filter contracts to verify based on verification status and deployment status
	var contractsToVerify []*registry.DeploymentInfo
	var skippedContracts []*registry.DeploymentInfo

	for _, deployment := range allDeployments {
		// Skip deployments that are not yet deployed
		if deployment.Entry.Deployment.Status != types.StatusExecuted {
			skippedContracts = append(skippedContracts, deployment)
			continue
		}

		status := deployment.Entry.Verification.Status

		if forceFlag {
			// With --force, verify all deployed contracts
			contractsToVerify = append(contractsToVerify, deployment)
		} else {
			// Without --force, verify only pending, failed, and partial contracts
			if status == "pending" || status == "failed" || status == "partial" || status == "" {
				contractsToVerify = append(contractsToVerify, deployment)
			}
			// Skip verified contracts unless force is used
		}
	}

	// Show skipped contracts first
	if len(skippedContracts) > 0 {
		color.New(color.FgCyan, color.Bold).Printf("Skipping %d pending deployments:\n", len(skippedContracts))
		for _, deployment := range skippedContracts {
			displayName := deployment.Entry.GetDisplayName()
			fmt.Printf("  ⏭️  %s/%s/%s (Status: %s)\n", deployment.NetworkName, deployment.Entry.Namespace, displayName, deployment.Entry.Deployment.Status)
		}
		fmt.Println()
	}

	if len(contractsToVerify) == 0 {
		if forceFlag {
			color.New(color.FgYellow).Println("No deployed contracts found to verify.")
		} else {
			color.New(color.FgYellow).Println("No unverified deployed contracts found. Use --force to re-verify all contracts.")
		}
		return nil
	}

	if forceFlag {
		color.New(color.FgCyan, color.Bold).Printf("Found %d deployed contracts to verify (including verified ones with --force):\n", len(contractsToVerify))
	} else {
		color.New(color.FgCyan, color.Bold).Printf("Found %d unverified deployed contracts to verify:\n", len(contractsToVerify))
	}

	successCount := 0
	for _, deployment := range contractsToVerify {
		displayName := deployment.Entry.GetDisplayName()
		status := deployment.Entry.Verification.Status

		// Show status indicator
		var statusIcon string
		switch status {
		case "verified":
			statusIcon = "🔄" // Re-verifying
		case "failed":
			statusIcon = "⚠️" // Retrying failed
		case "partial":
			statusIcon = "🔁" // Retrying partial
		case "pending":
			statusIcon = "⏳" // First attempt
		default:
			statusIcon = "🆕" // New verification
		}

		fmt.Printf("  %s %s/%s/%s\n", statusIcon, deployment.NetworkName, deployment.Entry.Namespace, displayName)

		// Start spinner for verification (unless debug mode or non-interactive)
		var s *spinner.Spinner
		if !debugFlag && !IsNonInteractive() {
			s = createSpinner(fmt.Sprintf("Verifying %s...", displayName))
		}

		var err error
		if debugFlag {
			err = verificationManager.VerifyContractWithDebug(deployment, true)
		} else {
			err = verificationManager.VerifyContract(deployment)
		}

		if s != nil {
			s.Stop()
		}

		if err != nil {
			color.New(color.FgRed).Printf("    ✗ Verification failed: %v\n", err)
		} else {
			color.New(color.FgGreen).Printf("    ✓ Verification completed\n")
			successCount++
		}
	}

	fmt.Printf("\nVerification complete: %d/%d successful\n", successCount, len(contractsToVerify))
	return nil
}

func verifySpecificContract(identifier string, verificationManager *verification.Manager, registryManager *registry.Manager, networkFilter, namespaceFilter string) error {
	// Create resolver context
	resolver := resolvers.NewContext(".", !IsNonInteractive())
	
	// Use resolver to find deployment with filters
	deployment, err := resolver.ResolveDeploymentWithFilters(identifier, registryManager, networkFilter, namespaceFilter)
	if err != nil {
		return err
	}

	// Check if already verified
	if deployment.Entry.Verification.Status == "verified" && !forceFlag {
		color.New(color.FgYellow).Printf("Contract %s is already verified. Use --force to re-verify.\n",
			deployment.Entry.GetDisplayName())
		return nil
	}

	// Handle manual contract path override
	var originalContractPath, originalSourceHash string
	var contractPathOverridden bool

	if contractPathFlag != "" {
		// Save original values
		originalContractPath = deployment.Entry.Metadata.ContractPath
		originalSourceHash = deployment.Entry.Metadata.SourceHash
		contractPathOverridden = true

		// Override with manual contract path
		deployment.Entry.Metadata.ContractPath = contractPathFlag

		// Calculate new source hash for the manual contract path
		if newSourceHash, err := calculateSourceHashFromPath(contractPathFlag); err == nil {
			deployment.Entry.Metadata.SourceHash = newSourceHash
			color.New(color.FgYellow).Printf("Using manual contract path: %s\n", contractPathFlag)
		} else {
			color.New(color.FgYellow).Printf("Warning: Could not calculate source hash for manual path: %v\n", err)
		}
	}

	displayName := deployment.Entry.GetDisplayName()

	// Start spinner for verification (unless debug mode or non-interactive)
	var s *spinner.Spinner
	if !debugFlag && !IsNonInteractive() {
		s = createSpinner(fmt.Sprintf("Verifying %s/%s/%s...",
			deployment.NetworkName, deployment.Entry.Namespace, displayName))
	}

	if debugFlag {
		err = verificationManager.VerifyContractWithDebug(deployment, true)
	} else {
		err = verificationManager.VerifyContract(deployment)
	}

	if s != nil {
		s.Stop()
	}

	if err != nil {
		// If verification failed and we overrode the contract path, restore original values
		if contractPathOverridden {
			deployment.Entry.Metadata.ContractPath = originalContractPath
			deployment.Entry.Metadata.SourceHash = originalSourceHash
		}
		color.New(color.FgRed).Printf("✗ Verification failed: %v\n", err)
		return err
	}

	color.New(color.FgGreen).Println("✓ Verification completed successfully!")

	// If verification succeeded and we used a manual contract path, save the updated metadata
	if contractPathOverridden {
		chainIDUint, parseErr := parseChainID(deployment.ChainID)
		if parseErr == nil {
			if updateErr := registryManager.UpdateDeployment(chainIDUint, deployment.Entry); updateErr == nil {
				color.New(color.FgGreen).Printf("✓ Updated registry with correct contract path: %s\n", contractPathFlag)
			} else {
				color.New(color.FgYellow).Printf("Warning: Could not update registry: %v\n", updateErr)
			}
		}
	}

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
			color.New(color.FgGreen).Printf("  %s: ✓ Verified", cases.Title(language.English).String(verifier))
			if status.URL != "" {
				fmt.Printf(" - %s", status.URL)
			}
			fmt.Println()
		case "failed":
			color.New(color.FgRed).Printf("  %s: ✗ Failed", cases.Title(language.English).String(verifier))
			if status.Reason != "" {
				fmt.Printf(" - %s", status.Reason)
			}
			fmt.Println()
		case "pending":
			color.New(color.FgYellow).Printf("  %s: ⏳ Pending\n", cases.Title(language.English).String(verifier))
		}
	}
}

// calculateSourceHashFromPath calculates the source hash for a given contract path
func calculateSourceHashFromPath(contractPath string) (string, error) {
	// Extract file path from contract path (format: ./path/to/Contract.sol:Contract)
	parts := strings.Split(contractPath, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid contract path format: %s", contractPath)
	}

	filePath := strings.TrimPrefix(parts[0], "./")

	// Read and hash the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read contract file: %w", err)
	}

	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// parseChainID parses a chain ID string to uint64
func parseChainID(chainIDStr string) (uint64, error) {
	return strconv.ParseUint(chainIDStr, 10, 64)
}

// createSpinner creates a new spinner with the given message
func createSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	_ = s.Color("cyan", "bold")
	s.Start()
	return s
}

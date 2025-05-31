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
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/cli/pkg/verification"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [deployment-id|address]",
	Short: "Verify contracts on block explorers",
	Long: `Verify contracts on block explorers (Etherscan and Sourcify) and update registry status.

Examples:
  treb verify Counter                      # Verify specific contract
  treb verify Counter:v2                   # Verify specific deployment by label
  treb verify staging/Counter              # Verify by namespace/contract
  treb verify 11155111/Counter             # Verify by chain/contract
  treb verify staging/11155111/Counter     # Verify by namespace/chain/contract
  treb verify 0x1234...                    # Verify by address (requires --chain)
  treb verify --all                        # Verify all unverified contracts (skip local)
  treb verify --all --force                # Re-verify all contracts including verified
  treb verify Counter --force              # Re-verify even if already verified
  treb verify Counter --chain 11155111 --namespace staging  # Verify with filters
  treb verify CounterProxy --contract-path "./src/Counter.sol:Counter"  # Manual contract path`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runVerify(cmd, args); err != nil {
			checkError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().Bool("all", false, "Verify all unverified contracts (pending/failed)")
	verifyCmd.Flags().Bool("force", false, "Re-verify even if already verified")
	verifyCmd.Flags().String("contract-path", "", "Manual contract path (e.g., ./src/Contract.sol:Contract)")
	verifyCmd.Flags().Bool("debug", false, "Show debug information including forge verify commands")
	verifyCmd.Flags().Uint64P("chain", "c", 0, "Filter by chain ID")
	verifyCmd.Flags().StringP("namespace", "n", "", "Filter by namespace")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Get flags
	allFlag, _ := cmd.Flags().GetBool("all")
	namespaceFlag, _ := cmd.Flags().GetString("namespace")

	// Initialize v2 registry manager
	manager, err := registry.NewManager(".")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Initialize network resolver
	networkResolver := network.NewResolver(".")

	// Initialize verification manager (for v2)
	verificationManager := verification.NewManager(manager, networkResolver)

	if allFlag {
		// Verify all unverified contracts (pending/failed) or all contracts if force is used
		return verifyAllContracts(cmd, verificationManager, manager)
	}

	if len(args) == 0 {
		return fmt.Errorf("please provide a deployment identifier or use --all flag")
	}

	// Verify specific contract
	identifier := args[0]
	chainID, _ := cmd.Flags().GetUint64("chain")
	return verifySpecificContract(cmd, identifier, verificationManager, manager, chainID, namespaceFlag)
}

func verifyAllContracts(cmd *cobra.Command, verificationManager *verification.Manager, manager *registry.Manager) error {
	// Get force flag
	forceFlag, _ := cmd.Flags().GetBool("force")
	debugFlag, _ := cmd.Flags().GetBool("debug")
	
	allDeployments := manager.GetAllDeployments()
	
	if debugFlag {
		fmt.Printf("DEBUG: Found %d total deployments in registry\n", len(allDeployments))
	}

	// Filter contracts to verify based on verification status and deployment status
	var contractsToVerify []*types.Deployment
	var skippedContracts []*types.Deployment

	for _, deployment := range allDeployments {
		// Skip local chain deployments
		if deployment.ChainID == 31337 {
			skippedContracts = append(skippedContracts, deployment)
			if debugFlag {
				fmt.Printf("  DEBUG: Skipping %s - Local chain (31337)\n", deployment.GetDisplayName())
			}
			continue
		}
		
		// Skip deployments that are not yet deployed
		// Check if deployment has a transaction and it's executed
		if deployment.TransactionID == "" {
			skippedContracts = append(skippedContracts, deployment)
			if debugFlag {
				fmt.Printf("  DEBUG: Skipping %s - No TransactionID\n", deployment.GetDisplayName())
			}
			continue
		}
		
		// Check transaction status
		tx, err := manager.GetTransaction(deployment.TransactionID)
		if err != nil {
			skippedContracts = append(skippedContracts, deployment)
			if debugFlag {
				fmt.Printf("  DEBUG: Skipping %s - Transaction not found: %s\n", deployment.GetDisplayName(), deployment.TransactionID)
			}
			continue
		}
		
		if tx.Status != types.TransactionStatusExecuted {
			skippedContracts = append(skippedContracts, deployment)
			if debugFlag {
				fmt.Printf("  DEBUG: Skipping %s - Transaction status: %s (not executed)\n", deployment.GetDisplayName(), tx.Status)
			}
			continue
		}
		
		if debugFlag {
			fmt.Printf("  DEBUG: Processing %s - TransactionID: %s, Status: EXECUTED, Verification Status: %s\n", 
				deployment.GetDisplayName(), deployment.TransactionID, deployment.Verification.Status)
		}

		status := deployment.Verification.Status

		if forceFlag {
			// With --force, verify all deployed contracts
			contractsToVerify = append(contractsToVerify, deployment)
		} else {
			// Without --force, verify only pending, failed, partial, and unverified contracts
			if status == types.VerificationStatusPending || 
			   status == types.VerificationStatusFailed || 
			   status == types.VerificationStatusPartial || 
			   status == types.VerificationStatusUnverified ||
			   status == "" {
				contractsToVerify = append(contractsToVerify, deployment)
			}
			// Skip verified contracts unless force is used
		}
	}

	// Show skipped contracts first
	if len(skippedContracts) > 0 {
		color.New(color.FgCyan, color.Bold).Printf("Skipping %d pending/undeployed contracts:\n", len(skippedContracts))
		for _, deployment := range skippedContracts {
			displayName := deployment.GetDisplayName()
			skipReason := "No TransactionID"
			
			// Determine skip reason
			if deployment.ChainID == 31337 {
				skipReason = "Local chain"
			} else if deployment.TransactionID != "" {
				// Try to get transaction to show specific status
				if tx, err := manager.GetTransaction(deployment.TransactionID); err == nil {
					skipReason = fmt.Sprintf("Transaction %s", tx.Status)
				} else {
					skipReason = "Transaction not found"
				}
			}
			fmt.Printf("  ⏭️  chain:%d/%s/%s (%s)\n", deployment.ChainID, deployment.Namespace, displayName, skipReason)
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
		displayName := deployment.GetDisplayName()
		status := deployment.Verification.Status

		// Show status indicator
		var statusIcon string
		switch status {
		case types.VerificationStatusVerified:
			statusIcon = "🔄" // Re-verifying
		case types.VerificationStatusFailed:
			statusIcon = "⚠️" // Retrying failed
		case types.VerificationStatusPartial:
			statusIcon = "🔁" // Retrying partial
		case types.VerificationStatusPending:
			statusIcon = "⏳" // First attempt
		default:
			statusIcon = "🆕" // New verification
		}

		fmt.Printf("  %s chain:%d/%s/%s\n", statusIcon, deployment.ChainID, deployment.Namespace, displayName)

		// Start spinner for verification (unless debug mode or non-interactive)
		var s *spinner.Spinner
		if !debugFlag && !IsNonInteractive() {
			s = createSpinner(fmt.Sprintf("Verifying %s...", displayName))
		}

		var err error
		if debugFlag {
			err = verificationManager.VerifyDeploymentWithDebug(deployment, true)
		} else {
			err = verificationManager.VerifyDeployment(deployment)
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

func verifySpecificContract(cmd *cobra.Command, identifier string, verificationManager *verification.Manager, manager *registry.Manager, chainIDFilter uint64, namespaceFilter string) error {
	// Get flags
	forceFlag, _ := cmd.Flags().GetBool("force")
	debugFlag, _ := cmd.Flags().GetBool("debug")
	contractPathFlag, _ := cmd.Flags().GetString("contract-path")
	
	var deployment *types.Deployment
	var err error

	// Check if identifier is an address (starts with 0x and is 42 chars)
	if strings.HasPrefix(identifier, "0x") && len(identifier) == 42 {
		// Look up by address
		if chainIDFilter == 0 {
			return fmt.Errorf("--chain flag is required when looking up by address")
		}
		deployment, err = manager.GetDeploymentByAddress(chainIDFilter, identifier)
		if err != nil {
			return fmt.Errorf("deployment not found at address %s on chain %d", identifier, chainIDFilter)
		}
	} else {
		// Parse deployment ID (could be Contract, Contract:label, namespace/Contract, etc.)
		deployments := manager.GetAllDeployments()
		
		// Filter by namespace if provided
		if namespaceFilter != "" {
			filtered := make([]*types.Deployment, 0)
			for _, d := range deployments {
				if d.Namespace == namespaceFilter {
					filtered = append(filtered, d)
				}
			}
			deployments = filtered
		}

		// Filter by chain if provided
		if chainIDFilter != 0 {
			filtered := make([]*types.Deployment, 0)
			for _, d := range deployments {
				if d.ChainID == chainIDFilter {
					filtered = append(filtered, d)
				}
			}
			deployments = filtered
		}

		// Look for matches based on various identifier formats
		matches := make([]*types.Deployment, 0)
		
		// Try to parse identifier parts
		parts := strings.Split(identifier, "/")
		
		for _, d := range deployments {
			matched := false
			
			// Simple match: just contract name or contract:label
			if d.ContractName == identifier || d.GetShortID() == identifier {
				matched = true
			}
			
			// Match namespace/contract or namespace/contract:label
			if len(parts) == 2 {
				namespace := parts[0]
				contractPart := parts[1]
				
				// Check if first part is a namespace
				if d.Namespace == namespace && (d.ContractName == contractPart || d.GetShortID() == contractPart) {
					matched = true
				}
				
				// Check if first part is a chain ID
				if chainID, err := strconv.ParseUint(parts[0], 10, 64); err == nil {
					if d.ChainID == chainID && (d.ContractName == contractPart || d.GetShortID() == contractPart) {
						matched = true
					}
				}
			}
			
			// Match namespace/chain/contract or similar complex patterns
			if len(parts) == 3 {
				// Could be namespace/chainID/contract
				if d.Namespace == parts[0] {
					if chainID, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
						if d.ChainID == chainID && (d.ContractName == parts[2] || d.GetShortID() == parts[2]) {
							matched = true
						}
					}
				}
			}
			
			// Match against the full deployment ID
			if d.ID == identifier {
				matched = true
			}
			
			if matched {
				matches = append(matches, d)
			}
		}

		if len(matches) == 0 {
			return fmt.Errorf("no deployments found matching '%s'", identifier)
		} else if len(matches) == 1 {
			deployment = matches[0]
		} else {
			// Multiple matches, use interactive picker
			selected, err := interactive.PickDeployment(matches, fmt.Sprintf("Multiple deployments found for '%s'", identifier))
			if err != nil {
				return err
			}
			deployment = selected
		}
	}

	// Check if already verified
	if deployment.Verification.Status == types.VerificationStatusVerified && !forceFlag {
		color.New(color.FgYellow).Printf("Contract %s is already verified. Use --force to re-verify.\n",
			deployment.GetDisplayName())
		return nil
	}

	// Handle manual contract path override
	var originalContractPath, originalSourceHash string
	var contractPathOverridden bool

	if contractPathFlag != "" {
		// Initialize metadata if needed
		if deployment.Metadata == nil {
			deployment.Metadata = &types.ContractMetadata{}
		}

		// Save original values
		originalContractPath = deployment.Metadata.ContractPath
		originalSourceHash = deployment.Metadata.SourceHash
		contractPathOverridden = true

		// Override with manual contract path
		deployment.Metadata.ContractPath = contractPathFlag

		// Calculate new source hash for the manual contract path
		if newSourceHash, err := calculateSourceHashFromPath(contractPathFlag); err == nil {
			deployment.Metadata.SourceHash = newSourceHash
			color.New(color.FgYellow).Printf("Using manual contract path: %s\n", contractPathFlag)
		} else {
			color.New(color.FgYellow).Printf("Warning: Could not calculate source hash for manual path: %v\n", err)
		}
	}

	displayName := deployment.GetDisplayName()

	// Start spinner for verification (unless debug mode or non-interactive)
	var s *spinner.Spinner
	if !debugFlag && !IsNonInteractive() {
		s = createSpinner(fmt.Sprintf("Verifying chain:%d/%s/%s...",
			deployment.ChainID, deployment.Namespace, displayName))
	}

	if debugFlag {
		err = verificationManager.VerifyDeploymentWithDebug(deployment, true)
	} else {
		err = verificationManager.VerifyDeployment(deployment)
	}

	if s != nil {
		s.Stop()
	}

	if err != nil {
		// If verification failed and we overrode the contract path, restore original values
		if contractPathOverridden {
			deployment.Metadata.ContractPath = originalContractPath
			deployment.Metadata.SourceHash = originalSourceHash
		}
		color.New(color.FgRed).Printf("✗ Verification failed: %v\n", err)
		return err
	}

	color.New(color.FgGreen).Println("✓ Verification completed successfully!")

	// If verification succeeded and we used a manual contract path, save the updated metadata
	if contractPathOverridden {
		if updateErr := manager.SaveDeployment(deployment); updateErr == nil {
			color.New(color.FgGreen).Printf("✓ Updated registry with correct contract path: %s\n", contractPathFlag)
		} else {
			color.New(color.FgYellow).Printf("Warning: Could not update registry: %v\n", updateErr)
		}
	}

	// Show verification status
	showVerificationStatus(deployment)
	return nil
}

func showVerificationStatus(deployment *types.Deployment) {
	if deployment.Verification.Verifiers == nil {
		return
	}

	fmt.Println("\nVerification Status:")
	for verifier, status := range deployment.Verification.Verifiers {
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

// createSpinner creates a new spinner with the given message
func createSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	_ = s.Color("cyan", "bold")
	s.Start()
	return s
}
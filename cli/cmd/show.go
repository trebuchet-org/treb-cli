package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/safe"
)

var showCmd = &cobra.Command{
	Use:   "show <contract|address>",
	Short: "Show deployment details",
	Long: `Display detailed information about a specific deployment.
Accepts contract name, address, or partial match.

Examples:
  treb show Counter
  treb show 0x1234...
  treb show Count`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]

		if err := showDeploymentByIdentifier(identifier); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// Add flags if needed in the future
}

func showDeploymentByIdentifier(identifier string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Get all deployments
	allDeployments := registryManager.GetAllDeployments()

	// Find matching deployments
	var matches []*registry.DeploymentInfo
	identifierLower := strings.ToLower(identifier)

	for _, deployment := range allDeployments {
		// Check if identifier is an address
		if strings.ToLower(deployment.Address.Hex()) == identifierLower {
			matches = append(matches, deployment)
			continue
		}

		// Check if identifier matches or is contained in display name
		displayName := deployment.Entry.GetDisplayName()
		if strings.EqualFold(displayName, identifier) || strings.Contains(strings.ToLower(displayName), identifierLower) {
			matches = append(matches, deployment)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("no deployment found matching '%s'", identifier)
	}

	// If single match, show it
	if len(matches) == 1 {
		return showDeploymentInfo(matches[0])
	}

	// Multiple matches - show selection
	fmt.Printf("Multiple deployments found matching '%s':\n\n", identifier)

	// Sort matches by network, then env, then contract name
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].NetworkName != matches[j].NetworkName {
			return matches[i].NetworkName < matches[j].NetworkName
		}
		if matches[i].Entry.Environment != matches[j].Entry.Environment {
			return matches[i].Entry.Environment < matches[j].Entry.Environment
		}
		return matches[i].Entry.ContractName < matches[j].Entry.ContractName
	})

	for i, match := range matches {
		displayName := match.Entry.GetDisplayName()
		fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Environment, displayName)
		fmt.Printf("%d. %s\n   Address: %s\n\n", i+1, fullId, match.Address.Hex())
	}

	// Ask user to select
	fmt.Print("Select deployment (1-", len(matches), "): ")
	var selection int
	fmt.Scanln(&selection)

	if selection < 1 || selection > len(matches) {
		return fmt.Errorf("invalid selection")
	}

	return showDeploymentInfo(matches[selection-1])
}

func showDeploymentInfo(deployment *registry.DeploymentInfo) error {
	// Get network info
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetworkByChainID(deployment.ChainID)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Display deployment information
	displayName := deployment.Entry.GetDisplayName()
	fmt.Printf("Deployment: %s\n", displayName)
	fmt.Printf("Environment: %s\n\n", deployment.Entry.Environment)

	// Basic deployment info
	fmt.Printf("Contract Information:\n")
	fmt.Printf("   Address: %s\n", deployment.Address.Hex())
	fmt.Printf("   Type: %s\n", deployment.Entry.Type)
	fmt.Printf("   Salt: %s\n", deployment.Entry.Salt)
	if deployment.Entry.InitCodeHash != "" {
		fmt.Printf("   Init Code Hash: %s\n", deployment.Entry.InitCodeHash)
	}
	fmt.Println()

	// Deployment details
	fmt.Printf("Deployment Details:\n")

	// Show deployment status
	status := deployment.Entry.Deployment.Status
	if status == "" {
		status = "deployed"
	}
	fmt.Printf("   Status: %s\n", status)

	// Safe-specific information for pending deployments
	if status == "pending_safe" {
		if deployment.Entry.Deployment.SafeAddress != "" {
			fmt.Printf("   Safe Address: %s\n", deployment.Entry.Deployment.SafeAddress)
		}
		if deployment.Entry.Deployment.SafeNonce > 0 {
			fmt.Printf("   Safe Nonce: %d\n", deployment.Entry.Deployment.SafeNonce)
		}
		if deployment.Entry.Deployment.SafeTxHash != nil {
			fmt.Printf("   Safe Tx Hash: %s\n", deployment.Entry.Deployment.SafeTxHash.Hex())

			// Try to get current confirmation status
			chainIDUint, err := strconv.ParseUint(deployment.ChainID, 10, 64)
			if err == nil {
				if safeClient, err := safe.NewClient(chainIDUint); err == nil {
					if tx, err := safeClient.GetTransaction(*deployment.Entry.Deployment.SafeTxHash); err == nil {
						fmt.Printf("   Confirmations: %d/%d\n", len(tx.Confirmations), tx.ConfirmationsRequired)
					}
				}
			}
		}
		fmt.Printf("   This deployment is pending execution in the Safe UI\n")
	} else {
		// Regular deployment info
		if deployment.Entry.Deployment.TxHash != nil && deployment.Entry.Deployment.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
			fmt.Printf("   Transaction: %s\n", deployment.Entry.Deployment.TxHash.Hex())
		}
		if deployment.Entry.Deployment.BlockNumber > 0 {
			fmt.Printf("   Block: %d\n", deployment.Entry.Deployment.BlockNumber)
		}
	}

	fmt.Printf("   Network: %s (Chain ID: %s)\n", networkInfo.Name, deployment.ChainID)
	fmt.Printf("   Timestamp: %s\n", deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05"))
	if deployment.Entry.Deployment.BroadcastFile != "" {
		fmt.Printf("   Broadcast File: %s\n", deployment.Entry.Deployment.BroadcastFile)
	}
	fmt.Println()

	// Metadata
	fmt.Printf("Contract Metadata:\n")
	if deployment.Entry.Metadata.ContractPath != "" {
		fmt.Printf("   Path: %s\n", deployment.Entry.Metadata.ContractPath)
	}
	fmt.Printf("   Compiler: %s\n", deployment.Entry.Metadata.Compiler)
	if deployment.Entry.Metadata.SourceCommit != "" {
		fmt.Printf("   Source Commit: %s\n", deployment.Entry.Metadata.SourceCommit)
	}
	if deployment.Entry.Metadata.SourceHash != "" {
		fmt.Printf("   Source Hash: %s\n", deployment.Entry.Metadata.SourceHash)
	}
	if len(deployment.Entry.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(deployment.Entry.Tags, ", "))
	}
	fmt.Println()

	// Verification status
	fmt.Printf("Verification:\n")
	fmt.Printf("   Status: %s\n", deployment.Entry.Verification.Status)
	if deployment.Entry.Verification.ExplorerUrl != "" {
		fmt.Printf("   Explorer: %s\n", deployment.Entry.Verification.ExplorerUrl)
	}
	if deployment.Entry.Verification.Reason != "" {
		fmt.Printf("   Reason: %s\n", deployment.Entry.Verification.Reason)
	}

	return nil
}
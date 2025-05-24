package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
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

	// Use shared picker
	deployment, err := pickDeployment(identifier, registryManager)
	if err != nil {
		return err
	}

	return showDeploymentInfo(deployment)
}

func showDeploymentInfo(deployment *registry.DeploymentInfo) error {
	// Get network info
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetworkByChainID(deployment.ChainID)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Color styles
	titleStyle := color.New(color.FgCyan, color.Bold)
	labelStyle := color.New(color.FgWhite, color.Bold)
	addressStyle := color.New(color.FgGreen, color.Bold)
	warningStyle := color.New(color.FgYellow, color.Bold)
	errorStyle := color.New(color.FgRed, color.Bold)
	successStyle := color.New(color.FgGreen)
	sectionStyle := color.New(color.FgWhite, color.Bold, color.Underline)
	
	// Header
	displayName := deployment.Entry.GetDisplayName()
	fmt.Println()
	titleStyle.Printf("Deployment Details: %s/%s/%s\n", networkInfo.Name, deployment.Entry.Environment, displayName)
	fmt.Println(strings.Repeat("=", 60))
	
	// Contract section
	fmt.Println()
	sectionStyle.Println("CONTRACT")
	labelStyle.Print("Address:      ")
	addressStyle.Println(deployment.Address.Hex())
	labelStyle.Print("Type:         ")
	fmt.Println(deployment.Entry.Type)
	if deployment.Entry.Metadata.ContractPath != "" {
		labelStyle.Print("Path:         ")
		fmt.Println(deployment.Entry.Metadata.ContractPath)
	}
	
	// Show proxy implementation details
	if deployment.Entry.Type == "proxy" && deployment.Entry.TargetContract != "" {
		labelStyle.Print("Implementation:")
		fmt.Println()
		
		// Find the implementation deployment
		registryManager, _ := registry.NewManager("deployments.json")
		chainIDUint, _ := strconv.ParseUint(deployment.ChainID, 10, 64)
		if impl := registryManager.GetDeployment(deployment.Entry.TargetContract, deployment.Entry.Environment, chainIDUint); impl != nil {
			labelStyle.Print("  Contract:   ")
			fmt.Println(deployment.Entry.TargetContract)
			labelStyle.Print("  Address:    ")
			addressStyle.Println(impl.Address.Hex())
			labelStyle.Print("  Identifier: ")
			fmt.Printf("%s/%s/%s\n", networkInfo.Name, deployment.Entry.Environment, deployment.Entry.TargetContract)
		} else {
			labelStyle.Print("  Contract:   ")
			fmt.Printf("%s (not found in registry)\n", deployment.Entry.TargetContract)
		}
	}
	
	// Network section
	fmt.Println()
	sectionStyle.Println("NETWORK")
	labelStyle.Print("Chain:        ")
	fmt.Printf("%s (ID: %s)\n", networkInfo.Name, deployment.ChainID)
	labelStyle.Print("Environment:  ")
	fmt.Println(deployment.Entry.Environment)
	
	// Deployment section
	fmt.Println()
	sectionStyle.Println("DEPLOYMENT")
	
	// Add salt and init code here
	if deployment.Entry.Salt != "" {
		labelStyle.Print("Salt:         ")
		fmt.Println(deployment.Entry.Salt)
	}
	if deployment.Entry.InitCodeHash != "" {
		labelStyle.Print("Init Code:    ")
		fmt.Println(deployment.Entry.InitCodeHash)
	}
	
	// Show deployer
	if deployment.Entry.Deployment.Deployer != "" {
		labelStyle.Print("Deployer:     ")
		addressStyle.Println(deployment.Entry.Deployment.Deployer)
	}
	
	// Show deployment status
	status := deployment.Entry.Deployment.Status
	if status == "" {
		status = "deployed"
	}
	labelStyle.Print("Status:       ")
	
	switch status {
	case "deployed":
		successStyle.Println("Deployed ✓")
	case "pending_safe":
		warningStyle.Println("Pending Safe Execution ⏳")
	default:
		fmt.Println(status)
	}
	
	// Safe-specific information for pending deployments
	if status == "pending_safe" {
		if deployment.Entry.Deployment.SafeAddress != "" {
			labelStyle.Print("Safe:         ")
			fmt.Println(deployment.Entry.Deployment.SafeAddress)
		}
		if deployment.Entry.Deployment.SafeTxHash != nil {
			labelStyle.Print("Safe Tx:      ")
			fmt.Println(deployment.Entry.Deployment.SafeTxHash.Hex())
			
			// Try to get current confirmation status
			chainIDUint, err := strconv.ParseUint(deployment.ChainID, 10, 64)
			if err == nil {
				if safeClient, err := safe.NewClient(chainIDUint); err == nil {
					if tx, err := safeClient.GetTransaction(*deployment.Entry.Deployment.SafeTxHash); err == nil {
						labelStyle.Print("Confirmations: ")
						if len(tx.Confirmations) >= tx.ConfirmationsRequired {
							successStyle.Printf("%d/%d (Ready to execute!)\n", len(tx.Confirmations), tx.ConfirmationsRequired)
						} else {
							fmt.Printf("%d/%d\n", len(tx.Confirmations), tx.ConfirmationsRequired)
						}
					}
				}
			}
		}
		if deployment.Entry.Deployment.SafeNonce > 0 {
			labelStyle.Print("Nonce:        ")
			fmt.Printf("%d\n", deployment.Entry.Deployment.SafeNonce)
		}
	} else {
		// Regular deployment info
		if deployment.Entry.Deployment.TxHash != nil && deployment.Entry.Deployment.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
			labelStyle.Print("Transaction:  ")
			fmt.Println(deployment.Entry.Deployment.TxHash.Hex())
		}
		if deployment.Entry.Deployment.BlockNumber > 0 {
			labelStyle.Print("Block:        ")
			fmt.Printf("%d\n", deployment.Entry.Deployment.BlockNumber)
		}
	}
	
	labelStyle.Print("Timestamp:    ")
	fmt.Println(deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05"))
	
	// Metadata section (only show if there's compiler or commit info)
	if deployment.Entry.Metadata.Compiler != "" || deployment.Entry.Metadata.SourceCommit != "" || len(deployment.Entry.Tags) > 0 {
		fmt.Println()
		sectionStyle.Println("METADATA")
		if deployment.Entry.Metadata.Compiler != "" {
			labelStyle.Print("Compiler:     ")
			fmt.Println(deployment.Entry.Metadata.Compiler)
		}
		if deployment.Entry.Metadata.SourceCommit != "" {
			labelStyle.Print("Commit:       ")
			fmt.Println(deployment.Entry.Metadata.SourceCommit)
		}
		if len(deployment.Entry.Tags) > 0 {
			labelStyle.Print("Tags:         ")
			color.New(color.FgCyan).Println(strings.Join(deployment.Entry.Tags, ", "))
		}
	}
	
	// Verification section
	fmt.Println()
	sectionStyle.Println("VERIFICATION")
	labelStyle.Print("Status:       ")
	if deployment.Entry.Verification.Status == "verified" {
		successStyle.Println("Verified ✓")
		if deployment.Entry.Verification.ExplorerUrl != "" {
			labelStyle.Print("Explorer:     ")
			fmt.Println(deployment.Entry.Verification.ExplorerUrl)
		}
	} else {
		errorStyle.Println("Not Verified ✗")
		if deployment.Entry.Verification.Reason != "" {
			labelStyle.Print("Reason:       ")
			fmt.Println(deployment.Entry.Verification.Reason)
		}
	}
	
	// Warnings and call-to-actions
	fmt.Println()
	warnings := []string{}
	
	// Pending safe warning
	if status == "pending_safe" && deployment.Entry.Deployment.SafeAddress != "" {
		safeUrl := getSafeUrl(networkInfo.Name, deployment.Entry.Deployment.SafeAddress)
		warnings = append(warnings, fmt.Sprintf("⚠️  This deployment is pending execution in Safe. Visit the Safe UI to execute:\n   %s", safeUrl))
	}
	
	// Verification warning
	if deployment.Entry.Verification.Status != "verified" && status == "deployed" {
		warnings = append(warnings, "⚠️  This contract is not verified. Run 'treb verify' to verify on block explorer.")
	}
	
	if len(warnings) > 0 {
		warningStyle.Println("ACTIONS REQUIRED:")
		for _, warning := range warnings {
			fmt.Println(warning)
		}
		fmt.Println()
	}

	return nil
}

// getSafeUrl returns the Safe UI URL for the given network and safe address
func getSafeUrl(network, safeAddress string) string {
	// Map network names to Safe UI subdomains
	safeNetworks := map[string]string{
		"mainnet":          "eth",
		"sepolia":          "sep",
		"optimism":         "oeth",
		"arbitrum":         "arb1",
		"arbitrum_sepolia": "arb-sep",
		"polygon":          "matic",
		"base":             "base",
		"base_sepolia":     "base-sep",
		"gnosis":           "gno",
		"celo":             "celo",
		"alfajores":        "celo-alf",
	}
	
	subdomain, ok := safeNetworks[network]
	if !ok {
		// Default to mainnet for unknown networks
		subdomain = "eth"
	}
	
	return fmt.Sprintf("https://app.safe.global/%s:%s/home", subdomain, safeAddress)
}
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	netpkg "github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var showCmd = &cobra.Command{
	Use:   "show <deployment>",
	Short: "Show detailed deployment information from registry",
	Long: `Show detailed information about a specific deployment.

You can specify deployments using:
- Contract name: "Counter"
- Contract with label: "Counter:v2"
- Namespace/contract: "staging/Counter"
- Chain/contract: "11155111/Counter"
- Full deployment ID: "production/1/Counter:v1"
- Contract address: "0x1234..."
- Alias: "MyCounter"

Examples:
  treb show Counter
  treb show Counter:v2
  treb show 0x1234567890abcdef...
  treb show production/1/Counter:v1
  treb show MyCounter`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deploymentRef := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")
		network, _ := cmd.Flags().GetString("network")
		namespace, _ := cmd.Flags().GetString("namespace")

		// Create registry manager
		manager, err := registry.NewManager(".")
		if err != nil {
			checkError(fmt.Errorf("failed to load registry: %w", err))
		}

		// Create deployment resolver
		resolver := resolvers.NewDeploymentsResolver(manager, !IsNonInteractive())

		// Resolve network to get chain ID if needed
		var chainID uint64
		if network != "" {
			netResolver, err := netpkg.NewResolver(".")
			if err != nil {
				checkError(fmt.Errorf("failed to create network resolver: %w", err))
			}
			networkInfo, err := netResolver.ResolveNetwork(network)
			if err != nil {
				checkError(fmt.Errorf("failed to resolve network: %w", err))
			}
			chainID = networkInfo.ChainID
		}

		// Use the deployment resolver to find the deployment
		deployment, err := resolver.ResolveDeployment(deploymentRef, manager, chainID, namespace)
		if err != nil {
			checkError(fmt.Errorf("failed to resolve deployment: %w", err))
		}

		// Get associated transaction
		var transaction *types.Transaction
		if deployment.TransactionID != "" {
			transaction, _ = manager.GetTransaction(deployment.TransactionID)
		}

		// Output JSON if requested
		if jsonOutput {
			output := map[string]interface{}{
				"deployment":  deployment,
				"transaction": transaction,
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				checkError(err)
			}
			fmt.Println(string(data))
			return
		}

		// Display deployment details
		displayDeployment(deployment, transaction, manager)
	},
}

func displayDeployment(dep *types.Deployment, tx *types.Transaction, manager *registry.Manager) {
	// Header
	color.New(color.FgCyan, color.Bold).Printf("Deployment: %s\n", dep.ID)
	fmt.Println(strings.Repeat("=", 80))

	// Basic Info
	fmt.Println("\nBasic Information:")
	fmt.Printf("  Contract: %s\n", color.New(color.FgYellow).Sprint(dep.ContractDisplayName()))
	fmt.Printf("  Address: %s\n", dep.Address)
	fmt.Printf("  Type: %s\n", dep.Type)
	fmt.Printf("  Namespace: %s\n", dep.Namespace)

	// Try to get network name for chain ID
	networkName := fmt.Sprintf("%d", dep.ChainID)
	if name, err := netpkg.GetNetworkByChainID(".", dep.ChainID); err == nil {
		networkName = fmt.Sprintf("%s (chain ID: %d)", name, dep.ChainID)
	}
	fmt.Printf("  Network: %s\n", networkName)

	if dep.Label != "" {
		fmt.Printf("  Label: %s\n", color.New(color.FgMagenta).Sprint(dep.Label))
	}

	// Deployment Strategy
	fmt.Println("\nDeployment Strategy:")
	fmt.Printf("  Method: %s\n", dep.DeploymentStrategy.Method)
	if dep.DeploymentStrategy.Factory != "" {
		fmt.Printf("  Factory: %s\n", dep.DeploymentStrategy.Factory)
	}
	if dep.DeploymentStrategy.Salt != "" && dep.DeploymentStrategy.Salt != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		fmt.Printf("  Salt: %s\n", dep.DeploymentStrategy.Salt)
	}
	if dep.DeploymentStrategy.Entropy != "" {
		fmt.Printf("  Entropy: %s\n", dep.DeploymentStrategy.Entropy)
	}
	if dep.DeploymentStrategy.InitCodeHash != "" {
		fmt.Printf("  Init Code Hash: %s\n", dep.DeploymentStrategy.InitCodeHash)
	}

	// Proxy Information
	if dep.ProxyInfo != nil {
		fmt.Println("\nProxy Information:")
		fmt.Printf("  Type: %s\n", dep.ProxyInfo.Type)

		// Try to resolve implementation contract details
		implDisplay := dep.ProxyInfo.Implementation
		if implDep, err := manager.GetDeploymentByAddress(dep.ChainID, dep.ProxyInfo.Implementation); err == nil {
			implDisplay = fmt.Sprintf("%s at %s",
				color.New(color.FgYellow, color.Bold).Sprint(implDep.ContractDisplayName()),
				dep.ProxyInfo.Implementation,
			)
			// Also show the implementation deployment ID
			fmt.Printf("  Implementation: %s\n", implDisplay)
			fmt.Printf("  Implementation ID: %s\n", color.New(color.FgCyan).Sprint(implDep.ID))
		} else {
			fmt.Printf("  Implementation: %s\n", implDisplay)
		}

		if dep.ProxyInfo.Admin != "" {
			fmt.Printf("  Admin: %s\n", dep.ProxyInfo.Admin)
		}
		if len(dep.ProxyInfo.History) > 0 {
			fmt.Println("  Upgrade History:")
			for i, upgrade := range dep.ProxyInfo.History {
				// Try to resolve the implementation name
				implName := upgrade.ImplementationID
				if histImpl, err := manager.GetDeployment(upgrade.ImplementationID); err == nil {
					implName = fmt.Sprintf("%s (%s)", histImpl.ContractDisplayName(), upgrade.ImplementationID)
				}
				fmt.Printf("    %d. %s (upgraded at %s)\n",
					i+1,
					implName,
					upgrade.UpgradedAt.Format("2006-01-02 15:04:05"),
				)
			}
		}
	}

	// Artifact Information
	fmt.Println("\nArtifact Information:")
	fmt.Printf("  Path: %s\n", dep.Artifact.Path)
	fmt.Printf("  Compiler: %s\n", dep.Artifact.CompilerVersion)
	if dep.Artifact.BytecodeHash != "" {
		fmt.Printf("  Bytecode Hash: %s\n", dep.Artifact.BytecodeHash)
	}
	if dep.Artifact.ScriptPath != "" {
		fmt.Printf("  Script: %s\n", dep.Artifact.ScriptPath)
	}
	if dep.Artifact.GitCommit != "" {
		fmt.Printf("  Git Commit: %s\n", dep.Artifact.GitCommit)
	}

	// Verification Status
	fmt.Println("\nVerification Status:")
	status := dep.Verification.Status
	statusColor := color.FgRed
	if status == types.VerificationStatusVerified {
		statusColor = color.FgGreen
	}
	fmt.Printf("  Status: %s\n", color.New(statusColor).Sprint(status))
	if dep.Verification.EtherscanURL != "" {
		fmt.Printf("  Etherscan: %s\n", dep.Verification.EtherscanURL)
	}
	if dep.Verification.VerifiedAt != nil {
		fmt.Printf("  Verified At: %s\n", dep.Verification.VerifiedAt.Format("2006-01-02 15:04:05"))
	}

	// Transaction Information
	if tx != nil {
		fmt.Println("\nTransaction Information:")
		fmt.Printf("  Hash: %s\n", tx.Hash)
		fmt.Printf("  Status: %s\n", tx.Status)
		fmt.Printf("  Sender: %s\n", tx.Sender)
		if tx.BlockNumber > 0 {
			fmt.Printf("  Block: %d\n", tx.BlockNumber)
		}
		if tx.SafeContext != nil {
			fmt.Println("  Safe Transaction:")
			fmt.Printf("    Safe: %s\n", tx.SafeContext.SafeAddress)
			fmt.Printf("    Safe Tx Hash: %s\n", tx.SafeContext.SafeTxHash)
			fmt.Printf("    Proposer: %s\n", tx.SafeContext.ProposerAddress)
		}
	}

	// Metadata
	if len(dep.Tags) > 0 {
		fmt.Println("\nTags:")
		for _, tag := range dep.Tags {
			fmt.Printf("  - %s\n", tag)
		}
	}

	// Timestamps
	fmt.Println("\nTimestamps:")
	fmt.Printf("  Created: %s\n", dep.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Updated: %s\n", dep.UpdatedAt.Format("2006-01-02 15:04:05"))
}

func init() {
	rootCmd.AddCommand(showCmd)

	// Output format flag
	showCmd.Flags().Bool("json", false, "Output in JSON format")

	// Network and namespace flags for better resolution
	showCmd.Flags().StringP("network", "n", "", "Network to use (e.g., mainnet, sepolia)")
	showCmd.Flags().String("namespace", "", "Namespace to use (defaults to current context)")
}

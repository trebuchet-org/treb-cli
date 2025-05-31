package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	registryv2 "github.com/trebuchet-org/treb-cli/cli/pkg/registry/v2"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var showCmd = &cobra.Command{
	Use:   "show <deployment-id>",
	Short: "Show detailed deployment information from registry",
	Long: `Show detailed information about a specific deployment.

The deployment ID format is: <namespace>/<chain-id>/<contract-name>:<label>

Examples:
  treb show production/1/Counter:v1
  treb show staging/31337/Token:usdc`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deploymentID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Create registry manager
		manager, err := registryv2.NewManager(".")
		if err != nil {
			checkError(fmt.Errorf("failed to load registry: %w", err))
		}

		// Get deployment
		deployment, err := manager.GetDeployment(deploymentID)
		if err != nil {
			// Try to find by partial match
			deployment = findDeploymentByPartialID(manager, deploymentID)
			if deployment == nil {
				checkError(fmt.Errorf("deployment not found: %s", deploymentID))
			}
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

func findDeploymentByPartialID(manager *registryv2.Manager, partialID string) *types.Deployment {
	allDeployments := manager.GetAllDeployments()
	
	var matches []*types.Deployment
	for _, dep := range allDeployments {
		if strings.Contains(dep.ID, partialID) {
			matches = append(matches, dep)
		}
	}

	if len(matches) == 1 {
		return matches[0]
	}
	
	if len(matches) > 1 {
		fmt.Println("Multiple deployments match:")
		for _, dep := range matches {
			fmt.Printf("  - %s\n", dep.ID)
		}
		checkError(fmt.Errorf("ambiguous deployment ID: %s", partialID))
	}

	return nil
}

func displayDeployment(dep *types.Deployment, tx *types.Transaction, manager *registryv2.Manager) {
	// Header
	color.New(color.FgCyan, color.Bold).Printf("Deployment: %s\n", dep.ID)
	fmt.Println(strings.Repeat("=", 80))

	// Basic Info
	fmt.Println("\nBasic Information:")
	fmt.Printf("  Contract: %s\n", color.New(color.FgYellow).Sprint(dep.ContractName))
	fmt.Printf("  Address: %s\n", dep.Address)
	fmt.Printf("  Type: %s\n", dep.Type)
	fmt.Printf("  Namespace: %s\n", dep.Namespace)
	fmt.Printf("  Chain ID: %d\n", dep.ChainID)
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
				color.New(color.FgYellow, color.Bold).Sprint(implDep.ContractName),
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
					implName = fmt.Sprintf("%s (%s)", histImpl.ContractName, upgrade.ImplementationID)
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
	switch status {
	case types.VerificationStatusVerified:
		statusColor = color.FgGreen
	case types.VerificationStatusPending:
		statusColor = color.FgYellow
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
}
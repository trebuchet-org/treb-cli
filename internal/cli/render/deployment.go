package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// DeploymentRenderer renders detailed information about a single deployment
type DeploymentRenderer struct {
	out   io.Writer
	color bool
}

// NewDeploymentRenderer creates a new deployment renderer
func NewDeploymentRenderer(out io.Writer, color bool) *DeploymentRenderer {
	return &DeploymentRenderer{
		out:   out,
		color: color,
	}
}

// RenderDeployment renders detailed deployment information
func (r *DeploymentRenderer) RenderDeployment(deployment *models.Deployment) error {
	// Header
	color.New(color.FgCyan, color.Bold).Fprintf(r.out, "Deployment: %s\n", deployment.ID)
	fmt.Fprintln(r.out, strings.Repeat("=", 80))

	// Basic Info
	fmt.Fprintln(r.out, "\nBasic Information:")
	fmt.Fprintf(r.out, "  Contract: %s\n", color.New(color.FgYellow).Sprint(deployment.ContractDisplayName()))
	fmt.Fprintf(r.out, "  Address: %s\n", deployment.Address)
	fmt.Fprintf(r.out, "  Type: %s\n", deployment.Type)
	fmt.Fprintf(r.out, "  Namespace: %s\n", deployment.Namespace)

	// Network name - for now just show chain ID
	// TODO: Resolve network name when network resolver is available
	networkName := fmt.Sprintf("%d", deployment.ChainID)
	fmt.Fprintf(r.out, "  Network: %s\n", networkName)

	if deployment.Label != "" {
		fmt.Fprintf(r.out, "  Label: %s\n", color.New(color.FgMagenta).Sprint(deployment.Label))
	}

	// Deployment Strategy
	fmt.Fprintln(r.out, "\nDeployment Strategy:")
	fmt.Fprintf(r.out, "  Method: %s\n", deployment.DeploymentStrategy.Method)
	if deployment.DeploymentStrategy.Factory != "" {
		fmt.Fprintf(r.out, "  Factory: %s\n", deployment.DeploymentStrategy.Factory)
	}
	if deployment.DeploymentStrategy.Salt != "" && deployment.DeploymentStrategy.Salt != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		fmt.Fprintf(r.out, "  Salt: %s\n", deployment.DeploymentStrategy.Salt)
	}
	if deployment.DeploymentStrategy.Entropy != "" {
		fmt.Fprintf(r.out, "  Entropy: %s\n", deployment.DeploymentStrategy.Entropy)
	}
	if deployment.DeploymentStrategy.InitCodeHash != "" {
		fmt.Fprintf(r.out, "  Init Code Hash: %s\n", deployment.DeploymentStrategy.InitCodeHash)
	}

	// Proxy Information
	if deployment.ProxyInfo != nil {
		fmt.Fprintln(r.out, "\nProxy Information:")
		fmt.Fprintf(r.out, "  Type: %s\n", deployment.ProxyInfo.Type)

		// Show implementation details
		implDisplay := deployment.ProxyInfo.Implementation
		if deployment.Implementation != nil {
			implDisplay = fmt.Sprintf("%s at %s",
				color.New(color.FgYellow, color.Bold).Sprint(deployment.Implementation.ContractDisplayName()),
				deployment.ProxyInfo.Implementation,
			)
			fmt.Fprintf(r.out, "  Implementation: %s\n", implDisplay)
			fmt.Fprintf(r.out, "  Implementation ID: %s\n", color.New(color.FgCyan).Sprint(deployment.Implementation.ID))
		} else {
			fmt.Fprintf(r.out, "  Implementation: %s\n", implDisplay)
		}

		if deployment.ProxyInfo.Admin != "" {
			fmt.Fprintf(r.out, "  Admin: %s\n", deployment.ProxyInfo.Admin)
		}

		if len(deployment.ProxyInfo.History) > 0 {
			fmt.Fprintln(r.out, "  Upgrade History:")
			for i, upgrade := range deployment.ProxyInfo.History {
				// For now just show the implementation ID
				// TODO: Resolve implementation names when we have access to the registry
				implName := upgrade.ImplementationID
				fmt.Fprintf(r.out, "    %d. %s (upgraded at %s)\n",
					i+1,
					implName,
					upgrade.UpgradedAt.Format("2006-01-02 15:04:05"),
				)
			}
		}
	}

	// Artifact Information
	fmt.Fprintln(r.out, "\nArtifact Information:")
	fmt.Fprintf(r.out, "  Path: %s\n", deployment.Artifact.Path)
	fmt.Fprintf(r.out, "  Compiler: %s\n", deployment.Artifact.CompilerVersion)
	if deployment.Artifact.BytecodeHash != "" {
		fmt.Fprintf(r.out, "  Bytecode Hash: %s\n", deployment.Artifact.BytecodeHash)
	}
	if deployment.Artifact.ScriptPath != "" {
		fmt.Fprintf(r.out, "  Script: %s\n", deployment.Artifact.ScriptPath)
	}
	if deployment.Artifact.GitCommit != "" {
		fmt.Fprintf(r.out, "  Git Commit: %s\n", deployment.Artifact.GitCommit)
	}

	// Verification Status
	fmt.Fprintln(r.out, "\nVerification Status:")
	status := deployment.Verification.Status
	statusColor := color.FgRed
	if status == models.VerificationStatusVerified {
		statusColor = color.FgGreen
	}
	fmt.Fprintf(r.out, "  Status: %s\n", color.New(statusColor).Sprint(status))
	if deployment.Verification.EtherscanURL != "" {
		fmt.Fprintf(r.out, "  Etherscan: %s\n", deployment.Verification.EtherscanURL)
	}
	if deployment.Verification.VerifiedAt != nil {
		fmt.Fprintf(r.out, "  Verified At: %s\n", deployment.Verification.VerifiedAt.Format("2006-01-02 15:04:05"))
	}

	// Transaction Information
	if deployment.Transaction != nil {
		tx := deployment.Transaction
		fmt.Fprintln(r.out, "\nTransaction Information:")
		fmt.Fprintf(r.out, "  Hash: %s\n", tx.Hash)
		fmt.Fprintf(r.out, "  Status: %s\n", tx.Status)
		fmt.Fprintf(r.out, "  Sender: %s\n", tx.Sender)
		if tx.BlockNumber > 0 {
			fmt.Fprintf(r.out, "  Block: %d\n", tx.BlockNumber)
		}
		if tx.SafeContext != nil {
			fmt.Fprintln(r.out, "  Safe Transaction:")
			fmt.Fprintf(r.out, "    Safe: %s\n", tx.SafeContext.SafeAddress)
			fmt.Fprintf(r.out, "    Safe Tx Hash: %s\n", tx.SafeContext.SafeTxHash)
			fmt.Fprintf(r.out, "    Proposer: %s\n", tx.SafeContext.ProposerAddress)
		}
	}

	// Metadata
	if len(deployment.Tags) > 0 {
		fmt.Fprintln(r.out, "\nTags:")
		for _, tag := range deployment.Tags {
			fmt.Fprintf(r.out, "  - %s\n", tag)
		}
	}

	// Timestamps
	fmt.Fprintln(r.out, "\nTimestamps:")
	fmt.Fprintf(r.out, "  Created: %s\n", deployment.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(r.out, "  Updated: %s\n", deployment.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}


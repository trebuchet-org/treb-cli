package verification

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// Verifier handles contract verification across multiple block explorers
type Verifier struct {
	projectRoot string
	debug       bool
}

// NewVerifier creates a new verifier
func NewVerifier(cfg *config.RuntimeConfig) (*Verifier, error) {
	return &Verifier{
		projectRoot: cfg.ProjectRoot,
		debug:       cfg.Debug,
	}, nil
}

// Verify performs contract verification
func (v *Verifier) Verify(ctx context.Context, deployment *models.Deployment, network *config.Network, verifiers []string, blockscoutVerifierURL string) error {
	// Initialize verification status
	if deployment.Verification.Verifiers == nil {
		deployment.Verification.Verifiers = make(map[string]models.VerifierStatus)
	}

	// Track verification errors
	var verificationErrors []string

	// Helper to check if verifier is enabled
	isEnabled := func(name string) bool {
		for _, v := range verifiers {
			if v == name {
				return true
			}
		}
		return false
	}

	// Verify on Etherscan if enabled
	if isEnabled("etherscan") {
		etherscanErr := v.verifyOnEtherscan(ctx, deployment, network)
		if etherscanErr != nil {
			deployment.Verification.Verifiers["etherscan"] = models.VerifierStatus{
				Status: "failed",
				Reason: etherscanErr.Error(),
			}
			verificationErrors = append(verificationErrors, fmt.Sprintf("etherscan: %v", etherscanErr))
		} else {
			deployment.Verification.Verifiers["etherscan"] = models.VerifierStatus{
				Status: "verified",
				URL:    v.buildEtherscanURL(network, deployment.Address),
			}
		}
	}

	// Verify on Blockscout if enabled
	if isEnabled("blockscout") {
		blockscoutErr := v.verifyOnBlockscout(ctx, deployment, network, blockscoutVerifierURL)
		if blockscoutErr != nil {
			deployment.Verification.Verifiers["blockscout"] = models.VerifierStatus{
				Status: "failed",
				Reason: blockscoutErr.Error(),
			}
			verificationErrors = append(verificationErrors, fmt.Sprintf("blockscout: %v", blockscoutErr))
		} else {
			deployment.Verification.Verifiers["blockscout"] = models.VerifierStatus{
				Status: "verified",
				URL:    v.buildBlockscoutURL(network, deployment.Address),
			}
		}
	}

	// Verify on Sourcify if enabled
	if isEnabled("sourcify") {
		sourcifyErr := v.verifyOnSourceify(ctx, deployment, network)
		if sourcifyErr != nil {
			deployment.Verification.Verifiers["sourcify"] = models.VerifierStatus{
				Status: "failed",
				Reason: sourcifyErr.Error(),
			}
			verificationErrors = append(verificationErrors, fmt.Sprintf("sourcify: %v", sourcifyErr))
		} else {
			deployment.Verification.Verifiers["sourcify"] = models.VerifierStatus{
				Status: "verified",
				URL:    v.buildSourceifyURL(network, deployment.Address),
			}
		}
	}

	// Update overall status
	v.updateOverallStatus(deployment)

	// Return error if all verifications failed
	if deployment.Verification.Status == models.VerificationStatusFailed {
		return fmt.Errorf("all verifications failed: %s", strings.Join(verificationErrors, "; "))
	}

	return nil
}

// verifyOnEtherscan performs verification on Etherscan
func (v *Verifier) verifyOnEtherscan(ctx context.Context, deployment *models.Deployment, network *config.Network) error {
	// Get constructor args from deployment strategy
	constructorArgs := deployment.DeploymentStrategy.ConstructorArgs
	if constructorArgs != "" && strings.HasPrefix(constructorArgs, "0x") {
		constructorArgs = constructorArgs[2:] // Remove 0x prefix
	}

	// Get contract path from artifact
	contractPath := fmt.Sprintf("%s:%s", deployment.Artifact.Path, deployment.ContractName)

	// Build the forge verify-contract command
	args := []string{
		"verify-contract",
		deployment.Address,
		contractPath,
		"--chain-id", fmt.Sprintf("%d", network.ChainID),
		"--verifier", "etherscan",
		"--watch",
	}

	// Add verifier URL if custom explorer is configured
	if network.ExplorerURL != "" {
		args = append(args, "--verifier-url", network.ExplorerURL)
	}

	// Add API key if available from environment
	if apiKey := os.Getenv("ETHERSCAN_API_KEY"); apiKey != "" {
		args = append(args, "--etherscan-api-key", apiKey)
	}

	// Add compiler version if available
	if deployment.Artifact.CompilerVersion != "" {
		args = append(args, "--compiler-version", deployment.Artifact.CompilerVersion)
	}

	// Add constructor args if available
	if constructorArgs != "" {
		args = append(args, "--constructor-args", constructorArgs)
	}

	// Execute the command
	return v.executeForgeVerify(ctx, args)
}

// verifyOnSourceify performs verification on Sourcify
func (v *Verifier) verifyOnSourceify(ctx context.Context, deployment *models.Deployment, network *config.Network) error {
	// Get contract path from artifact
	contractPath := fmt.Sprintf("%s:%s", deployment.Artifact.Path, deployment.ContractName)

	// Build the forge verify-contract command for Sourcify
	args := []string{
		"verify-contract",
		deployment.Address,
		contractPath,
		"--chain-id", fmt.Sprintf("%d", network.ChainID),
		"--verifier", "sourcify",
		"--watch",
	}

	// Add compiler version if available
	if deployment.Artifact.CompilerVersion != "" {
		args = append(args, "--compiler-version", deployment.Artifact.CompilerVersion)
	}

	// Add constructor args if available
	constructorArgs := deployment.DeploymentStrategy.ConstructorArgs
	if constructorArgs != "" && strings.HasPrefix(constructorArgs, "0x") {
		constructorArgs = constructorArgs[2:] // Remove 0x prefix
	}
	if constructorArgs != "" {
		args = append(args, "--constructor-args", constructorArgs)
	}

	// Execute the command
	return v.executeForgeVerify(ctx, args)
}

// verifyOnBlockscout performs verification on Blockscout
func (v *Verifier) verifyOnBlockscout(ctx context.Context, deployment *models.Deployment, network *config.Network, blockscoutVerifierURL string) error {
	// Get constructor args from deployment strategy
	constructorArgs := deployment.DeploymentStrategy.ConstructorArgs
	if constructorArgs != "" && strings.HasPrefix(constructorArgs, "0x") {
		constructorArgs = constructorArgs[2:] // Remove 0x prefix
	}

	// Get contract path from artifact
	contractPath := fmt.Sprintf("%s:%s", deployment.Artifact.Path, deployment.ContractName)

	// Build the forge verify-contract command for Blockscout
	args := []string{
		"verify-contract",
		deployment.Address,
		contractPath,
		"--chain-id", fmt.Sprintf("%d", network.ChainID),
		"--verifier", "blockscout",
		"--watch",
	}

	// Add verifier URL if provided (for custom/self-hosted Blockscout instances)
	if blockscoutVerifierURL != "" {
		args = append(args, "--verifier-url", blockscoutVerifierURL)
	}
	// Note: If no custom URL is provided, Foundry automatically determines
	// the Blockscout instance based on chain ID

	// Add compiler version if available
	if deployment.Artifact.CompilerVersion != "" {
		args = append(args, "--compiler-version", deployment.Artifact.CompilerVersion)
	}

	// Add constructor args if available
	if constructorArgs != "" {
		args = append(args, "--constructor-args", constructorArgs)
	}

	// Execute the command
	return v.executeForgeVerify(ctx, args)
}

// executeForgeVerify executes a forge verify-contract command
func (v *Verifier) executeForgeVerify(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "forge", args...)
	cmd.Dir = v.projectRoot

	// Print the command if debug is enabled
	if v.debug {
		cmdStr := fmt.Sprintf("forge %s", strings.Join(args, " "))
		fmt.Fprintf(os.Stderr, "\n[DEBUG] Executing: %s\n", cmdStr)
		fmt.Fprintf(os.Stderr, "[DEBUG] Working directory: %s\n\n", v.projectRoot)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Parse the output for specific error messages
		outputStr := string(output)
		if strings.Contains(outputStr, "Already Verified") ||
			strings.Contains(outputStr, "is already verified") ||
			strings.Contains(outputStr, "already verified") {
			// Contract is already verified, not an error
			return nil
		}
		return fmt.Errorf("verification failed: %s", strings.TrimSpace(outputStr))
	}

	// Check if verification was successful
	outputStr := string(output)
	if strings.Contains(outputStr, "Contract successfully verified") ||
		strings.Contains(outputStr, "Already Verified") ||
		strings.Contains(outputStr, "is already verified") ||
		strings.Contains(outputStr, "already verified") {
		return nil
	}

	return fmt.Errorf("verification status unclear: %s", strings.TrimSpace(outputStr))
}

// buildEtherscanURL builds the Etherscan URL for a contract
func (v *Verifier) buildEtherscanURL(network *config.Network, address string) string {
	if network.ExplorerURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/address/%s#code", network.ExplorerURL, address)
}

// buildSourceifyURL builds the Sourcify URL for a contract
func (v *Verifier) buildSourceifyURL(network *config.Network, address string) string {
	return fmt.Sprintf("https://sourcify.dev/#/lookup/%s", address)
}

// buildBlockscoutURL builds the Blockscout URL for a contract
func (v *Verifier) buildBlockscoutURL(network *config.Network, address string) string {
	if network.ExplorerURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/address/%s", network.ExplorerURL, address)
}

// updateOverallStatus updates the overall verification status based on individual verifiers
func (v *Verifier) updateOverallStatus(deployment *models.Deployment) {
	if deployment.Verification.Verifiers == nil {
		deployment.Verification.Status = models.VerificationStatusUnverified
		return
	}

	verifiedCount := 0
	failedCount := 0
	totalCount := len(deployment.Verification.Verifiers)

	for _, status := range deployment.Verification.Verifiers {
		switch status.Status {
		case "verified":
			verifiedCount++
		case "failed":
			failedCount++
		}
	}

	if verifiedCount == totalCount {
		deployment.Verification.Status = models.VerificationStatusVerified
		// Set the explorer URL to the first verified one
		for _, status := range deployment.Verification.Verifiers {
			if status.Status == "verified" && status.URL != "" {
				deployment.Verification.EtherscanURL = status.URL
				break
			}
		}
	} else if verifiedCount > 0 {
		deployment.Verification.Status = models.VerificationStatusPartial
	} else if failedCount == totalCount {
		deployment.Verification.Status = models.VerificationStatusFailed
		// Combine all failure reasons
		var reasons []string
		for verifier, status := range deployment.Verification.Verifiers {
			if status.Status == "failed" && status.Reason != "" {
				reasons = append(reasons, fmt.Sprintf("%s: %s", verifier, status.Reason))
			}
		}
		deployment.Verification.Reason = strings.Join(reasons, "; ")
	} else {
		deployment.Verification.Status = models.VerificationStatusUnverified
	}
}

// GetVerificationStatus checks the verification status of a contract
func (v *Verifier) GetVerificationStatus(ctx context.Context, deployment *models.Deployment) (*models.VerificationInfo, error) {
	// This would check the explorer API for verification status
	// For now, return the stored status
	return &deployment.Verification, nil
}

// Ensure it implements the interface
var _ usecase.ContractVerifier = (*Verifier)(nil)

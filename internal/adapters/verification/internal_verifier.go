package verification

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// InternalVerifier is the new implementation without pkg dependencies
type InternalVerifier struct {
	projectRoot string
}

// NewInternalVerifier creates a new internal verifier
func NewInternalVerifier(cfg *config.RuntimeConfig) (*InternalVerifier, error) {
	return &InternalVerifier{
		projectRoot: cfg.ProjectRoot,
	}, nil
}

// Verify performs contract verification
func (v *InternalVerifier) Verify(ctx context.Context, deployment *domain.Deployment, network *domain.NetworkInfo) error {
	// Initialize verification status
	if deployment.Verification.Verifiers == nil {
		deployment.Verification.Verifiers = make(map[string]domain.VerifierStatus)
	}

	// Track verification errors
	var verificationErrors []string

	// Verify on Etherscan
	etherscanErr := v.verifyOnEtherscan(deployment, network)
	if etherscanErr != nil {
		deployment.Verification.Verifiers["etherscan"] = domain.VerifierStatus{
			Status: "failed",
			Reason: etherscanErr.Error(),
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("etherscan: %v", etherscanErr))
	} else {
		deployment.Verification.Verifiers["etherscan"] = domain.VerifierStatus{
			Status: "verified",
			URL:    v.buildEtherscanURL(network, deployment.Address),
		}
	}

	// Verify on Sourcify
	sourcifyErr := v.verifyOnSourceify(deployment, network)
	if sourcifyErr != nil {
		deployment.Verification.Verifiers["sourcify"] = domain.VerifierStatus{
			Status: "failed",
			Reason: sourcifyErr.Error(),
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("sourcify: %v", sourcifyErr))
	} else {
		deployment.Verification.Verifiers["sourcify"] = domain.VerifierStatus{
			Status: "verified",
			URL:    v.buildSourceifyURL(network, deployment.Address),
		}
	}

	// Update overall status
	v.updateOverallStatus(deployment)

	// Return error if all verifications failed
	if deployment.Verification.Status == domain.VerificationStatusFailed {
		return fmt.Errorf("all verifications failed: %s", strings.Join(verificationErrors, "; "))
	}

	return nil
}

// verifyOnEtherscan performs verification on Etherscan
func (v *InternalVerifier) verifyOnEtherscan(deployment *domain.Deployment, network *domain.NetworkInfo) error {
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
	return v.executeForgeVerify(args)
}

// verifyOnSourceify performs verification on Sourcify
func (v *InternalVerifier) verifyOnSourceify(deployment *domain.Deployment, network *domain.NetworkInfo) error {
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
	return v.executeForgeVerify(args)
}

// executeForgeVerify executes a forge verify-contract command
func (v *InternalVerifier) executeForgeVerify(args []string) error {
	cmd := exec.Command("forge", args...)
	cmd.Dir = v.projectRoot

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
func (v *InternalVerifier) buildEtherscanURL(network *domain.NetworkInfo, address string) string {
	if network.ExplorerURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/address/%s#code", network.ExplorerURL, address)
}

// buildSourceifyURL builds the Sourcify URL for a contract
func (v *InternalVerifier) buildSourceifyURL(network *domain.NetworkInfo, address string) string {
	return fmt.Sprintf("https://sourcify.dev/#/lookup/%s", address)
}

// updateOverallStatus updates the overall verification status based on individual verifiers
func (v *InternalVerifier) updateOverallStatus(deployment *domain.Deployment) {
	if deployment.Verification.Verifiers == nil {
		deployment.Verification.Status = domain.VerificationStatusUnverified
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
		deployment.Verification.Status = domain.VerificationStatusVerified
		// Set the explorer URL to the first verified one
		for _, status := range deployment.Verification.Verifiers {
			if status.Status == "verified" && status.URL != "" {
				deployment.Verification.EtherscanURL = status.URL
				break
			}
		}
	} else if verifiedCount > 0 {
		deployment.Verification.Status = domain.VerificationStatusPartial
	} else if failedCount == totalCount {
		deployment.Verification.Status = domain.VerificationStatusFailed
		// Combine all failure reasons
		var reasons []string
		for verifier, status := range deployment.Verification.Verifiers {
			if status.Status == "failed" && status.Reason != "" {
				reasons = append(reasons, fmt.Sprintf("%s: %s", verifier, status.Reason))
			}
		}
		deployment.Verification.Reason = strings.Join(reasons, "; ")
	} else {
		deployment.Verification.Status = domain.VerificationStatusUnverified
	}
}

// GetVerificationStatus checks the verification status of a contract
func (v *InternalVerifier) GetVerificationStatus(ctx context.Context, deployment *domain.Deployment) (*domain.VerificationInfo, error) {
	// This would check the explorer API for verification status
	// For now, return the stored status
	return &deployment.Verification, nil
}

// Ensure it implements the interface
var _ usecase.ContractVerifier = (*InternalVerifier)(nil)
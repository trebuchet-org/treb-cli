package verification

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

type Manager struct {
	registryManager *registry.Manager
	networkResolver *network.Resolver
}

func NewManager(registryManager *registry.Manager, networkResolver *network.Resolver) *Manager {
	return &Manager{
		registryManager: registryManager,
		networkResolver: networkResolver,
	}
}

// VerifyDeployment verifies a deployment on both Etherscan and Sourcify
func (vm *Manager) VerifyDeployment(deployment *types.Deployment) error {
	return vm.VerifyDeploymentWithDebug(deployment, false)
}

// VerifyDeploymentWithDebug verifies a deployment with optional debug output
func (vm *Manager) VerifyDeploymentWithDebug(deployment *types.Deployment, debug bool) error {
	// Get network name from chain ID for resolver
	networkName := getNetworkName(deployment.ChainID)

	// Get network info
	networkInfo, err := vm.networkResolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Initialize verification status
	if deployment.Verification.Verifiers == nil {
		deployment.Verification.Verifiers = make(map[string]types.VerifierStatus)
	}

	// Track verification errors
	var verificationErrors []string

	// Verify on Etherscan
	etherscanErr := vm.verifyOnEtherscanWithDebug(deployment, networkInfo, debug)
	if etherscanErr != nil {
		deployment.Verification.Verifiers["etherscan"] = types.VerifierStatus{
			Status: "failed",
			Reason: etherscanErr.Error(),
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("etherscan: %v", etherscanErr))
	} else {
		deployment.Verification.Verifiers["etherscan"] = types.VerifierStatus{
			Status: "verified",
			URL:    vm.buildEtherscanURL(networkInfo, deployment.Address),
		}
	}

	// Verify on Sourcify
	sourcifyErr := vm.verifyOnSourceifyWithDebug(deployment, networkInfo, debug)
	if sourcifyErr != nil {
		deployment.Verification.Verifiers["sourcify"] = types.VerifierStatus{
			Status: "failed",
			Reason: sourcifyErr.Error(),
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("sourcify: %v", sourcifyErr))
	} else {
		deployment.Verification.Verifiers["sourcify"] = types.VerifierStatus{
			Status: "verified",
			URL:    vm.buildSourceifyURL(networkInfo, deployment.Address),
		}
	}

	// Update overall status
	vm.updateOverallStatus(deployment)

	// Check verification status before saving to registry
	verificationFailed := deployment.Verification.Status == types.VerificationStatusFailed

	// Save to registry
	registryErr := vm.registryManager.SaveDeployment(deployment)
	if registryErr != nil {
		return fmt.Errorf("failed to update registry: %w", registryErr)
	}

	// Return error based on verification status
	if verificationFailed {
		return fmt.Errorf("all verifications failed: %s", strings.Join(verificationErrors, "; "))
	}

	// Return nil if at least one verification succeeded (registry was updated successfully)
	return nil
}

func (vm *Manager) verifyOnEtherscanWithDebug(deployment *types.Deployment, networkInfo *network.NetworkInfo, debug bool) error {
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
		"--chain-id", fmt.Sprintf("%d", networkInfo.ChainID),
		"--watch",
	}

	// Add compiler version if available
	compilerVersion := deployment.Artifact.CompilerVersion
	if compilerVersion != "" {
		args = append(args, "--compiler-version", compilerVersion)
	}

	// Add constructor args if available
	if constructorArgs != "" {
		args = append(args, "--constructor-args", constructorArgs)
	}

	if debug {
		fmt.Printf("\nEtherscan verification command:\n")
		fmt.Printf("forge %s\n\n", strings.Join(args, " "))
	}

	// Execute the command
	cmd := exec.Command("forge", args...)
	cmd.Dir = "." // Run from project root

	output, err := cmd.CombinedOutput()
	if debug && len(output) > 0 {
		fmt.Printf("Etherscan output:\n%s\n", string(output))
	}

	if err != nil {
		// Parse the output for specific error messages
		outputStr := string(output)
		if strings.Contains(outputStr, "Already Verified") || strings.Contains(outputStr, "is already verified") {
			// Contract is already verified, not an error
			return nil
		}
		return fmt.Errorf("etherscan verification failed: %s", strings.TrimSpace(outputStr))
	}

	// Check if verification was successful
	outputStr := string(output)
	if strings.Contains(outputStr, "Contract successfully verified") ||
		strings.Contains(outputStr, "Already Verified") ||
		strings.Contains(outputStr, "is already verified") {
		return nil
	}

	return fmt.Errorf("etherscan verification status unclear: %s", strings.TrimSpace(outputStr))
}

func (vm *Manager) verifyOnSourceifyWithDebug(deployment *types.Deployment, networkInfo *network.NetworkInfo, debug bool) error {
	// Get contract path from artifact
	contractPath := fmt.Sprintf("%s:%s", deployment.Artifact.Path, deployment.ContractName)

	// Build the forge verify-contract command for Sourcify
	args := []string{
		"verify-contract",
		deployment.Address,
		contractPath,
		"--chain-id", fmt.Sprintf("%d", networkInfo.ChainID),
		"--verifier", "sourcify",
		"--watch",
	}

	// Add compiler version if available
	compilerVersion := deployment.Artifact.CompilerVersion
	if compilerVersion != "" {
		args = append(args, "--compiler-version", compilerVersion)
	}

	// Add constructor args if available
	constructorArgs := deployment.DeploymentStrategy.ConstructorArgs
	if constructorArgs != "" && strings.HasPrefix(constructorArgs, "0x") {
		constructorArgs = constructorArgs[2:] // Remove 0x prefix
	}
	if constructorArgs != "" {
		args = append(args, "--constructor-args", constructorArgs)
	}

	if debug {
		fmt.Printf("\nSourceify verification command:\n")
		fmt.Printf("forge %s\n\n", strings.Join(args, " "))
	}

	// Execute the command
	cmd := exec.Command("forge", args...)
	cmd.Dir = "." // Run from project root

	output, err := cmd.CombinedOutput()
	if debug && len(output) > 0 {
		fmt.Printf("Sourcify output:\n%s\n", string(output))
	}

	if err != nil {
		// Parse the output for specific error messages
		outputStr := string(output)
		if strings.Contains(outputStr, "already verified") {
			// Contract is already verified, not an error
			return nil
		}
		return fmt.Errorf("sourcify verification failed: %s", strings.TrimSpace(outputStr))
	}

	// Check if verification was successful
	outputStr := string(output)
	if strings.Contains(outputStr, "Contract successfully verified") || strings.Contains(outputStr, "already verified") {
		return nil
	}

	return fmt.Errorf("sourcify verification status unclear: %s", strings.TrimSpace(outputStr))
}

// buildEtherscanURL builds the Etherscan URL for a contract
func (vm *Manager) buildEtherscanURL(networkInfo *network.NetworkInfo, address string) string {
	var baseURL string

	// Map common chain IDs to their explorer URLs
	switch networkInfo.ChainID {
	case 1:
		baseURL = "https://etherscan.io"
	case 5:
		baseURL = "https://goerli.etherscan.io"
	case 11155111:
		baseURL = "https://sepolia.etherscan.io"
	case 10:
		baseURL = "https://optimistic.etherscan.io"
	case 137:
		baseURL = "https://polygonscan.com"
	case 8453:
		baseURL = "https://basescan.org"
	case 42161:
		baseURL = "https://arbiscan.io"
	case 43114:
		baseURL = "https://snowtrace.io"
	case 56:
		baseURL = "https://bscscan.com"
	case 250:
		baseURL = "https://ftmscan.com"
	case 1101:
		baseURL = "https://zkevm.polygonscan.com"
	case 324:
		baseURL = "https://explorer.zksync.io"
	case 42220:
		baseURL = "https://celoscan.io"
	case 44787:
		baseURL = "https://alfajores.celoscan.io"
	default:
		// For unknown chains, return empty
		return ""
	}

	return fmt.Sprintf("%s/address/%s#code", baseURL, address)
}

// buildSourceifyURL builds the Sourcify URL for a contract
func (vm *Manager) buildSourceifyURL(networkInfo *network.NetworkInfo, address string) string {
	return fmt.Sprintf("https://sourcify.dev/#/lookup/%s", address)
}

// updateOverallStatus updates the overall verification status based on individual verifiers
func (vm *Manager) updateOverallStatus(deployment *types.Deployment) {
	if deployment.Verification.Verifiers == nil {
		deployment.Verification.Status = types.VerificationStatusUnverified
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
		deployment.Verification.Status = types.VerificationStatusVerified
		// Set the explorer URL to the first verified one
		for _, status := range deployment.Verification.Verifiers {
			if status.Status == "verified" && status.URL != "" {
				deployment.Verification.EtherscanURL = status.URL
				break
			}
		}
	} else if verifiedCount > 0 {
		deployment.Verification.Status = types.VerificationStatusPartial
	} else if failedCount == totalCount {
		deployment.Verification.Status = types.VerificationStatusFailed
		// Combine all failure reasons
		var reasons []string
		for verifier, status := range deployment.Verification.Verifiers {
			if status.Status == "failed" && status.Reason != "" {
				reasons = append(reasons, fmt.Sprintf("%s: %s", verifier, status.Reason))
			}
		}
		deployment.Verification.Reason = strings.Join(reasons, "; ")
	} else {
		deployment.Verification.Status = types.VerificationStatusUnverified
	}
}

// getNetworkName returns a network name for a given chain ID
func getNetworkName(chainID uint64) string {
	// Map common chain IDs to network names
	switch chainID {
	case 1:
		return "mainnet"
	case 5:
		return "goerli"
	case 11155111:
		return "sepolia"
	case 10:
		return "optimism"
	case 137:
		return "polygon"
	case 8453:
		return "base"
	case 42161:
		return "arbitrum"
	case 43114:
		return "avalanche"
	case 56:
		return "bsc"
	case 250:
		return "fantom"
	case 1101:
		return "polygon-zkevm"
	case 324:
		return "zksync"
	case 42220:
		return "celo"
	case 44787:
		return "celo-alfajores"
	case 31337:
		return "local"
	default:
		return fmt.Sprintf("chain-%d", chainID)
	}
}

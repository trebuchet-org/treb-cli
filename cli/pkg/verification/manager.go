package verification

import (
	"fmt"
	"os/exec"
	"strconv"
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

// VerifyContract verifies a contract on both Etherscan and Sourcify
func (vm *Manager) VerifyContract(deployment *registry.DeploymentInfo) error {
	return vm.VerifyContractWithDebug(deployment, false)
}

// VerifyContractWithDebug verifies a contract with optional debug output
func (vm *Manager) VerifyContractWithDebug(deployment *registry.DeploymentInfo, debug bool) error {

	// Get network info
	networkInfo, err := vm.networkResolver.ResolveNetwork(deployment.NetworkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Initialize verification status
	if deployment.Entry.Verification.Verifiers == nil {
		deployment.Entry.Verification.Verifiers = make(map[string]types.VerifierStatus)
	}

	// Track verification errors
	var verificationErrors []string

	// Verify on Etherscan
	etherscanErr := vm.verifyOnEtherscanWithDebug(deployment, networkInfo, debug)
	if etherscanErr != nil {
		deployment.Entry.Verification.Verifiers["etherscan"] = types.VerifierStatus{
			Status: "failed",
			Reason: etherscanErr.Error(),
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("etherscan: %v", etherscanErr))
	} else {
		deployment.Entry.Verification.Verifiers["etherscan"] = types.VerifierStatus{
			Status: "verified",
			URL:    vm.buildEtherscanURL(networkInfo, deployment.Address.Hex()),
		}
	}

	// Verify on Sourcify
	sourcifyErr := vm.verifyOnSourceifyWithDebug(deployment, networkInfo, debug)
	if sourcifyErr != nil {
		deployment.Entry.Verification.Verifiers["sourcify"] = types.VerifierStatus{
			Status: "failed",
			Reason: sourcifyErr.Error(),
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("sourcify: %v", sourcifyErr))
	} else {
		deployment.Entry.Verification.Verifiers["sourcify"] = types.VerifierStatus{
			Status: "verified",
			URL:    vm.buildSourceifyURL(networkInfo, deployment.Address.Hex()),
		}
	}

	// Update overall status
	vm.updateOverallStatus(deployment)

	// Check verification status before saving to registry
	verificationFailed := deployment.Entry.Verification.Status == "failed"

	// Save to registry
	chainIDUint, _ := strconv.ParseUint(deployment.ChainID, 10, 64)
	registryErr := vm.registryManager.UpdateDeployment(chainIDUint, deployment.Entry)
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

func (vm *Manager) verifyOnEtherscanWithDebug(deployment *registry.DeploymentInfo, networkInfo *network.NetworkInfo, debug bool) error {
	cmd := []string{
		"forge", "verify-contract",
		deployment.Address.Hex(),
		fmt.Sprintf("%s:%s", deployment.Entry.Metadata.ContractPath, deployment.Entry.ContractName),
		"--chain", networkInfo.Name,
	}

	// Add constructor args if available
	if deployment.Entry.ConstructorArgs != "" && deployment.Entry.ConstructorArgs != "0x" {
		cmd = append(cmd, "--constructor-args", deployment.Entry.ConstructorArgs)
	}

	// Add compiler version if available
	if deployment.Entry.Metadata.Compiler != "" && deployment.Entry.Metadata.Compiler != "unknown" {
		cmd = append(cmd, "--compiler-version", deployment.Entry.Metadata.Compiler)
	}

	// Print debug information if requested
	if debug {
		fmt.Printf("🔍 Etherscan verification command:\n")
		fmt.Printf("   %s\n", strings.Join(cmd, " "))
	}

	// Execute command
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		if debug {
			fmt.Printf("🔍 Etherscan verification output:\n%s\n", string(output))
		}
		return fmt.Errorf("etherscan verification failed: %s", string(output))
	}

	if debug {
		fmt.Printf("🔍 Etherscan verification successful\n")
	}

	return nil
}

func (vm *Manager) verifyOnSourceifyWithDebug(deployment *registry.DeploymentInfo, networkInfo *network.NetworkInfo, debug bool) error {
	// Build forge verify-contract command for Sourcify
	cmd := []string{
		"forge", "verify-contract",
		deployment.Address.Hex(),
		fmt.Sprintf("%s:%s", deployment.Entry.Metadata.ContractPath, deployment.Entry.ContractName),
		"--chain", networkInfo.Name,
		"--verifier", "sourcify",
	}

	// Add constructor args if available
	if deployment.Entry.ConstructorArgs != "" && deployment.Entry.ConstructorArgs != "0x" {
		cmd = append(cmd, "--constructor-args", deployment.Entry.ConstructorArgs)
	}

	// Print debug information if requested
	if debug {
		fmt.Printf("🔍 Sourcify verification command:\n")
		fmt.Printf("   %s\n", strings.Join(cmd, " "))
	}

	// Execute command
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		if debug {
			fmt.Printf("🔍 Sourcify verification output:\n%s\n", string(output))
		}
		return fmt.Errorf("sourcify verification failed: %s", string(output))
	}

	if debug {
		fmt.Printf("🔍 Sourcify verification successful\n")
	}

	return nil
}

// buildEtherscanURL builds the Etherscan URL for a contract
func (vm *Manager) buildEtherscanURL(networkInfo *network.NetworkInfo, address string) string {
	switch networkInfo.Name {
	case "mainnet":
		return fmt.Sprintf("https://etherscan.io/address/%s#code", address)
	case "sepolia":
		return fmt.Sprintf("https://sepolia.etherscan.io/address/%s#code", address)
	case "polygon":
		return fmt.Sprintf("https://polygonscan.com/address/%s#code", address)
	case "arbitrum":
		return fmt.Sprintf("https://arbiscan.io/address/%s#code", address)
	case "optimism":
		return fmt.Sprintf("https://optimistic.etherscan.io/address/%s#code", address)
	case "base":
		return fmt.Sprintf("https://basescan.org/address/%s#code", address)
	case "celo-alfajores":
		return fmt.Sprintf("https://alfajores.celoscan.io/address/%s#code", address)
	case "celo":
		return fmt.Sprintf("https://celoscan.io/address/%s#code", address)
	default:
		return ""
	}
}

// buildSourceifyURL builds the Sourcify URL for a contract
func (vm *Manager) buildSourceifyURL(networkInfo *network.NetworkInfo, address string) string {
	return fmt.Sprintf("https://sourcify.dev/#/lookup/%s", address)
}

// updateOverallStatus updates the overall verification status based on individual verifiers
func (vm *Manager) updateOverallStatus(deployment *registry.DeploymentInfo) {
	if deployment.Entry.Verification.Verifiers == nil {
		deployment.Entry.Verification.Status = "pending"
		return
	}

	verifiedCount := 0
	failedCount := 0
	totalCount := len(deployment.Entry.Verification.Verifiers)

	for _, status := range deployment.Entry.Verification.Verifiers {
		switch status.Status {
		case "verified":
			verifiedCount++
		case "failed":
			failedCount++
		}
	}

	if verifiedCount == totalCount {
		deployment.Entry.Verification.Status = "verified"
		// Set the explorer URL to the first verified one
		for _, status := range deployment.Entry.Verification.Verifiers {
			if status.Status == "verified" && status.URL != "" {
				deployment.Entry.Verification.ExplorerUrl = status.URL
				break
			}
		}
	} else if verifiedCount > 0 {
		deployment.Entry.Verification.Status = "partial"
	} else if failedCount == totalCount {
		deployment.Entry.Verification.Status = "failed"
		// Combine all failure reasons
		var reasons []string
		for verifier, status := range deployment.Entry.Verification.Verifiers {
			if status.Status == "failed" && status.Reason != "" {
				reasons = append(reasons, fmt.Sprintf("%s: %s", verifier, status.Reason))
			}
		}
		deployment.Entry.Verification.Reason = strings.Join(reasons, "; ")
	} else {
		deployment.Entry.Verification.Status = "pending"
	}
}

// VerifyPendingContracts verifies all pending contracts for a specific chain
func (vm *Manager) VerifyPendingContracts(chainID uint64) error {
	allDeployments := vm.registryManager.GetAllDeployments()

	for _, deployment := range allDeployments {
		// Only process contracts on the specified chain that are pending verification
		deploymentChainID, _ := strconv.ParseUint(deployment.ChainID, 10, 64)
		if deploymentChainID != chainID {
			continue
		}

		if deployment.Entry.Verification.Status != "pending" {
			continue
		}

		// Check if Safe tx is executed for Safe deployments
		if deployment.Entry.Deployment.SafeTxHash != nil {
			// TODO: Implement Safe transaction status check
			continue
		}

		err := vm.VerifyContract(deployment)
		if err != nil {
			fmt.Printf("Failed to verify %s: %v\n", deployment.Entry.GetDisplayName(), err)
		}
	}

	return nil
}

// CheckVerificationStatus checks if contracts are verified using forge verify-check
func (vm *Manager) CheckVerificationStatus(deployment *registry.DeploymentInfo) error {
	networkInfo, err := vm.networkResolver.ResolveNetwork(deployment.NetworkName)
	if err != nil {
		return err
	}

	// Check Etherscan status
	cmd := exec.Command("forge", "verify-check",
		"--chain", networkInfo.Name,
		deployment.Address.Hex())

	if cmd.Run() == nil {
		if deployment.Entry.Verification.Verifiers == nil {
			deployment.Entry.Verification.Verifiers = make(map[string]types.VerifierStatus)
		}
		deployment.Entry.Verification.Verifiers["etherscan"] = types.VerifierStatus{
			Status: "verified",
			URL:    vm.buildEtherscanURL(networkInfo, deployment.Address.Hex()),
		}
	}

	// Check Sourcify status
	cmd = exec.Command("forge", "verify-check",
		"--chain", networkInfo.Name,
		"--verifier", "sourcify",
		deployment.Address.Hex())

	if cmd.Run() == nil {
		if deployment.Entry.Verification.Verifiers == nil {
			deployment.Entry.Verification.Verifiers = make(map[string]types.VerifierStatus)
		}
		deployment.Entry.Verification.Verifiers["sourcify"] = types.VerifierStatus{
			Status: "verified",
			URL:    vm.buildSourceifyURL(networkInfo, deployment.Address.Hex()),
		}
	}

	vm.updateOverallStatus(deployment)
	return nil
}

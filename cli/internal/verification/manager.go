package verification

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/ethereum/go-ethereum/common"
)

type RegistryManager interface {
	GetPendingVerifications(chainID uint64) map[string]*types.DeploymentEntry
	UpdateDeployment(key string, deployment *types.DeploymentEntry) error
}

type Manager struct {
	registry RegistryManager
}

func NewManager(executor interface{}, registry RegistryManager) *Manager {
	return &Manager{
		registry: registry,
	}
}

func (vm *Manager) VerifyPendingContracts(chainID uint64) error {
	deployments := vm.registry.GetPendingVerifications(chainID)

	for key, deployment := range deployments {
		if deployment.Deployment.SafeTxHash != nil {
			// Check if Safe tx is executed
			executed, err := vm.checkSafeTransactionStatus(*deployment.Deployment.SafeTxHash)
			if err != nil || !executed {
				continue
			}
		}

		// Verify the contract
		err := vm.verifyContract(deployment)
		if err != nil {
			deployment.Verification.Status = "failed"
			deployment.Verification.Reason = err.Error()
		} else {
			deployment.Verification.Status = "verified"
			deployment.Verification.ExplorerUrl = vm.buildExplorerUrl(chainID, deployment.Address)
		}

		vm.registry.UpdateDeployment(key, deployment)
	}

	return nil
}

func (vm *Manager) verifyContract(deployment *types.DeploymentEntry) error {
	// TODO: Implement contract verification
	// This could use forge verify-contract or call etherscan API directly
	return fmt.Errorf("contract verification not implemented")
}

func (vm *Manager) checkSafeTransactionStatus(safeTxHash common.Hash) (bool, error) {
	// TODO: Check Safe transaction status via Safe API
	return false, fmt.Errorf("safe transaction status check not implemented")
}

func (vm *Manager) buildExplorerUrl(chainID uint64, address common.Address) string {
	switch chainID {
	case 1:
		return fmt.Sprintf("https://etherscan.io/address/%s#code", address.Hex())
	case 11155111:
		return fmt.Sprintf("https://sepolia.etherscan.io/address/%s#code", address.Hex())
	case 137:
		return fmt.Sprintf("https://polygonscan.com/address/%s#code", address.Hex())
	case 42161:
		return fmt.Sprintf("https://arbiscan.io/address/%s#code", address.Hex())
	default:
		return ""
	}
}
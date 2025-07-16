package abi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// DeploymentLookup is the interface we need from registry.Manager
type DeploymentLookup interface {
	GetDeploymentByAddress(chainID uint64, address string) (*types.Deployment, error)
}

// ContractInfo represents the minimal contract information we need
type ContractInfo interface {
	GetArtifactPath() string
}

// ContractLookup is the interface we need from contracts.Indexer
type ContractLookup interface {
	GetContractByArtifact(artifact string) ContractInfo
}

// RegistryABIResolver implements ABIResolver using the deployment registry and contracts indexer
type RegistryABIResolver struct {
	deploymentLookup DeploymentLookup
	contractLookup   ContractLookup
	chainID          uint64
	debug            bool
}

// NewRegistryABIResolver creates a new registry-based ABI resolver
func NewRegistryABIResolver(deploymentLookup DeploymentLookup, contractLookup ContractLookup, chainID uint64) ABIResolver {
	return &RegistryABIResolver{
		deploymentLookup: deploymentLookup,
		contractLookup:   contractLookup,
		chainID:          chainID,
		debug:            false,
	}
}

// EnableDebug enables debug logging
func (r *RegistryABIResolver) EnableDebug(debug bool) {
	r.debug = debug
}

// ResolveByAddress attempts to find and load the ABI for a given address
func (r *RegistryABIResolver) ResolveByAddress(address common.Address) (contractName string, abiJSON string, isProxy bool, implAddress *common.Address) {
	if r.deploymentLookup == nil || r.contractLookup == nil {
		return "", "", false, nil
	}

	if r.debug {
		fmt.Printf("[ABIResolver] Looking up deployment for address %s on chain %d\n", address.Hex(), r.chainID)
	}

	// Look up the deployment in the registry (normalize address to lowercase)
	normalizedAddr := strings.ToLower(address.Hex())
	deployment, err := r.deploymentLookup.GetDeploymentByAddress(r.chainID, normalizedAddr)
	if err != nil || deployment == nil {
		// Try with checksummed address
		deployment, err = r.deploymentLookup.GetDeploymentByAddress(r.chainID, address.Hex())
		if err != nil || deployment == nil {
			if r.debug {
				fmt.Printf("[ABIResolver] No deployment found for address %s (tried %s and %s): %v\n",
					address.Hex(), normalizedAddr, address.Hex(), err)
			}
			return "", "", false, nil
		}
	}

	if r.debug {
		displayName := deployment.ContractDisplayName()
		fmt.Printf("[ABIResolver] Found deployment: %s (type=%s, artifact=%s)\n",
			displayName, deployment.Type, deployment.Artifact.Path)
		if deployment.ProxyInfo != nil {
			fmt.Printf("[ABIResolver] Proxy info: impl=%s\n", deployment.ProxyInfo.Implementation)
		}
	}

	// Check if it's a proxy - if so, we need to get the implementation's ABI
	if deployment.Type == types.ProxyDeployment && deployment.ProxyInfo != nil && deployment.ProxyInfo.Implementation != "" {
		if r.debug {
			fmt.Printf("[ABIResolver] Proxy detected, looking up implementation at %s\n", deployment.ProxyInfo.Implementation)
		}

		// Normalize implementation address too
		normalizedImpl := strings.ToLower(deployment.ProxyInfo.Implementation)
		implDeployment, err := r.deploymentLookup.GetDeploymentByAddress(r.chainID, normalizedImpl)
		if err != nil || implDeployment == nil {
			// Try with original case
			implDeployment, err = r.deploymentLookup.GetDeploymentByAddress(r.chainID, deployment.ProxyInfo.Implementation)
			if err != nil || implDeployment == nil {
				if r.debug {
					fmt.Printf("[ABIResolver] Implementation not found in registry, falling back to proxy ABI\n")
				}
				// Fall back to proxy's own ABI if we can't find the implementation
				contractName, abiJSON, _, _ := r.loadContractABI(deployment)
				implAddr := common.HexToAddress(deployment.ProxyInfo.Implementation)
				// Still mark it as a proxy even if we only have proxy ABI
				return contractName, abiJSON, true, &implAddr
			}
		}

		if r.debug {
			implDisplayName := implDeployment.ContractDisplayName()
			fmt.Printf("[ABIResolver] Found implementation deployment: %s\n", implDisplayName)
		}

		// Load the implementation's ABI
		_, abiJSON, _, _ := r.loadContractABI(implDeployment)
		if abiJSON == "" {
			if r.debug {
				fmt.Printf("[ABIResolver] Failed to load implementation ABI, falling back to proxy ABI\n")
			}
			// Fall back to proxy's own ABI if we can't load implementation ABI
			contractName, abiJSON, _, _ := r.loadContractABI(deployment)
			implAddr := common.HexToAddress(deployment.ProxyInfo.Implementation)
			return contractName, abiJSON, true, &implAddr
		}

		// Return implementation ABI with proxy info
		implAddr := common.HexToAddress(deployment.ProxyInfo.Implementation)
		// Use the proxy's display name
		displayName := deployment.ContractName
		if deployment.Label != "" {
			displayName = fmt.Sprintf("%s:%s", deployment.ContractName, deployment.Label)
		}
		return displayName, abiJSON, true, &implAddr
	}

	// Not a proxy, load the contract's own ABI
	return r.loadContractABI(deployment)
}

// loadContractABI loads the ABI for a deployment
func (r *RegistryABIResolver) loadContractABI(deployment *types.Deployment) (contractName string, abiJSON string, isProxy bool, implAddress *common.Address) {
	if r.debug {
		fmt.Printf("[ABIResolver] Loading ABI for %s from artifact %s\n", deployment.ContractName, deployment.Artifact.Path)
	}

	// Get contract info from the indexer using the artifact path
	contractInfo := r.contractLookup.GetContractByArtifact(deployment.Artifact.Path)
	if contractInfo == nil {
		if r.debug {
			fmt.Printf("[ABIResolver] Contract info not found in indexer for %s\n", deployment.Artifact.Path)
		}
		return "", "", false, nil
	}

	artifactPath := contractInfo.GetArtifactPath()
	if artifactPath == "" {
		if r.debug {
			fmt.Printf("[ABIResolver] No artifact path in contract info\n")
		}
		return "", "", false, nil
	}

	if r.debug {
		fmt.Printf("[ABIResolver] Loading ABI from artifact file: %s\n", artifactPath)
	}

	// Load ABI from the artifact file
	abiJSON = r.loadABIFromArtifact(artifactPath)
	if abiJSON == "" {
		if r.debug {
			fmt.Printf("[ABIResolver] Failed to load ABI from artifact file\n")
		}
		return "", "", false, nil
	}

	// Use the deployment's display name
	contractName = deployment.ContractName
	if deployment.Label != "" {
		contractName = fmt.Sprintf("%s:%s", deployment.ContractName, deployment.Label)
	}

	if r.debug {
		fmt.Printf("[ABIResolver] Successfully loaded ABI for %s\n", contractName)
	}

	return contractName, abiJSON, false, nil
}

// loadABIFromArtifact loads ABI JSON from an artifact file
func (r *RegistryABIResolver) loadABIFromArtifact(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	// Parse the Foundry artifact JSON
	var artifact struct {
		ABI json.RawMessage `json:"abi"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return ""
	}

	return string(artifact.ABI)
}

// ResolveByArtifact attempts to find and load the ABI for a given artifact name
func (r *RegistryABIResolver) ResolveByArtifact(artifact string) (contractName string, abiJSON string) {
	if r.contractLookup == nil {
		return "", ""
	}

	if r.debug {
		fmt.Printf("[ABIResolver] Looking up ABI for artifact %s\n", artifact)
	}

	// Get contract info from the indexer using the artifact path
	contractInfo := r.contractLookup.GetContractByArtifact(artifact)
	if contractInfo == nil {
		if r.debug {
			fmt.Printf("[ABIResolver] Contract info not found in indexer for %s\n", artifact)
		}
		return "", ""
	}

	artifactPath := contractInfo.GetArtifactPath()
	if artifactPath == "" {
		if r.debug {
			fmt.Printf("[ABIResolver] No artifact path in contract info\n")
		}
		return "", ""
	}

	if r.debug {
		fmt.Printf("[ABIResolver] Loading ABI from artifact file: %s\n", artifactPath)
	}

	// Load ABI from the artifact file
	abiJSON = r.loadABIFromArtifact(artifactPath)
	if abiJSON == "" {
		if r.debug {
			fmt.Printf("[ABIResolver] Failed to load ABI from artifact file\n")
		}
		return "", ""
	}

	// Extract contract name from artifact path (e.g., "src/Counter.sol:Counter" -> "Counter")
	contractName = artifact
	if idx := strings.LastIndex(artifact, ":"); idx != -1 {
		contractName = artifact[idx+1:]
	}

	if r.debug {
		fmt.Printf("[ABIResolver] Successfully loaded ABI for %s\n", contractName)
	}

	return contractName, abiJSON
}

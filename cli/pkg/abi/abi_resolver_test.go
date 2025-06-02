package abi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

func TestRegistryABIResolver(t *testing.T) {
	// Create a temporary directory for test data
	tmpDir := t.TempDir()

	// Create a mock deployment in registry
	deployment := &types.Deployment{
		ID:           "test/1/Counter:v1",
		Namespace:    "test",
		ChainID:      1,
		ContractName: "Counter",
		Label:        "v1",
		Address:      "0x1234567890123456789012345678901234567890",
		Type:         types.SingletonDeployment,
		Artifact: types.ArtifactInfo{
			Path: "src/Counter.sol:Counter",
		},
	}

	// Create a mock artifact file
	artifactDir := filepath.Join(tmpDir, "out", "src", "Counter.sol")
	require.NoError(t, os.MkdirAll(artifactDir, 0755))

	mockABI := `[{"inputs":[],"name":"increment","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	artifact := map[string]interface{}{
		"abi": json.RawMessage(mockABI),
	}
	artifactData, err := json.Marshal(artifact)
	require.NoError(t, err)

	artifactPath := filepath.Join(artifactDir, "Counter.json")
	require.NoError(t, os.WriteFile(artifactPath, artifactData, 0644))

	// Create a mock indexer
	indexer := &mockIndexer{
		contracts: map[string]*mockContractInfo{
			"src/Counter.sol:Counter": {
				artifactPath: artifactPath,
			},
		},
	}

	// Create a mock registry manager
	manager := &mockRegistryManager{
		deployments: map[string]*types.Deployment{
			"0x1234567890123456789012345678901234567890": deployment,
		},
	}

	// Create resolver using the factory function with concrete types
	resolver := &RegistryABIResolver{
		deploymentLookup: manager,
		contractLookup:   indexer,
		chainID:          1,
	}

	// Test resolving a known contract
	t.Run("resolve known contract", func(t *testing.T) {
		address := common.HexToAddress("0x1234567890123456789012345678901234567890")
		contractName, abiJSON, isProxy, implAddr := resolver.ResolveABI(address)

		assert.Equal(t, "Counter", contractName)
		assert.Equal(t, mockABI, abiJSON)
		assert.False(t, isProxy)
		assert.Nil(t, implAddr)
	})

	// Test resolving unknown contract
	t.Run("resolve unknown contract", func(t *testing.T) {
		address := common.HexToAddress("0x0000000000000000000000000000000000000000")
		contractName, abiJSON, isProxy, implAddr := resolver.ResolveABI(address)

		assert.Empty(t, contractName)
		assert.Empty(t, abiJSON)
		assert.False(t, isProxy)
		assert.Nil(t, implAddr)
	})

	// Test resolving proxy contract
	t.Run("resolve proxy contract", func(t *testing.T) {
		// Create a proxy deployment
		proxyDeployment := &types.Deployment{
			ID:           "test/1/ProxyCounter:v1",
			Namespace:    "test",
			ChainID:      1,
			ContractName: "ProxyCounter",
			Label:        "v1",
			Address:      "0x2234567890123456789012345678901234567890",
			Type:         types.ProxyDeployment,
			Artifact: types.ArtifactInfo{
				Path: "src/ProxyCounter.sol:ProxyCounter",
			},
			ProxyInfo: &types.ProxyInfo{
				Type:           "ERC1967",
				Implementation: "0x1234567890123456789012345678901234567890",
			},
		}

		// Create minimal proxy artifact
		proxyArtifactDir := filepath.Join(tmpDir, "out", "src", "ProxyCounter.sol")
		require.NoError(t, os.MkdirAll(proxyArtifactDir, 0755))

		mockProxyABI := `[{"inputs":[],"name":"implementation","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
		proxyArtifact := map[string]interface{}{
			"abi": json.RawMessage(mockProxyABI),
		}
		proxyArtifactData, err := json.Marshal(proxyArtifact)
		require.NoError(t, err)

		proxyArtifactPath := filepath.Join(proxyArtifactDir, "ProxyCounter.json")
		require.NoError(t, os.WriteFile(proxyArtifactPath, proxyArtifactData, 0644))

		// Add proxy contract info to indexer
		indexer.contracts["src/ProxyCounter.sol:ProxyCounter"] = &mockContractInfo{
			artifactPath: proxyArtifactPath,
		}

		manager.deployments["0x2234567890123456789012345678901234567890"] = proxyDeployment

		address := common.HexToAddress("0x2234567890123456789012345678901234567890")
		contractName, abiJSON, isProxy, implAddr := resolver.ResolveABI(address)

		// Should return implementation ABI with proxy naming
		assert.Equal(t, "ProxyCounter[Counter]", contractName)
		assert.Equal(t, mockABI, abiJSON) // Should be implementation ABI, not proxy ABI
		assert.True(t, isProxy)
		assert.NotNil(t, implAddr)
		assert.Equal(t, common.HexToAddress("0x1234567890123456789012345678901234567890"), *implAddr)
	})
}

// Mock implementations for testing

type mockRegistryManager struct {
	deployments map[string]*types.Deployment
}

func (m *mockRegistryManager) GetDeploymentByAddress(chainID uint64, address string) (*types.Deployment, error) {
	if deployment, exists := m.deployments[address]; exists && deployment.ChainID == chainID {
		return deployment, nil
	}
	return nil, nil
}

type mockContractInfo struct {
	artifactPath string
}

func (m *mockContractInfo) GetArtifactPath() string {
	return m.artifactPath
}

type mockIndexer struct {
	contracts map[string]*mockContractInfo
}

func (m *mockIndexer) GetContractByArtifact(artifact string) ContractInfo {
	info := m.contracts[artifact]
	if info == nil {
		return nil
	}
	return info
}

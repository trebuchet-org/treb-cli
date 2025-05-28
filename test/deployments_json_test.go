package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test deployments.json integrity for different deployment types
func TestDeploymentsJSONIntegrity(t *testing.T) {
	
	t.Run("singleton deployment integrity", func(t *testing.T) {
		// Clean start
		cleanupGeneratedFiles(t)
		
		// Create treb context
		tc := NewTrebContext(t)
		
		// Generate and deploy a singleton with unique label
		_, err := tc.treb("gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3")
		require.NoError(t, err)
		
		_, err = tc.treb("deploy", "src/Counter.sol:Counter", "--label", "singleton-test")
		require.NoError(t, err)
		
		// Check deployments.json
		deploymentsFile := filepath.Join(fixtureDir, "deployments.json")
		data, err := os.ReadFile(deploymentsFile)
		require.NoError(t, err)
		
		var registry map[string]interface{}
		err = json.Unmarshal(data, &registry)
		require.NoError(t, err)
		
		// Verify structure - navigate through networks
		networks, ok := registry["networks"].(map[string]interface{})
		require.True(t, ok, "networks should be a map")
		
		// Get anvil network (chain ID 31337)
		anvilNetwork, ok := networks["31337"].(map[string]interface{})
		require.True(t, ok, "anvil network (31337) should exist")
		
		// Get deployments
		deployments, ok := anvilNetwork["deployments"].(map[string]interface{})
		require.True(t, ok, "deployments should be a map")
		
		// Find the Counter deployment
		var counterDeployment map[string]interface{}
		for _, deployment := range deployments {
			dep := deployment.(map[string]interface{})
			if dep["contract_name"] == "Counter" {
				counterDeployment = dep
				break
			}
		}
		require.NotNil(t, counterDeployment, "Counter deployment not found")
		
		// Verify deployment properties
		assert.Equal(t, "Counter", counterDeployment["contract_name"])
		assert.Equal(t, "SINGLETON", counterDeployment["type"])
		assert.NotEmpty(t, counterDeployment["address"])
		assert.NotEmpty(t, counterDeployment["salt"])
		assert.NotEmpty(t, counterDeployment["init_code_hash"])
		
		// Verify deployment info
		deployment := counterDeployment["deployment"].(map[string]interface{})
		assert.NotEmpty(t, deployment["tx_hash"])
		assert.NotEmpty(t, deployment["block_number"])
		assert.Equal(t, "EXECUTED", deployment["status"])
		assert.NotEmpty(t, deployment["deployer"])
		
		// Verify metadata
		metadata := counterDeployment["metadata"].(map[string]interface{})
		assert.Equal(t, "src/Counter.sol", metadata["contract_path"])
		assert.NotEmpty(t, metadata["compiler"])
		
		// Verify verification status
		verification := counterDeployment["verification"].(map[string]interface{})
		assert.Equal(t, "pending", verification["status"])
	})
	
	t.Run("proxy deployment integrity", func(t *testing.T) {
		// Clean start
		cleanupGeneratedFiles(t)
		
		// Create treb context
		tc := NewTrebContext(t)
		
		// Step 1: Generate and deploy the implementation first
		_, err := tc.treb("gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--strategy", "CREATE3")
		require.NoError(t, err)
		
		_, err = tc.treb("deploy", "src/UpgradeableCounter.sol:UpgradeableCounter")
		require.NoError(t, err)
		
		// Step 2: Generate and deploy the proxy pointing to the implementation (using ShortID)
		_, err = tc.treb("gen", "proxy",
			"src/UpgradeableCounter.sol:UpgradeableCounter",
			"--strategy", "CREATE3",
			"--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol:TransparentUpgradeableProxy")
		require.NoError(t, err)
		
		// Deploy proxy using the ShortID "UpgradeableCounter" which will resolve to the deployed implementation
		_, err = tc.treb("deploy", "proxy", "UpgradeableCounter")
		require.NoError(t, err)
		
		// Check deployments.json
		deploymentsFile := filepath.Join(fixtureDir, "deployments.json")
		data, err := os.ReadFile(deploymentsFile)
		require.NoError(t, err)
		
		var registry map[string]interface{}
		err = json.Unmarshal(data, &registry)
		require.NoError(t, err)
		
		// Navigate to deployments
		networks := registry["networks"].(map[string]interface{})
		anvilNetwork := networks["31337"].(map[string]interface{})
		deployments := anvilNetwork["deployments"].(map[string]interface{})
		
		// Find proxy and singleton deployments
		var proxyDeployment, singletonDeployment map[string]interface{}
		proxyCount := 0
		implCount := 0
		
		for _, deployment := range deployments {
			dep := deployment.(map[string]interface{})
			depType := dep["type"].(string)
			sid := dep["sid"].(string)
			
			if depType == "PROXY" && sid == "UpgradeableCounterProxy" {
				proxyDeployment = dep
				proxyCount++
			} else if depType == "SINGLETON" && sid == "UpgradeableCounter" {
				singletonDeployment = dep
				implCount++
			}
		}
		
		// Verify we found both
		require.NotNil(t, proxyDeployment, "Proxy deployment not found")
		require.NotNil(t, singletonDeployment, "Singleton deployment not found")
		assert.Equal(t, 1, proxyCount, "Should have exactly one proxy")
		assert.Equal(t, 1, implCount, "Should have exactly one singleton")
		
		// Verify proxy fields
		assert.Equal(t, "PROXY", proxyDeployment["type"])
		assert.NotEmpty(t, proxyDeployment["address"])
		
		// Verify singleton fields
		assert.Equal(t, "SINGLETON", singletonDeployment["type"])
		assert.NotEmpty(t, singletonDeployment["address"])
		
		// Note: The proxy deployment may not have implementation_address field in the current structure
		// But both should have proper deployment info
		proxyDeployInfo := proxyDeployment["deployment"].(map[string]interface{})
		assert.NotEmpty(t, proxyDeployInfo["tx_hash"])
		assert.Equal(t, "EXECUTED", proxyDeployInfo["status"])
		
		singletonDeployInfo := singletonDeployment["deployment"].(map[string]interface{})
		assert.NotEmpty(t, singletonDeployInfo["tx_hash"])
		assert.Equal(t, "EXECUTED", singletonDeployInfo["status"])
	})
	
	t.Run("library deployment integrity", func(t *testing.T) {
		t.Skip("Library deployment is not yet supported through the standard deploy flow")
		// TODO: Once library deployment is supported, update this test
	})
	
	t.Run("multiple deployments integrity", func(t *testing.T) {
		// Deploy multiple contracts and verify registry structure
		cleanupGeneratedFiles(t)
		
		// Create treb context
		tc := NewTrebContext(t)
		
		// Deploy Counter with unique label
		_, _ = tc.treb("gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3")
		_, _ = tc.treb("deploy", "src/Counter.sol:Counter", "--label", "multi-test")
		
		// Deploy TestCounter
		_, _ = tc.treb("gen", "deploy", "src/TestCounter.sol:TestCounter", "--strategy", "CREATE2")
		_, _ = tc.treb("deploy", "src/TestCounter.sol:TestCounter")
		
		// Check deployments.json
		deploymentsFile := filepath.Join(fixtureDir, "deployments.json")
		data, err := os.ReadFile(deploymentsFile)
		require.NoError(t, err)
		
		var registry map[string]interface{}
		err = json.Unmarshal(data, &registry)
		require.NoError(t, err)
		
		// Navigate to deployments
		networks, ok := registry["networks"].(map[string]interface{})
		require.True(t, ok, "networks should exist")
		
		anvilNetwork, ok := networks["31337"].(map[string]interface{})
		require.True(t, ok, "anvil network should exist")
		
		deployments := anvilNetwork["deployments"].(map[string]interface{})
		
		// Count unique contracts
		contractNames := make(map[string]bool)
		for _, deployment := range deployments {
			dep := deployment.(map[string]interface{})
			contractNames[dep["contract_name"].(string)] = true
		}
		
		assert.GreaterOrEqual(t, len(contractNames), 2, "Should have at least 2 different contracts deployed")
		assert.True(t, contractNames["Counter"], "Counter should be deployed")
		assert.True(t, contractNames["TestCounter"], "TestCounter should be deployed")
	})
}
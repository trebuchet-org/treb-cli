package integration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test full deployment flow
func TestDeploymentFlow(t *testing.T) {
	// Cleanup
	cleanupGeneratedFiles(t)
	
	// Generate
	output, err := runTreb(t, "gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3")
	require.NoError(t, err)
	assert.Contains(t, output, "Generated deploy script")
	
	// Deploy
	output, err = runTreb(t, "deploy", "src/Counter.sol:Counter", "--network", "anvil")
	require.NoError(t, err)
	assert.Contains(t, output, "Deployment Successful")
	
	// Extract address from output
	lines := strings.Split(output, "\n")
	var address string
	for _, line := range lines {
		if strings.Contains(line, "Address:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				address = parts[len(parts)-1]
				break
			}
		}
	}
	assert.NotEmpty(t, address, "Should have deployment address")
	assert.True(t, strings.HasPrefix(address, "0x"), "Address should start with 0x")
}

// Test show and list commands
func TestShowAndList(t *testing.T) {
	// This test depends on the deployment from TestDeploymentFlow
	// In a real test suite, you might want to ensure proper test ordering
	// or deploy a contract within this test
	
	t.Run("list deployments", func(t *testing.T) {
		output, err := runTreb(t, "list")
		assert.NoError(t, err)
		// Check for either empty deployments or Counter deployment
		assert.True(t, 
			strings.Contains(output, "No deployments found") || strings.Contains(output, "Counter"),
			"List should show deployments or empty message")
	})
	
	t.Run("show existing deployment", func(t *testing.T) {
		// First ensure we have a deployment
		cleanupGeneratedFiles(t)
		
		// Generate and deploy
		_, err := runTreb(t, "gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3")
		require.NoError(t, err)
		
		_, err = runTreb(t, "deploy", "src/Counter.sol:Counter", "--network", "anvil")
		require.NoError(t, err)
		
		// Now test show
		output, err := runTreb(t, "show", "Counter")
		assert.NoError(t, err)
		assert.Contains(t, output, "Counter")
		assert.Contains(t, output, "0x") // Should have address
	})
	
	t.Run("show with network filter", func(t *testing.T) {
		output, err := runTreb(t, "show", "Counter", "--network", "anvil")
		assert.NoError(t, err)
		assert.Contains(t, output, "Counter")
	})
	
	t.Run("show non-existent deployment", func(t *testing.T) {
		output, err := runTreb(t, "show", "NonExistentContract")
		assert.Error(t, err)
		assert.Contains(t, output, "no deployment found")
	})
}

// Test verify command behavior
func TestVerifyCommand(t *testing.T) {
	t.Run("verify non-existent contract", func(t *testing.T) {
		output, err := runTreb(t, "verify", "NonExistent")
		assert.Error(t, err)
		assert.Contains(t, output, "no deployment found")
	})
	
	t.Run("verify without deployments", func(t *testing.T) {
		// Clean up to ensure no deployments
		cleanupGeneratedFiles(t)
		
		output, err := runTreb(t, "verify", "Counter")
		assert.Error(t, err)
		assert.Contains(t, output, "no deployment found")
	})
	
	// Note: We skip testing actual verification on anvil as it would timeout
	// since anvil doesn't have a block explorer
}
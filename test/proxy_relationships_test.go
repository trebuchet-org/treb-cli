package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyDeploymentRelationships(t *testing.T) {
	// Skip this test in CI for now - needs infrastructure improvements
	if os.Getenv("CI") != "" {
		t.Skip("Skipping proxy relationships test in CI - needs infrastructure improvements")
	}

	t.Run("deploy_proxy_using_script", func(t *testing.T) {
		// Clean slate
		cleanupGeneratedFiles(t)

		// Deploy using the DeployUCProxy script which deploys an upgradeable proxy
		output, err := runScriptDebug(t, "script/DeployUCProxy.s.sol", "deployer=anvil", "label=proxy-test")
		require.NoError(t, err)

		// Should have deployed contracts
		assert.Contains(t, output, "Deployment Summary:")
		assert.Contains(t, output, "UpgradeableCounter")
		assert.Contains(t, output, "Proxy[UpgradeableCounter]")

		// Check that proxy relationships were detected
		assert.Contains(t, output, "Proxy Relationships Detected:")

		// Verify list shows the deployments
		listOutput, err := runTrebDebug(t, "list")
		require.NoError(t, err)

		// Should show deployments
		assert.Contains(t, listOutput, "31337")                    // Chain ID
		assert.Contains(t, strings.ToLower(listOutput), "default") // Namespace (might be uppercase)
	})

	t.Run("list_shows_deployments", func(t *testing.T) {
		// Deploy something first
		cleanupGeneratedFiles(t)

		_, err := runScriptDebug(t, "script/DeployWithTreb.s.sol", "deployer=anvil", "COUNTER_LABEL=list-show-test", "TOKEN_LABEL=list-show-test")
		require.NoError(t, err)

		// Run list command
		output, err := runTrebDebug(t, "list")
		require.NoError(t, err)

		outputStr := string(output)
		t.Logf("List output:\n%s", outputStr)

		// Verify we see deployments
		assert.Contains(t, outputStr, "31337")
		assert.Contains(t, strings.ToLower(outputStr), "default")

		// Test list with chain filter
		output, err = runTrebDebug(t, "list", "--chain", "31337")
		require.NoError(t, err)
		assert.Contains(t, string(output), "31337")

		// Test list with namespace filter
		output, err = runTrebDebug(t, "list", "--namespace", "default")
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(string(output)), "default")
	})
}

// Helper to remove old deployments file (no longer used with v2 registry)
func removeDeploymentsFile() {
	os.RemoveAll(filepath.Join(fixtureDir, ".treb"))
}

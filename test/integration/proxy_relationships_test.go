package integration

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
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
		helpers.IsolatedTest(t, "deploy_proxy_using_script", func(t *testing.T, ctx *helpers.TrebContext) {
			// Deploy using the DeployUCProxy script which deploys an upgradeable proxy
			output, err := ctx.Treb("run", "script/DeployUCProxy.s.sol", "--env", "deployer=anvil", "--env", "label=proxy-test")
			require.NoError(t, err)

			// Should have deployed contracts
			assert.Contains(t, output, "Deployment Summary:")
			assert.Contains(t, output, "UpgradeableCounter")
			assert.Contains(t, output, "ERC1967Proxy[UpgradeableCounter]")

			// The output should show the proxy deployment with its implementation
			// Look for either the proxy pattern in deployment summary or transaction details
			assert.Contains(t, output, "ERC1967Proxy")

			// Verify list shows the deployments
			listOutput, err := ctx.Treb("list")
			require.NoError(t, err)

			// Should show deployments
			assert.Contains(t, listOutput, "31337")                    // Chain ID
			assert.Contains(t, strings.ToLower(listOutput), "default") // Namespace (might be uppercase)
		})
	})

	t.Run("list_shows_deployments", func(t *testing.T) {
		helpers.IsolatedTest(t, "list_shows_deployments", func(t *testing.T, ctx *helpers.TrebContext) {
			_, err := ctx.Treb("run", "script/DeployWithTreb.s.sol", "--env", "deployer=anvil", "--env", "COUNTER_LABEL=list-show-test", "--env", "TOKEN_LABEL=list-show-test")
			require.NoError(t, err)

			// Run list command
			output, err := ctx.Treb("list")
			require.NoError(t, err)

			outputStr := string(output)
			t.Logf("List output:\n%s", outputStr)

			// Verify we see deployments
			assert.Contains(t, outputStr, "31337")
			assert.Contains(t, strings.ToLower(outputStr), "default")

			// Test list with chain filter
			output, err = ctx.Treb("list", "--chain", "31337")
			require.NoError(t, err)
			assert.Contains(t, string(output), "31337")

			// Test list with namespace filter
			output, err = ctx.Treb("list", "--namespace", "default")
			require.NoError(t, err)
			assert.Contains(t, strings.ToLower(string(output)), "default")
		})
	})
}

// Helper to remove old deployments file (no longer used with v2 registry)
func removeDeploymentsFile() {
	os.RemoveAll(filepath.Join(helpers.GetFixtureDir(), ".treb"))
}

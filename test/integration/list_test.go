package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestListCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name:     "list_empty",
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_deployments",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_multiple_chains",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_multiple_namespaces_and_chains",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338", "--namespace", "production"},
			},
			TestCmds: [][]string{
				s("list --namespace production"),
				s("list --namespace default"),
			},
			OutputArtifacts: []string{},
		},
		{
			Name: "list_with_proxy_relationships",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy", "--proxy-contract", "ERC1967Proxy.sol:ERC1967Proxy"},
				{"run", "DeployUpgradeableCounterProxy"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_labels",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v2"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v3"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_all_categories",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				// Deploy a library
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
				{"run", "script/deploy/DeployStringUtils.s.sol"},
				// Deploy a proxy with implementation
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy", "--proxy-contract", "ERC1967Proxy.sol:ERC1967Proxy"},
				{"run", "DeployUpgradeableCounterProxy"},
				// Deploy a singleton
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_filter_by_namespace",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "staging"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				{"list", "--namespace", "staging"},
				{"list", "--namespace", "production"},
			},
		},
		{
			Name: "list_filter_by_chain",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{
				{"list", "--network", "anvil-31337"},
				{"list", "--network", "anvil-31338"},
			},
		},
		{
			Name: "list_with_tags",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "Counter", "--add", "latest"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
				{"tag", "SampleToken", "--add", "token-v1"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_multiple_contract_types",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
				{"run", "script/deploy/DeployStringUtils.s.sol"},
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter"},
				{"run", "script/deploy/DeployUpgradeableCounter.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_json_output",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds:   [][]string{{"list", "--json"}},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				// Extract JSON from output (framework prepends "=== cmd N: ... ===\n")
				jsonStr := extractJSONObject(output)

				// Verify valid JSON output (now wrapped in object with "deployments" key)
				var result map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(jsonStr), &result))
				entries, ok := result["deployments"].([]interface{})
				require.True(t, ok, "expected deployments array")
				require.Len(t, entries, 1)
				entry := entries[0].(map[string]interface{})
				assert.Equal(t, "Counter", entry["contractName"])
				assert.NotEmpty(t, entry["address"])
				// No otherNamespaces when deployments exist
				assert.Nil(t, result["otherNamespaces"])
			},
		},
		{
			Name: "list_with_mixed_deployment_status",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				// Intentionally not deploying SampleToken to show pending
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_contracts_in_subdirectories",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/other/MyToken.sol:MyToken"},
				{"run", "script/deploy/DeployMyToken.s.sol"},
				{"gen", "deploy", "src/test-dir/SimpleStorage.sol:SimpleStorage"},
				{"run", "script/deploy/DeploySimpleStorage.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
		// Namespace discovery tests
		{
			Name: "list_namespace_discovery_hint",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				// Deploy Counter in default namespace
				{"run", "script/deploy/DeployCounter.s.sol"},
				// Deploy Counter in production namespace too
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				// List in staging namespace (empty) — should show discovery hint
				{"list", "--namespace", "staging"},
			},
			OutputArtifacts: []string{},
		},
		{
			Name: "list_namespace_discovery_json",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				// Deploy in default namespace
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				// JSON list in staging namespace (empty) — should include otherNamespaces
				{"list", "--namespace", "staging", "--json"},
			},
			SkipGolden:      true,
			OutputArtifacts: []string{},
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				jsonStr := extractJSONObject(output)

				var result map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(jsonStr), &result))

				// Deployments should be empty
				entries, ok := result["deployments"].([]interface{})
				require.True(t, ok, "expected deployments array")
				assert.Empty(t, entries)

				// otherNamespaces should include "default" with count
				otherNs, ok := result["otherNamespaces"].(map[string]interface{})
				require.True(t, ok, "expected otherNamespaces map")
				assert.Contains(t, otherNs, "default")
				assert.Equal(t, float64(1), otherNs["default"])
			},
		},
	}

	RunIntegrationTests(t, tests)
}

// extractJSONObject extracts a JSON object from output that may contain framework headers.
func extractJSONObject(output string) string {
	idx := strings.Index(output, "\n{")
	if idx >= 0 {
		return strings.TrimSpace(output[idx+1:])
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		return strings.TrimSpace(output)
	}
	return output
}

package compatibility

import (
	"testing"
)

func TestListCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name:     "list_empty",
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_deployments",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_multiple_chains",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_multiple_namespaces_and_chains",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338", "--namespace", "production"},
			},
			TestCmds:        [][]string{{"list"}},
			ExpectDiff:      true, // ordering in the transactions.json file is different
			OutputArtifacts: []string{},
		},
		{
			Name: "list_with_proxy_relationships",
			SetupCmds: [][]string{
				{"gen", "deploy", "UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
				{"run", "DeployUpgradeableCounterProxy"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_with_labels",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v2"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v3"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name: "list_all_flag",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "staging"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{{"list", "--all"}},
		},
		{
			Name: "list_filter_by_namespace",
			SetupCmds: [][]string{
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
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds:   [][]string{{"list", "--json"}},
			ExpectDiff: true,
			ExpectErr:  ErrorBoth, // --json flag doesn't exist in either v1 or v2
		},
		{
			Name: "list_with_mixed_deployment_status",
			SetupCmds: [][]string{
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
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/other/MyToken.sol:MyToken"},
				{"run", "script/deploy/DeployMyToken.s.sol"},
				{"gen", "deploy", "src/test-dir/SimpleStorage.sol:SimpleStorage"},
				{"run", "script/deploy/DeploySimpleStorage.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
	}

	RunCompatibilityTests(t, tests)
}

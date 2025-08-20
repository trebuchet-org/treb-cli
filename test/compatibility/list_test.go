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
			TestCmds:            [][]string{{"list"}},
			ExpectDiff:          true, // ordering in the transactions.json file is different
			IgnoreRegistryFiles: true,
		},
		{
			Name: "list_with_proxy_relationships",
			SetupCmds: [][]string{
				{"gen", "deploy", "UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
				{"run", "DeployUpgradeableCounterProxy"},
			},
			TestCmds: [][]string{{"list"}},
		},
	}

	RunCompatibilityTests(t, tests)
}

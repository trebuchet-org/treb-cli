package compatibility

import (
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestListCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name:     "list_empty",
			TestCmds: [][]string{{"list"}},
		},
		{
			Name:     "list_with_deployments",
			SetupCtx: helpers.NewTrebContext(t, helpers.BinaryV1),
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name:     "list_with_multiple_chains",
			SetupCtx: helpers.NewTrebContext(t, helpers.BinaryV1),
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name:     "list_with_multiple_namespaces_and_chains",
			SetupCtx: helpers.NewTrebContext(t, helpers.BinaryV1),
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338", "--namespace", "production"},
			},
			TestCmds: [][]string{{"list"}},
		},
		{
			Name:     "list_with_proxy_relationships",
			SetupCtx: helpers.NewTrebContext(t, helpers.BinaryV1),
			SetupCmds: [][]string{
				{"gen", "deploy", "UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
				{"run", "DeployUpgradeableCounterProxy"},
			},
			TestCmds: [][]string{{"list"}},
		},
	}

	RunCompatibilityTests(t, tests)
}

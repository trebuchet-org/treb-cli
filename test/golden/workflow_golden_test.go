package golden

import (
	"testing"
)

func TestDeploymentWorkflowGolden(t *testing.T) {
	tests := []GoldenTest{
		{
			Name: "full_deployment_flow",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"list"},
				{"show", "Counter"},
			},
			GoldenFile: "workflows/full_deployment_flow.golden",
		},
		{
			Name: "proxy_deployment_flow",
			TestCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter",
					"--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
				{"run", "script/deploy/DeployUpgradeableCounterProxy.s.sol"},
				{"list"},
				{"show", "UpgradeableCounter"},
			},
			GoldenFile: "workflows/proxy_deployment_flow.golden",
		},
		{
			Name: "multi_namespace_flow",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
				{"list"},
				{"list", "--namespace", "production"},
			},
			GoldenFile: "workflows/multi_namespace_flow.golden",
		},
	}

	RunGoldenTests(t, tests)
}

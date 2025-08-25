package integration

import (
	"testing"
)

func TestComposeCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "without_config",
			TestCmds: [][]string{
				{"compose", "nonexistent.yaml"},
			},
			ExpectErr: true,
		},
		{
			Name: "help",
			TestCmds: [][]string{
				{"compose", "--help"},
			},
		},
		{
			Name: "with_config",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
			},
			TestCmds: [][]string{
				{"compose", "compose.yaml", "--network", "anvil-31337"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}

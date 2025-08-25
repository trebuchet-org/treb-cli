package compatibility

import (
	"testing"
)

func TestComposeCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "without_config",
			TestCmds: [][]string{
				{"conpose", "nonexistent.yaml"},
			},
			ExpectErr: ErrorBoth, // Both versions will error (v1: file not found, v2: command not found)
		},
		{
			Name: "help",
			TestCmds: [][]string{
				{"compose", "--help"},
			},
			ExpectDiff: true,
		},
		{
			Name: "with_config",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
			},
			TestCmds: [][]string{
				{"compose", "compose.yaml", "--network", "anvil-31337"},
			},
			ExpectDiff: true,
		},
	}

	RunCompatibilityTests(t, tests)
}


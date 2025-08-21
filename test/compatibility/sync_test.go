package compatibility

import (
	"testing"
)

func TestSyncCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "sync_empty_registry",
			TestCmds: [][]string{
				{"sync"},
			},
		},
		{
			Name: "sync_with_deployments",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"sync"},
			},
			ExpectErr: ErrorBoth, // May fail on local anvil network
		},
		{
			Name: "sync_specific_chain",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{
				{"sync", "--network", "anvil-31337"},
			},
			ExpectErr: ErrorBoth, // May fail on local anvil network
		},
		{
			Name: "sync_dry_run",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"sync", "--dry-run"},
			},
			ExpectErr: ErrorBoth, // May fail on local anvil network
		},
		{
			Name: "sync_with_namespace",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				{"sync", "--namespace", "production"},
			},
			ExpectErr: ErrorBoth, // May fail on local anvil network
		},
		{
			Name: "sync_force_update",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"sync", "--force"},
			},
			ExpectErr: ErrorBoth, // May fail on local anvil network
		},
	}

	RunCompatibilityTests(t, tests)
}
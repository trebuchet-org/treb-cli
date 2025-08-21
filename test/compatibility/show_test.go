package compatibility

import (
	"testing"
)

func TestShowCommand(t *testing.T) {
	RunCompatibilityTests(t, []CompatibilityTest{
		{
			Name: "simple",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "Counter"},
			},
			ExpectDiff: true,
		},
		{
			Name: "with_label",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "-e", "LABEL=test"},
			},
			TestCmds: [][]string{
				{"show", "Counter:test"},
			},
			ExpectDiff: true,
		},
		{
			Name: "show_by_address",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				// Using deterministic address from CreateX
				{"show", "0x74148047D6bDf624C94eFc07F60cEE7b6052FB29"},
			},
			ExpectDiff: true, // v1 doesn't support showing by address
		},
		{
			Name: "show_nonexistent_contract",
			TestCmds: [][]string{
				{"show", "NonExistentContract"},
			},
			ExpectErr: ErrorBoth,
		},
		{
			Name: "show_with_namespace",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				{"show", "Counter", "--namespace", "production"},
			},
		},
		{
			Name: "show_with_tags",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"tag", "Counter", "--add", "v1.0.0"},
				{"tag", "Counter", "--add", "latest"},
			},
			TestCmds: [][]string{
				{"show", "Counter"},
			},
		},
		{
			Name: "show_library",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
				{"run", "script/deploy/DeployStringUtils.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "StringUtils"},
			},
		},
		{
			Name: "show_token_with_constructor_args",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "SampleToken"},
			},
		},
		{
			Name: "show_upgradeable_contract",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter"},
				{"run", "script/deploy/DeployUpgradeableCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "UpgradeableCounter"},
			},
		},
		{
			Name: "show_by_deployment_id",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "default/31337/Counter"},
			},
		},
		{
			Name: "show_multiple_deployments_different_networks",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{
				{"show", "Counter", "--network", "anvil-31337"},
				{"show", "Counter", "--network", "anvil-31338"},
			},
		},
		{
			Name: "show_contract_in_subdirectory",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/other/MyToken.sol:MyToken"},
				{"run", "script/deploy/DeployMyToken.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "MyToken"},
			},
		},
		{
			Name: "show_with_json_output",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "Counter", "--json"},
			},
			ExpectDiff: true,
			ExpectErr: ErrorOnlyV2, // v1 has --json flag but v2 doesn't
		},
	})
}

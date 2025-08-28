package integration

import (
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestRunCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "simple",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"ls"},
				{"show", "Counter"},
			},
		},
		{
			Name: "run_with_env_vars",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},
				{"show", "Counter:v1"},
			},
		},
		{
			Name: "run_dry_run",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--dry-run"},
				{"list"}, // Should be empty since dry-run doesn't persist
			},
		},
		{
			Name: "run_with_namespace",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
				{"show", "Counter", "--namespace", "production"},
			},
		},
		{
			Name: "run_with_network",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
				{"show", "Counter", "--network", "anvil-31338"},
			},
		},
		{
			Name: "run_multiple_contracts",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
				{"list"},
			},
		},
		{
			Name: "run_with_constructor_args",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeploySampleToken.s.sol"},
				{"show", "SampleToken"},
			},
		},
		{
			Name: "run_library_deployment",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployStringUtils.s.sol"},
				{"show", "StringUtils"},
			},
		},
		{
			Name: "run_upgradeable_contract",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployUpgradeableCounter.s.sol"},
				{"show", "UpgradeableCounter"},
			},
		},
		{
			Name: "run_with_debug",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--debug"},
			},
		},
		{
			Name: "run_nonexistent_script",
			TestCmds: [][]string{
				s("run script/deploy/NonExistent.s.sol --network anvil-31337"),
			},
			ExpectErr: true,
		},
		{
			Name: "run_with_multiple_env_vars",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=test", "--env", "VERSION=1.0.0"},
				{"show", "Counter:test"},
			},
		},
		{
			Name: "run_subdirectory_contract",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/other/MyToken.sol:MyToken"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployMyToken.s.sol"},
				{"show", "MyToken"},
			},
		},
		{
			Name: "run_with_json_output",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--debug-json"},
			},
		},
		{
			Name: "run_redeployment_same_label",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"}, // Should handle gracefully
			},
		},
		{
			Name: "run_with_custom_proxy",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"run", "script/DeployCustomProxy.s.sol"},
				{"list"},
			},
			Normalizers: []helpers.Normalizer{
				helpers.LegacySolidityNormalizer{},
			},
		},
		{
			Name: "run_deploy_with_library",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/TestWithNewLib.sol:MathUtils"},
				{"run", "script/deploy/DeployMathUtils.s.sol"},
				{"gen", "deploy", "src/TestWithNewLib.sol:TestWithNewLib"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployTestWithNewLib.s.sol"},
				{"show", "TestWithNewLib"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}

package compatibility

import (
	"testing"
	// "github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestGenCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "gen_deploy_singleton",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
		},
		{
			Name: "gen_deploy_no_contract",
			TestCmds: [][]string{
				{"gen", "deploy", "DoesntExist"},
			},
			ExpectErr:  ErrorBoth,
			ExpectDiff: true,
		},
		{
			Name: "gen_deploy_multiple",
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr:  ErrorBoth,
			ExpectDiff: true,
		},
		{
			Name: "gen_deploy_library",
			TestCmds: [][]string{
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
			},
		},
		{
			Name: "gen_deploy_token",
			TestCmds: [][]string{
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
			},
		},
		{
			Name: "gen_deploy_upgradeable",
			TestCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter"},
			},
		},
		{
			Name: "gen_deploy_with_proxy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy"},
			},
			ExpectDiff: true, // Different proxy handling between versions
		},
		{
			Name: "gen_deploy_with_specific_proxy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
			},
			ExpectDiff: true, // Different proxy handling between versions
		},
		{
			Name: "gen_deploy_with_strategy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3"},
			},
		},
		{
			Name: "gen_deploy_subdirectory",
			TestCmds: [][]string{
				{"gen", "deploy", "src/other/MyToken.sol:MyToken"},
			},
		},
		{
			Name: "gen_deploy_force_overwrite",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--force"},
			},
		},
		{
			Name: "gen_deploy_custom_output",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--output", "script/custom/DeployMyCounter.s.sol"},
			},
		},
		{
			Name: "gen_deploy_with_imports",
			TestCmds: [][]string{
				{"gen", "deploy", "src/TestWithNewLib.sol:TestWithNewLib"},
			},
		},
		{
			Name: "gen_deploy_abstract_contract",
			TestCmds: [][]string{
				{"gen", "deploy", "src/test-dir/Counter.sol:Counter"}, // Different counter in subdirectory
			},
		},
		{
			Name: "gen_deploy_no_strategy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE"},
			},
		},
		{
			Name: "gen_deploy_create2_strategy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE2"},
			},
		},
		{
			Name: "gen_library",
			TestCmds: [][]string{
				{"gen", "library", "StringUtils"},
			},
			ExpectDiff: true, // May have different implementations
		},
		{
			Name: "gen_proxy",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter"},
			},
			TestCmds: [][]string{
				{"gen", "proxy", "UpgradeableCounter"},
			},
			ExpectDiff: true, // Different proxy generation approaches
		},
	}

	RunCompatibilityTests(t, tests)
}

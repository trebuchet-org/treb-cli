package integration

import (
	"testing"
	// "github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestGenCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "gen_deploy_singleton",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_no_contract",
			TestCmds: [][]string{
				{"gen", "deploy", "DoesntExist"},
			},
			ExpectErr: true,
		},
		{
			Name: "gen_deploy_multiple",
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr: true,
		},
		{
			Name: "gen_deploy_library",
			TestCmds: [][]string{
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployStringUtils.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_token",
			TestCmds: [][]string{
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeploySampleToken.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_upgradeable",
			TestCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployUpgradeableCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_with_specific_proxy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter", "--proxy", "--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployUpgradeableCounterProxy.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_with_strategy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_subdirectory",
			TestCmds: [][]string{
				{"gen", "deploy", "src/other/MyToken.sol:MyToken"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployMyToken.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_custom_output",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--script-path", "script/custom/DeployMyCounter.s.sol"},
			},
			OutputArtifacts: append(
				[]string{"script/custom/DeployMyCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_with_imports",
			TestCmds: [][]string{
				{"gen", "deploy", "src/TestWithNewLib.sol:TestWithNewLib"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployTestWithNewLib.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_abstract_contract",
			TestCmds: [][]string{
				{"gen", "deploy", "src/test-dir/Counter.sol:Counter"}, // Different counter in subdirectory
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
		{
			Name: "gen_deploy_no_strategy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
			ExpectErr: true,
		},
		{
			Name: "gen_deploy_create2_strategy",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE2"},
			},
			OutputArtifacts: append(
				[]string{"script/deploy/DeployCounter.s.sol"},
				DefaultOutputArtifacs...,
			),
		},
	}

	RunIntegrationTests(t, tests)
}

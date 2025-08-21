package compatibility

import (
	"testing"
)

func TestVerifyCommand(t *testing.T) {
	_ = []CompatibilityTest{
		{
			Name: "verify_single_deployment",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"verify", "Counter"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_nonexistent_deployment",
			TestCmds: [][]string{
				{"verify", "NonExistent"},
			},
			ExpectErr: ErrorBoth,
		},
		{
			Name: "verify_by_address",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"verify", "0x74148047D6bDf624C94eFc07F60cEE7b6052FB29"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_with_namespace",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},
			},
			TestCmds: [][]string{
				{"verify", "Counter", "--namespace", "production"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_library",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
				{"run", "script/deploy/DeployStringUtils.s.sol"},
			},
			TestCmds: [][]string{
				{"verify", "StringUtils"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_token_with_constructor_args",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"verify", "SampleToken"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_all_deployments",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"verify", "--all"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_specific_chain",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			TestCmds: [][]string{
				{"verify", "Counter", "--network", "anvil-31338"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_with_label",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},
			},
			TestCmds: [][]string{
				{"verify", "Counter:v1"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
		{
			Name: "verify_deployment_id",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"verify", "default/31337/Counter"},
			},
			ExpectErr: ErrorBoth, // Will fail without real network/API key
		},
	}

	t.Skip("Verify needs a HTTP mock for etherscan.")
	// RunCompatibilityTests(t, tests)
}


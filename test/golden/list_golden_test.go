package golden

import (
	"testing"
)

func TestListCommandGolden(t *testing.T) {
	setup := [][]string{
		{"gen", "deploy", "src/Counter.sol:Counter"},
		{"run", "script/deploy/DeployCounter.s.sol"},
		{"run", "script/deploy/DeployCounter.s.sol", "--env", "label=prod", "--namespace", "production"},
	}

	tests := []GoldenTest{
		{
			Name:       "empty",
			TestCmds:   [][]string{{"list"}},
			GoldenFile: "commands/list/empty.golden",
		},
		{
			Name:       "default_with_deployments",
			SetupCmds:  setup,
			TestCmds:   [][]string{{"list"}},
			GoldenFile: "commands/list/default.golden",
		},
		{
			Name:       "with_namespace",
			SetupCmds:  setup,
			TestCmds:   [][]string{{"list", "--namespace", "production"}},
			GoldenFile: "commands/list/with_namespace.golden",
		},
		{
			Name:       "with_chain",
			SetupCmds:  setup,
			TestCmds:   [][]string{{"list", "--chain", "31337"}},
			GoldenFile: "commands/list/with_chain.golden",
		},
		{
			Name:       "with_contract",
			SetupCmds:  setup,
			TestCmds:   [][]string{{"list", "--contract", "Counter"}},
			GoldenFile: "commands/list/with_contract.golden",
		},
	}

	RunGoldenTests(t, tests)
}

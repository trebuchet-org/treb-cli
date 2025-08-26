package integration

import (
	"testing"
)

func TestInitCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "init_new_project",
			TestCmds: [][]string{
				{"init"},
			},
		},
		{
			Name: "init_existing_project",
			SetupCmds: [][]string{
				{"init"}, // First init
			},
			TestCmds: [][]string{
				{"init"}, // Should handle gracefully when already initialized
			},
		},
		{
			Name: "init_and_deploy",
			TestCmds: [][]string{
				{"init"},
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol", "-n", "anvil-31337"},
				{"list"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}

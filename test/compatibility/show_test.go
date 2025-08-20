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
		},
	})
}

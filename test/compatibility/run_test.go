package compatibility

import (
	"testing"
)

func TestRunCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "simple",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"ls"},
				{"show", "Counter"},
			},
			ExpectDiff: true,
		},
	}

	RunCompatibilityTests(t, tests)
}

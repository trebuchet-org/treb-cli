package compatibility

import (
	"testing"
)

func TestRunCommandCompatibility(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "simple",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
		},
	}

	RunCompatibilityTests(t, tests)
}

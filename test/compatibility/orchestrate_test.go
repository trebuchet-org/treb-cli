package compatibility

import (
	"testing"
)

func TestOrchestrateCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "orchestrate_without_config",
			TestCmds: [][]string{
				{"orchestrate", "nonexistent.yaml"},
			},
			ExpectErr: true,
		},
		// Note: orchestrate is a v1-only command, so most tests will fail for v2
		// These tests primarily document the v1 behavior
		{
			Name: "orchestrate_help",
			TestCmds: [][]string{
				{"orchestrate", "--help"},
			},
			ExpectDiff: true, // v2 won't have this command
		},
	}

	RunCompatibilityTests(t, tests)
}
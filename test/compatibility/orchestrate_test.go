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
			ExpectErr: ErrorBoth, // Both versions will error (v1: file not found, v2: command not found)
		},
		// Note: orchestrate is a v1-only command, so most tests will fail for v2
		// These tests primarily document the v1 behavior
		{
			Name: "orchestrate_help",
			TestCmds: [][]string{
				{"orchestrate", "--help"},
			},
			ExpectDiff: true,
			ExpectErr: ErrorOnlyV2, // v2 doesn't have orchestrate command
		},
	}

	RunCompatibilityTests(t, tests)
}
package compatibility

import (
	"testing"
)

func TestDevCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "dev_help",
			TestCmds: [][]string{
				{"dev", "--help"},
			},
			ExpectDiff: true, // Different help text between versions
		},
		{
			Name: "dev_build",
			TestCmds: [][]string{
				{"dev", "build"},
			},
			ExpectDiff: true, // Different implementations
		},
		{
			Name: "dev_anvil_status",
			TestCmds: [][]string{
				{"dev", "anvil", "status"},
			},
			ExpectDiff: true, // Different implementations
		},
		// Note: Not testing anvil start/stop to avoid conflicts with test infrastructure
	}

	RunCompatibilityTests(t, tests)
}
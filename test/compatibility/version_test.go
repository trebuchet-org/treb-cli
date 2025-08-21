package compatibility

import (
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "version_basic",
			TestCmds: [][]string{
				{"version"},
			},
			ExpectDiff: true, // Different version outputs expected
		},
		{
			Name: "version_short_flag",
			TestCmds: [][]string{
				{"-v"},
			},
			ExpectDiff: true, // Different version outputs expected
			ExpectErr: ErrorBoth, // Both versions don't support -v as a standalone flag
		},
		{
			Name: "version_long_flag",
			TestCmds: [][]string{
				{"--version"},
			},
			ExpectDiff: true, // Different version outputs expected
			ExpectErr: ErrorBoth, // Both versions don't support --version as a standalone flag
		},
	}

	RunCompatibilityTests(t, tests)
}
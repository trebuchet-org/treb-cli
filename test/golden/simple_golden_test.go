package golden

import (
	"testing"
)

// TestSimpleCommandsGolden tests commands that don't require deployments
func TestSimpleCommandsGolden(t *testing.T) {
	tests := []GoldenTest{
		{
			Name:       "help",
			TestCmds:   [][]string{{"--help"}},
			GoldenFile: "commands/help/default.golden",
		},
		{
			Name:       "version",
			TestCmds:   [][]string{{"version"}},
			GoldenFile: "commands/version/default.golden",
		},
		{
			Name:       "list_empty",
			TestCmds:   [][]string{{"list"}},
			GoldenFile: "commands/list/empty.golden",
		},
		{
			Name:       "show_not_found",
			TestCmds:   [][]string{{"show", "NonExistent"}},
			GoldenFile: "commands/show/not_found.golden",
			ExpectErr:  true,
		},
		{
			Name:       "verify_not_found",
			TestCmds:   [][]string{{"verify", "NonExistent"}},
			GoldenFile: "commands/verify/not_found.golden",
			ExpectErr:  true,
		},
		{
			Name:       "config_show",
			TestCmds:   [][]string{{"config", "show"}},
			GoldenFile: "commands/config/show.golden",
		},
		{
			Name:       "networks",
			TestCmds:   [][]string{{"networks"}},
			GoldenFile: "commands/networks/default.golden",
		},
	}

	RunGoldenTests(t, tests)
}

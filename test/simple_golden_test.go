package integration_test

import (
	"testing"
)

// TestSimpleCommandsGolden tests commands that don't require deployments
func TestSimpleCommandsGolden(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		goldenFile string
		expectErr  bool
	}{
		{
			name:       "help",
			args:       []string{"--help"},
			goldenFile: "commands/help/default.golden",
		},
		{
			name:       "version",
			args:       []string{"version"},
			goldenFile: "commands/version/default.golden",
		},
		{
			name:       "list_empty",
			args:       []string{"list"},
			goldenFile: "commands/list/empty.golden",
		},
		{
			name:       "show_not_found",
			args:       []string{"show", "NonExistent"},
			goldenFile: "commands/show/not_found.golden",
			expectErr:  true,
		},
		{
			name:       "verify_not_found",
			args:       []string{"verify", "NonExistent"},
			goldenFile: "commands/verify/not_found.golden",
			expectErr:  true,
		},
		{
			name:       "config_show",
			args:       []string{"config", "show"},
			goldenFile: "commands/config/show.golden",
		},
		{
			name:       "networks",
			args:       []string{"networks"},
			goldenFile: "commands/networks/default.golden",
		},
	}

	for _, tt := range tests {
		test := tt // capture range variable
		IsolatedTest(t, test.name, func(t *testing.T, ctx *TrebContext) {
			if test.expectErr {
				ctx.trebGoldenWithError(test.goldenFile, test.args...)
			} else {
				ctx.trebGolden(test.goldenFile, test.args...)
			}
		})
	}
}
package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
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
		helpers.IsolatedTest(t, test.name, func(t *testing.T, ctx *helpers.TrebContext) {
			if test.expectErr {
				TrebGoldenWithError(t, ctx, test.goldenFile, test.args...)
			} else {
				TrebGolden(t, ctx, test.goldenFile, test.args...)
			}
		})
	}
}


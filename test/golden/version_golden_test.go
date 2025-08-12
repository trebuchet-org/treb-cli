package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"testing"
)

func TestVersionGolden(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		goldenFile string
	}{
		{
			name:       "version",
			args:       []string{"version"},
			goldenFile: "commands/version/default.golden",
		},
		{
			name:       "help",
			args:       []string{"--help"},
			goldenFile: "commands/help/default.golden",
		},
	}

	for _, tt := range tests {
		helpers.IsolatedTest(t, tt.name, func(t *testing.T, ctx *helpers.TrebContext) {
			output, err := ctx.Treb(tt.args...)
			if err != nil {
				t.Fatalf("Command failed unexpectedly: %v\nArgs: %v\nOutput:\n%s", err, tt.args, output)
			}

			compareGolden(t, output, GoldenConfig{
				Path:        tt.goldenFile,
				Normalizers: getDefaultNormalizers(),
			})
		})
	}
}

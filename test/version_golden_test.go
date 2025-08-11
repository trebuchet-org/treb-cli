package integration_test

import (
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
		t.Run(tt.name, func(t *testing.T) {
			output, err := runTreb(t, tt.args...)
			if err != nil {
				t.Fatalf("Command failed unexpectedly: %v\nArgs: %v\nOutput:\n%s", err, tt.args, output)
			}

			// Use custom normalizers for version command
			normalizers := []Normalizer{
				ColorNormalizer{},
				TimestampNormalizer{},
				PathNormalizer{},
			}

			// Add version normalizer for version command
			if tt.name == "version" {
				normalizers = append(normalizers, VersionNormalizer{})
			}

			compareGolden(t, output, GoldenConfig{
				Path:        tt.goldenFile,
				Normalizers: normalizers,
			})
		})
	}
}


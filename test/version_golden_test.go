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
			runTrebGolden(t, tt.goldenFile, tt.args...)
		})
	}
}
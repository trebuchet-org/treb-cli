package integration_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/cli"
)

// TestShowArchitectureCompatibility verifies that the new show command produces
// output compatible with the existing golden files
func TestShowArchitectureCompatibility(t *testing.T) {
	if os.Getenv("TEST_NEW_ARCHITECTURE") != "true" {
		t.Skip("New architecture tests disabled. Set TEST_NEW_ARCHITECTURE=true to run")
	}

	tests := []struct {
		name      string
		setup     func(t *testing.T, ctx *TrebContext)
		args      []string
		goldenFile string
		expectErr bool
	}{
		{
			name: "show_counter",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"Counter"},
			goldenFile: "commands/show/counter.golden",
		},
		{
			name: "show_with_namespace",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"Counter", "--namespace", "production"},
			goldenFile: "commands/show/counter_production.golden",
		},
		{
			name:       "show_not_found",
			args:       []string{"NonExistent"},
			goldenFile: "commands/show/not_found.golden",
			expectErr:  true,
		},
		// Skip show by address test for now as it requires getting the actual address
		// Skip proxy test as it requires proxy setup
	}

	for _, tt := range tests {
		IsolatedTest(t, tt.name, func(t *testing.T, ctx *TrebContext) {
			if tt.setup != nil {
				tt.setup(t, ctx)
			}

			// Load config
			cfg, err := app.LoadConfig()
			require.NoError(t, err)
			cfg.ProjectRoot = fixtureDir

			// Create the new show command
			showCmd := cli.NewShowCmd(cfg)
			showCmd.SetArgs(tt.args)

			// Capture output
			output, err := captureCommandOutput(showCmd)
			if tt.expectErr {
				require.Error(t, err)
				output = err.Error()
			} else {
				require.NoError(t, err)
			}

			// Compare with golden file
			normalizers := []Normalizer{
				ColorNormalizer{},
				TimestampNormalizer{},
				PathNormalizer{},
				AddressNormalizer{},
				HashNormalizer{},
				GitCommitNormalizer{},
				// Add chain ID normalizer for network display
				ChainIDNormalizer{},
			}

			compareGolden(t, output, GoldenConfig{
				Path:        tt.goldenFile,
				Normalizers: normalizers,
			})
		})
	}
}

// ChainIDNormalizer normalizes chain IDs in network display
type ChainIDNormalizer struct{}

func (n ChainIDNormalizer) Normalize(output string) string {
	// Normalize "Network: 31337" to "Network: <CHAIN_ID>"
	return regexp.MustCompile(`Network: \d+`).ReplaceAllString(output, "Network: <CHAIN_ID>")
}
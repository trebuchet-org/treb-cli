package integration_test

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/cli"
)

// TestArchitectureCompatibility verifies that the new architecture produces 
// output compatible with the existing golden files
func TestArchitectureCompatibility(t *testing.T) {
	if os.Getenv("TEST_NEW_ARCHITECTURE") != "true" {
		t.Skip("New architecture tests disabled. Set TEST_NEW_ARCHITECTURE=true to run")
	}

	tests := []struct {
		name      string
		setup     func(t *testing.T, ctx *TrebContext)
		args      []string
		goldenFile string
	}{
		{
			name:       "list_empty",
			args:       []string{},
			goldenFile: "commands/list/empty.golden",
		},
		{
			name: "list_with_deployments",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{},
			goldenFile: "commands/list/default.golden",
		},
		{
			name: "list_with_namespace",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"--namespace", "production"},
			goldenFile: "commands/list/with_namespace.golden",
		},
		{
			name: "list_with_chain",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"--chain", "31337"},
			goldenFile: "commands/list/with_chain.golden",
		},
		{
			name: "list_with_contract",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"--contract", "Counter"},
			goldenFile: "commands/list/with_contract.golden",
		},
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

			// Create the new list command
			listCmd := cli.NewListCmd(cfg)
			listCmd.SetArgs(tt.args)

			// Capture output
			output, err := captureCommandOutput(listCmd)
			require.NoError(t, err)

			// Compare with golden file
			normalizers := []Normalizer{
				ColorNormalizer{},
				TimestampNormalizer{},
				PathNormalizer{},
				AddressNormalizer{},
				HashNormalizer{},
				GitCommitNormalizer{},
			}

			compareGolden(t, output, GoldenConfig{
				Path:        tt.goldenFile,
				Normalizers: normalizers,
			})
		})
	}
}

// Helper to run the command and capture output
func captureCommandOutput(cmd *cobra.Command) (string, error) {
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	err := cmd.Execute()
	return buf.String(), err
}
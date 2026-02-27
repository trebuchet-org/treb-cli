package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestMigrateConfigCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "migrate_config_creates_treb_toml",
			TestCmds: [][]string{
				{"migrate-config"},
			},
			OutputArtifacts: []string{"treb.toml"},
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				// Verify treb.toml was created with expected content
				data, err := os.ReadFile(filepath.Join(ctx.WorkDir, "treb.toml"))
				require.NoError(t, err)
				content := string(data)

				assert.Contains(t, content, "[ns.default]")
				assert.Contains(t, content, `profile = "default"`)
				assert.Contains(t, content, "[ns.default.senders.anvil]")
				assert.Contains(t, content, `type = "private_key"`)

				// Verify foundry.toml is untouched (non-interactive does NOT modify it)
				foundryData, err := os.ReadFile(filepath.Join(ctx.WorkDir, "foundry.toml"))
				require.NoError(t, err)
				foundryContent := string(foundryData)
				assert.Contains(t, foundryContent, "[profile.default.treb.senders.anvil]")
			},
		},
		{
			Name: "migrate_config_no_config_to_migrate",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Replace foundry.toml with one that has no treb sections
				foundryContent := `[profile.default]
src = "src"
out = "out"

[rpc_endpoints]
anvil-31337 = "http://localhost:8545"
`
				err := os.WriteFile(filepath.Join(ctx.WorkDir, "foundry.toml"), []byte(foundryContent), 0644)
				require.NoError(t, err)
			},
			TestCmds: [][]string{
				{"migrate-config"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				assert.Contains(t, output, "nothing to migrate")

				// treb.toml should not exist
				_, err := os.Stat(filepath.Join(ctx.WorkDir, "treb.toml"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			Name: "migrate_config_overwrites_existing_treb_toml",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Create an existing treb.toml
				err := os.WriteFile(filepath.Join(ctx.WorkDir, "treb.toml"), []byte("# old content\n"), 0644)
				require.NoError(t, err)
			},
			TestCmds: [][]string{
				{"migrate-config"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				// Should warn about overwrite
				assert.Contains(t, output, "already exists")

				// treb.toml should have new content
				data, err := os.ReadFile(filepath.Join(ctx.WorkDir, "treb.toml"))
				require.NoError(t, err)
				content := string(data)
				assert.Contains(t, content, "[ns.default]")
				assert.NotContains(t, content, "old content")
			},
		},
		{
			Name: "migrate_config_then_commands_work",
			TestCmds: [][]string{
				{"migrate-config"},
				{"config"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				// treb.toml should exist and be valid
				data, err := os.ReadFile(filepath.Join(ctx.WorkDir, "treb.toml"))
				require.NoError(t, err)
				assert.Contains(t, string(data), "[ns.default]")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

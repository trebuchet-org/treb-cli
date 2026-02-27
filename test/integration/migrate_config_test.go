package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

// readFixtureFoundryTomlWithTreb returns a foundry.toml content that includes
// legacy [profile.*.treb.*] sections, for testing the migrate-config command.
func readFixtureFoundryTomlWithTreb(t *testing.T) []byte {
	t.Helper()
	content := `[profile.default]
src = "src"
out = "out"
libs = ["lib"]
test = "test"
script = "script"
optimizer_runs = 0
fs_permissions = [{ access = "read-write", path = "./" }]
bytecode_hash = "none"
cbor_metadata = false

[lint]
lint_on_build = false

[rpc_endpoints]
celo-sepolia = "https://forno.celo-sepolia.celo-testnet.org"
base-sepolia = "https://sepolia.base.org"
polygon = "https://polygon-bor-rpc.publicnode.com"
celo = "https://forno.celo.org"
anvil-31337 = "http://localhost:8545"
anvil-31338 = "http://localhost:9545"

[etherscan]
sepolia = { key = "${ETHERSCAN_API_KEY}" }
celo-sepolia = { key = "${ETHERSCAN_API_KEY}", chain = 11142220 }
celo = { key = "${ETHERSCAN_API_KEY}", chain = 42220 }

[profile.default.treb.senders.anvil]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

[profile.default.treb.senders.governor]
type = "oz_governor"
governor = "${GOVERNOR_ADDRESS}"
timelock = "${TIMELOCK_ADDRESS}"
proposer = "anvil"

[profile.live.treb.senders.safe0]
type = "safe"
safe = "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F"
signer = "signer0"

[profile.live.treb.senders.safe1]
type = "safe"
safe = "0x8dcD47D7aC5FEBC1E49a532644D21cd9D9dd97b2"
signer = "signer0"

[profile.live.treb.senders.signer0]
type = "private_key"
private_key="${BASE_SEPOLIA_SIGNER0_PK}"
`
	return []byte(content)
}

func TestMigrateConfigCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "migrate_config_creates_treb_toml",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Remove the treb.toml from fixture so migrate-config can create it
				os.Remove(filepath.Join(ctx.WorkDir, "treb.toml"))
				// Write foundry.toml with legacy treb sections for migration
				foundryContent := readFixtureFoundryTomlWithTreb(t)
				err := os.WriteFile(filepath.Join(ctx.WorkDir, "foundry.toml"), foundryContent, 0644)
				require.NoError(t, err)
			},
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
				// Remove treb.toml from fixture
				os.Remove(filepath.Join(ctx.WorkDir, "treb.toml"))
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
				// Write foundry.toml with legacy treb sections for migration
				foundryContent := readFixtureFoundryTomlWithTreb(t)
				err := os.WriteFile(filepath.Join(ctx.WorkDir, "foundry.toml"), foundryContent, 0644)
				require.NoError(t, err)
				// Create an existing treb.toml with old content
				err = os.WriteFile(filepath.Join(ctx.WorkDir, "treb.toml"), []byte("# old content\n"), 0644)
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
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Remove treb.toml so migrate-config can create it
				os.Remove(filepath.Join(ctx.WorkDir, "treb.toml"))
				// Write foundry.toml with legacy treb sections for migration
				foundryContent := readFixtureFoundryTomlWithTreb(t)
				err := os.WriteFile(filepath.Join(ctx.WorkDir, "foundry.toml"), foundryContent, 0644)
				require.NoError(t, err)
			},
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

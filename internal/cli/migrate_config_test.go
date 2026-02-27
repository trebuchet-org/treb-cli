package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestExtractTrebProfiles(t *testing.T) {
	t.Run("nil foundry config returns nil", func(t *testing.T) {
		profiles := extractTrebProfiles(nil)
		assert.Nil(t, profiles)
	})

	t.Run("no treb config in any profile", func(t *testing.T) {
		fc := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"default": {SrcPath: "src"},
			},
		}
		profiles := extractTrebProfiles(fc)
		assert.Empty(t, profiles)
	})

	t.Run("extracts profiles with treb senders", func(t *testing.T) {
		fc := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"default": {
					Treb: &config.TrebConfig{
						Senders: map[string]config.SenderConfig{
							"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
						},
					},
				},
				"staging": {
					Treb: &config.TrebConfig{
						Senders: map[string]config.SenderConfig{
							"deployer": {Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
						},
					},
				},
				"other": {SrcPath: "src"}, // no treb config
			},
		}
		profiles := extractTrebProfiles(fc)
		require.Len(t, profiles, 2)
		// default should come first
		assert.Equal(t, "default", profiles[0].Name)
		assert.Equal(t, "staging", profiles[1].Name)
	})

	t.Run("skips profiles with nil senders", func(t *testing.T) {
		fc := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"default": {
					Treb: &config.TrebConfig{
						Senders: map[string]config.SenderConfig{},
					},
				},
			},
		}
		profiles := extractTrebProfiles(fc)
		assert.Empty(t, profiles)
	})

	t.Run("sort order: default first then alphabetical", func(t *testing.T) {
		fc := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"production": {Treb: &config.TrebConfig{Senders: map[string]config.SenderConfig{"a": {Type: "private_key"}}}},
				"default":    {Treb: &config.TrebConfig{Senders: map[string]config.SenderConfig{"b": {Type: "private_key"}}}},
				"beta":       {Treb: &config.TrebConfig{Senders: map[string]config.SenderConfig{"c": {Type: "private_key"}}}},
			},
		}
		profiles := extractTrebProfiles(fc)
		require.Len(t, profiles, 3)
		assert.Equal(t, "default", profiles[0].Name)
		assert.Equal(t, "beta", profiles[1].Name)
		assert.Equal(t, "production", profiles[2].Name)
	})
}

func TestGenerateTrebToml(t *testing.T) {
	t.Run("single profile with private key sender", func(t *testing.T) {
		profiles := []trebProfile{
			{
				Name: "default",
				Senders: map[string]config.SenderConfig{
					"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"},
				},
			},
		}
		content := generateTrebToml(profiles)
		assert.Contains(t, content, "[ns.default]")
		assert.Contains(t, content, `profile = "default"`)
		assert.Contains(t, content, "[ns.default.senders.deployer]")
		assert.Contains(t, content, `type = "private_key"`)
		assert.Contains(t, content, `private_key = "${DEPLOYER_PRIVATE_KEY}"`)
	})

	t.Run("multiple profiles with different sender types", func(t *testing.T) {
		profiles := []trebProfile{
			{
				Name: "default",
				Senders: map[string]config.SenderConfig{
					"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
				},
			},
			{
				Name: "production",
				Senders: map[string]config.SenderConfig{
					"safe":     {Type: config.SenderTypeSafe, Safe: "0xABC", Signer: "proposer"},
					"proposer": {Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
				},
			},
		}
		content := generateTrebToml(profiles)

		assert.Contains(t, content, "[ns.default]")
		assert.Contains(t, content, "[ns.production]")
		assert.Contains(t, content, `profile = "production"`)
		assert.Contains(t, content, "[ns.production.senders.proposer]")
		assert.Contains(t, content, `type = "ledger"`)
		assert.Contains(t, content, `derivation_path = "m/44'/60'/0'/0/0"`)
		assert.Contains(t, content, "[ns.production.senders.safe]")
		assert.Contains(t, content, `type = "safe"`)
		assert.Contains(t, content, `safe = "0xABC"`)
		assert.Contains(t, content, `signer = "proposer"`)
	})

	t.Run("includes header comments", func(t *testing.T) {
		profiles := []trebProfile{
			{Name: "default", Senders: map[string]config.SenderConfig{"d": {Type: "private_key"}}},
		}
		content := generateTrebToml(profiles)
		assert.Contains(t, content, "# treb.toml")
		assert.Contains(t, content, "Migrated from foundry.toml")
	})

	t.Run("all sender fields rendered when set", func(t *testing.T) {
		profiles := []trebProfile{
			{
				Name: "default",
				Senders: map[string]config.SenderConfig{
					"full": {
						Type:           config.SenderTypeOZGovernor,
						Address:        "0x123",
						Governor:       "0xGOV",
						Timelock:       "0xTL",
						Proposer:       "0xPROP",
						PrivateKey:     "${PK}",
						Safe:           "0xSAFE",
						Signer:         "signer1",
						DerivationPath: "m/44",
					},
				},
			},
		}
		content := generateTrebToml(profiles)
		assert.Contains(t, content, `address = "0x123"`)
		assert.Contains(t, content, `governor = "0xGOV"`)
		assert.Contains(t, content, `timelock = "0xTL"`)
		assert.Contains(t, content, `proposer = "0xPROP"`)
		assert.Contains(t, content, `private_key = "${PK}"`)
		assert.Contains(t, content, `safe = "0xSAFE"`)
		assert.Contains(t, content, `signer = "signer1"`)
		assert.Contains(t, content, `derivation_path = "m/44"`)
	})

	t.Run("omits empty sender fields", func(t *testing.T) {
		profiles := []trebProfile{
			{
				Name: "default",
				Senders: map[string]config.SenderConfig{
					"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
				},
			},
		}
		content := generateTrebToml(profiles)
		assert.NotContains(t, content, "address =")
		assert.NotContains(t, content, "safe =")
		assert.NotContains(t, content, "signer =")
		assert.NotContains(t, content, "derivation_path =")
		assert.NotContains(t, content, "governor =")
		assert.NotContains(t, content, "timelock =")
		assert.NotContains(t, content, "proposer =")
	})
}

func TestRunMigrateConfig(t *testing.T) {
	t.Run("no treb config prints message and exits", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {SrcPath: "src"},
				},
			},
			NonInteractive: true,
		}

		err := runMigrateConfig(cfg)
		require.NoError(t, err)

		// treb.toml should NOT be written
		_, err = os.Stat(filepath.Join(tmpDir, "treb.toml"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("non-interactive writes treb.toml without prompts", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrateConfig(cfg)
		require.NoError(t, err)

		// treb.toml should be written
		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "[ns.default]")
		assert.Contains(t, string(data), `type = "private_key"`)
	})

	t.Run("non-interactive overwrites existing treb.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		existingPath := filepath.Join(tmpDir, "treb.toml")
		require.NoError(t, os.WriteFile(existingPath, []byte("old content"), 0644))

		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrateConfig(cfg)
		require.NoError(t, err)

		// Should be overwritten with new content
		data, err := os.ReadFile(existingPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "[ns.default]")
		assert.NotContains(t, string(data), "old content")
	})

	t.Run("multiple profiles converted correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
							},
						},
					},
					"production": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"safe":     {Type: config.SenderTypeSafe, Safe: "0xABC", Signer: "proposer"},
								"proposer": {Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrateConfig(cfg)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "[ns.default]")
		assert.Contains(t, content, "[ns.production]")
		assert.Contains(t, content, "[ns.production.senders.safe]")
		assert.Contains(t, content, "[ns.production.senders.proposer]")
	})

	t.Run("non-interactive does not modify foundry.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		foundryContent := `[profile.default]
src = "src"

[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "0xkey"
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644))

		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "0xkey"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrateConfig(cfg)
		require.NoError(t, err)

		// foundry.toml should be untouched
		data, err := os.ReadFile(filepath.Join(tmpDir, "foundry.toml"))
		require.NoError(t, err)
		assert.Equal(t, foundryContent, string(data))
	})
}

func TestRemoveTrebFromFoundryToml(t *testing.T) {
	t.Run("removes single treb section", func(t *testing.T) {
		input := `[profile.default]
src = "src"

[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "0xkey"

[rpc_endpoints]
anvil = "http://localhost:8545"
`
		result := removeTrebFromFoundryToml(input)
		assert.Contains(t, result, `[profile.default]`)
		assert.Contains(t, result, `src = "src"`)
		assert.Contains(t, result, `[rpc_endpoints]`)
		assert.Contains(t, result, `anvil = "http://localhost:8545"`)
		assert.NotContains(t, result, `[profile.default.treb.senders.deployer]`)
		assert.NotContains(t, result, `private_key`)
	})

	t.Run("removes multiple treb sections across profiles", func(t *testing.T) {
		input := `[profile.default]
src = "src"

[profile.default.treb.senders.anvil]
type = "private_key"
private_key = "0xkey"

[profile.default.treb.senders.governor]
type = "oz_governor"
governor = "0xGOV"

[profile.live.treb.senders.safe0]
type = "safe"
safe = "0xSAFE"
signer = "signer0"

[rpc_endpoints]
anvil = "http://localhost:8545"
`
		result := removeTrebFromFoundryToml(input)
		assert.Contains(t, result, `[profile.default]`)
		assert.Contains(t, result, `[rpc_endpoints]`)
		assert.NotContains(t, result, `[profile.default.treb.senders.anvil]`)
		assert.NotContains(t, result, `[profile.default.treb.senders.governor]`)
		assert.NotContains(t, result, `[profile.live.treb.senders.safe0]`)
		assert.NotContains(t, result, `oz_governor`)
	})

	t.Run("preserves comments outside treb sections", func(t *testing.T) {
		input := `# Main config
[profile.default]
src = "src"

[profile.default.treb.senders.deployer]
type = "private_key"
# private key comment
private_key = "0xkey"

[rpc_endpoints]
# RPC config
anvil = "http://localhost:8545"
`
		result := removeTrebFromFoundryToml(input)
		assert.Contains(t, result, "# Main config")
		assert.Contains(t, result, "# RPC config")
		assert.NotContains(t, result, "private key comment")
		assert.NotContains(t, result, "private_key")
	})

	t.Run("handles treb section at end of file", func(t *testing.T) {
		input := `[profile.default]
src = "src"

[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "0xkey"
`
		result := removeTrebFromFoundryToml(input)
		assert.Contains(t, result, `[profile.default]`)
		assert.Contains(t, result, `src = "src"`)
		assert.NotContains(t, result, "private_key")
	})

	t.Run("no treb sections returns content unchanged", func(t *testing.T) {
		input := `[profile.default]
src = "src"

[rpc_endpoints]
anvil = "http://localhost:8545"
`
		result := removeTrebFromFoundryToml(input)
		assert.Equal(t, input, result)
	})

	t.Run("preserves exact foundry.toml content from test fixture", func(t *testing.T) {
		input := `[profile.default]
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
type = "private_key" # anvil user 0
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

# Governor sender configuration - addresses are set after governance deployment
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
		result := removeTrebFromFoundryToml(input)

		// Should preserve all non-treb sections
		assert.Contains(t, result, `[profile.default]`)
		assert.Contains(t, result, `src = "src"`)
		assert.Contains(t, result, `[lint]`)
		assert.Contains(t, result, `[rpc_endpoints]`)
		assert.Contains(t, result, `[etherscan]`)

		// Should remove all treb sections
		assert.NotContains(t, result, `[profile.default.treb`)
		assert.NotContains(t, result, `[profile.live.treb`)
		assert.NotContains(t, result, "private_key")
		assert.NotContains(t, result, "oz_governor")
		assert.NotContains(t, result, "GOVERNOR_ADDRESS")
	})

	t.Run("removes bare treb section header", func(t *testing.T) {
		input := `[profile.default]
src = "src"

[profile.default.treb]
some_key = "value"

[rpc_endpoints]
anvil = "http://localhost:8545"
`
		result := removeTrebFromFoundryToml(input)
		assert.NotContains(t, result, "[profile.default.treb]")
		assert.NotContains(t, result, "some_key")
		assert.Contains(t, result, "[rpc_endpoints]")
	})
}

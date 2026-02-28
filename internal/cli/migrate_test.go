package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestGenerateTrebTomlV2(t *testing.T) {
	t.Run("single account and namespace", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"deployer-key": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"},
		}
		namespaces := map[string]nsInfo{
			"default": {profile: "default", roles: map[string]string{"deployer": "deployer-key"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)
		assert.Contains(t, content, "[accounts.deployer-key]")
		assert.Contains(t, content, `type = "private_key"`)
		assert.Contains(t, content, `private_key = "${DEPLOYER_PRIVATE_KEY}"`)
		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, `profile = "default"`)
		assert.Contains(t, content, `deployer = "deployer-key"`)
	})

	t.Run("multiple accounts with deduplication", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"deployer-key": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
			"safe-3d3378":  {Type: config.SenderTypeSafe, Safe: "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F", Signer: "deployer-key"},
		}
		namespaces := map[string]nsInfo{
			"default":    {profile: "default", roles: map[string]string{"deployer": "deployer-key"}},
			"production": {profile: "production", roles: map[string]string{"deployer": "safe-3d3378"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)

		// Accounts section
		assert.Contains(t, content, "[accounts.deployer-key]")
		assert.Contains(t, content, "[accounts.safe-3d3378]")
		assert.Contains(t, content, `safe = "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F"`)
		assert.Contains(t, content, `signer = "deployer-key"`)

		// Namespace section
		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, "[namespace.production]")
		assert.Contains(t, content, `profile = "production"`)
	})

	t.Run("default namespace comes first", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"key": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
		}
		namespaces := map[string]nsInfo{
			"production": {profile: "production", roles: map[string]string{"deployer": "key"}},
			"default":    {profile: "default", roles: map[string]string{"deployer": "key"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)

		// default should appear before production
		defaultIdx := indexOfSubstring(content, "[namespace.default]")
		prodIdx := indexOfSubstring(content, "[namespace.production]")
		assert.Less(t, defaultIdx, prodIdx, "default namespace should come before production")
	})

	t.Run("includes header comments", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"key": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
		}
		namespaces := map[string]nsInfo{
			"default": {profile: "default", roles: map[string]string{"deployer": "key"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)
		assert.Contains(t, content, "# treb.toml")
		assert.Contains(t, content, "Migrated from foundry.toml")
	})

	t.Run("all account fields rendered when set", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"full": {
				Type:           config.SenderTypeOZGovernor,
				Address:        "0x123",
				Governor:       "0xGOV",
				Timelock:       "0xTL",
				Proposer:       "signer",
				PrivateKey:     "${PK}",
				Safe:           "0xSAFE",
				Signer:         "signer1",
				DerivationPath: "m/44",
			},
		}
		namespaces := map[string]nsInfo{
			"default": {profile: "default", roles: map[string]string{"gov": "full"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)
		assert.Contains(t, content, `address = "0x123"`)
		assert.Contains(t, content, `governor = "0xGOV"`)
		assert.Contains(t, content, `timelock = "0xTL"`)
		assert.Contains(t, content, `proposer = "signer"`)
		assert.Contains(t, content, `private_key = "${PK}"`)
		assert.Contains(t, content, `safe = "0xSAFE"`)
		assert.Contains(t, content, `signer = "signer1"`)
		assert.Contains(t, content, `derivation_path = "m/44"`)
	})

	t.Run("omits empty account fields", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"key": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
		}
		namespaces := map[string]nsInfo{
			"default": {profile: "default", roles: map[string]string{"deployer": "key"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)
		assert.NotContains(t, content, "address =")
		assert.NotContains(t, content, "safe =")
		assert.NotContains(t, content, "signer =")
		assert.NotContains(t, content, "derivation_path =")
		assert.NotContains(t, content, "governor =")
		assert.NotContains(t, content, "timelock =")
		assert.NotContains(t, content, "proposer =")
	})

	t.Run("role mappings sorted alphabetically", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"key":  {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
			"key2": {Type: config.SenderTypeLedger, DerivationPath: "m/44"},
		}
		namespaces := map[string]nsInfo{
			"default": {profile: "default", roles: map[string]string{
				"deployer": "key",
				"admin":    "key2",
			}},
		}

		content := generateTrebTomlV2(accounts, namespaces)

		// admin should appear before deployer
		adminIdx := indexOfSubstring(content, `admin = "key2"`)
		deployerIdx := indexOfSubstring(content, `deployer = "key"`)
		assert.Less(t, adminIdx, deployerIdx, "admin role should come before deployer")
	})
}

func TestRunMigrate(t *testing.T) {
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

		err := runMigrate(cfg)
		require.NoError(t, err)

		// treb.toml should NOT be written
		_, err = os.Stat(filepath.Join(tmpDir, "treb.toml"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("errors when treb.toml already in v2 format", func(t *testing.T) {
		tmpDir := t.TempDir()
		v2Content := `[accounts.deployer]
type = "private_key"
private_key = "${PK}"

[namespace.default]
deployer = "deployer"
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "treb.toml"), []byte(v2Content), 0644))

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

		err := runMigrate(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already uses the new accounts/namespace format")
	})

	t.Run("non-interactive writes v2 treb.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrate(cfg)
		require.NoError(t, err)

		// treb.toml should be written with v2 format
		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "[accounts.")
		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, `profile = "default"`)
		// Should NOT contain v1 format
		assert.NotContains(t, content, "[ns.")
	})

	t.Run("deduplicates identical senders across profiles", func(t *testing.T) {
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
								"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrate(cfg)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)

		// Should have a single account (deduplicated)
		assert.Contains(t, content, "[accounts.")
		// Both namespaces should reference the same account
		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, "[namespace.production]")
	})

	t.Run("non-interactive overwrites existing v1 treb.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		v1Content := `[ns.default]
profile = "default"

[ns.default.senders.deployer]
type = "private_key"
private_key = "${PK}"
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "treb.toml"), []byte(v1Content), 0644))

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

		err := runMigrate(cfg)
		require.NoError(t, err)

		// Should be overwritten with v2 content
		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, "[accounts.")
		assert.Contains(t, content, "[namespace.default]")
		assert.NotContains(t, content, "[ns.")
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

		err := runMigrate(cfg)
		require.NoError(t, err)

		// foundry.toml should be untouched
		data, err := os.ReadFile(filepath.Join(tmpDir, "foundry.toml"))
		require.NoError(t, err)
		assert.Equal(t, foundryContent, string(data))
	})

	t.Run("safe senders with cross-references", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.RuntimeConfig{
			ProjectRoot: tmpDir,
			FoundryConfig: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"safe":   {Type: config.SenderTypeSafe, Safe: "0xABC", Signer: "signer"},
								"signer": {Type: config.SenderTypePrivateKey, PrivateKey: "${SIGNER_KEY}"},
							},
						},
					},
				},
			},
			NonInteractive: true,
		}

		err := runMigrate(cfg)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)

		// Both accounts should exist
		assert.Contains(t, content, "[accounts.")
		assert.Contains(t, content, `type = "safe"`)
		assert.Contains(t, content, `type = "private_key"`)
		assert.Contains(t, content, `safe = "0xABC"`)
	})

	t.Run("multiple profiles with different senders", func(t *testing.T) {
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

		err := runMigrate(cfg)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, "[namespace.production]")
		assert.Contains(t, content, `profile = "default"`)
		assert.Contains(t, content, `profile = "production"`)
	})
}

// indexOfSubstring returns the index of the first occurrence of substr in s, or -1 if not found.
func indexOfSubstring(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

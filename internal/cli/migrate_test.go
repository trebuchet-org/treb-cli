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
		assert.Contains(t, content, "[namespace.default.senders]")
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

	t.Run("quotes dot-separated namespace keys", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"key": {Type: config.SenderTypePrivateKey, PrivateKey: "${PK}"},
		}
		namespaces := map[string]nsInfo{
			"default":        {profile: "default", roles: map[string]string{"deployer": "key"}},
			"production":     {profile: "production", roles: map[string]string{"deployer": "key"}},
			"production.ntt": {profile: "production", roles: map[string]string{"deployer": "key"}},
		}

		content := generateTrebTomlV2(accounts, namespaces)

		// Simple namespace names should NOT be quoted
		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, "[namespace.default.senders]")
		assert.Contains(t, content, "[namespace.production]")
		assert.Contains(t, content, "[namespace.production.senders]")
		// Dot-separated namespace names MUST be quoted to avoid TOML nested table interpretation
		assert.Contains(t, content, `[namespace."production.ntt"]`)
		assert.Contains(t, content, `[namespace."production.ntt".senders]`)
		assert.NotContains(t, content, "[namespace.production.ntt]")
		assert.NotContains(t, content, "[namespace.production.ntt.senders]")
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
		assert.Contains(t, content, "[namespace.default.senders]")

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
		assert.Contains(t, content, "[namespace.default.senders]")
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

func TestValidateAccountName(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		usedNames := map[string]bool{}
		assert.Empty(t, validateAccountName("deployer", usedNames))
		assert.Empty(t, validateAccountName("deployer-key", usedNames))
		assert.Empty(t, validateAccountName("my_account", usedNames))
		assert.Empty(t, validateAccountName("key-2", usedNames))
		assert.Empty(t, validateAccountName("ABC123", usedNames))
	})

	t.Run("empty name rejected", func(t *testing.T) {
		msg := validateAccountName("", map[string]bool{})
		assert.Equal(t, "name cannot be empty", msg)
	})

	t.Run("invalid characters rejected", func(t *testing.T) {
		usedNames := map[string]bool{}
		assert.NotEmpty(t, validateAccountName("has space", usedNames))
		assert.NotEmpty(t, validateAccountName("has.dot", usedNames))
		assert.NotEmpty(t, validateAccountName("has/slash", usedNames))
		assert.NotEmpty(t, validateAccountName("has@at", usedNames))
		assert.NotEmpty(t, validateAccountName("has=equals", usedNames))
	})

	t.Run("duplicate name rejected", func(t *testing.T) {
		usedNames := map[string]bool{"deployer": true}
		msg := validateAccountName("deployer", usedNames)
		assert.Contains(t, msg, "already taken")
	})

	t.Run("non-duplicate passes", func(t *testing.T) {
		usedNames := map[string]bool{"deployer": true}
		assert.Empty(t, validateAccountName("deployer-2", usedNames))
	})
}

func TestFormatAccountSummary(t *testing.T) {
	tests := []struct {
		name     string
		acct     config.AccountConfig
		expected string
	}{
		{
			name:     "private key",
			acct:     config.AccountConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_KEY}"},
			expected: "private_key (${DEPLOYER_KEY})",
		},
		{
			name:     "safe",
			acct:     config.AccountConfig{Type: config.SenderTypeSafe, Safe: "0xABC123"},
			expected: "safe (0xABC123)",
		},
		{
			name:     "ledger with address",
			acct:     config.AccountConfig{Type: config.SenderTypeLedger, Address: "0xDEAD"},
			expected: "ledger (0xDEAD)",
		},
		{
			name:     "ledger with derivation path only",
			acct:     config.AccountConfig{Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
			expected: "ledger (path: m/44'/60'/0'/0/0)",
		},
		{
			name:     "trezor with address",
			acct:     config.AccountConfig{Type: config.SenderTypeTrezor, Address: "0xCAFE"},
			expected: "trezor (0xCAFE)",
		},
		{
			name:     "trezor with derivation path only",
			acct:     config.AccountConfig{Type: config.SenderTypeTrezor, DerivationPath: "m/44'/60'/0'/0/0"},
			expected: "trezor (path: m/44'/60'/0'/0/0)",
		},
		{
			name:     "oz governor",
			acct:     config.AccountConfig{Type: config.SenderTypeOZGovernor, Governor: "0xGOV"},
			expected: "oz_governor (0xGOV)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatAccountSummary(tt.acct))
		})
	}
}

func TestCountDeploymentsPerNamespace(t *testing.T) {
	t.Run("returns nil when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		counts, err := countDeploymentsPerNamespace(tmpDir)
		require.NoError(t, err)
		assert.Nil(t, counts)
	})

	t.Run("returns empty map for empty deployments", func(t *testing.T) {
		tmpDir := t.TempDir()
		trebDir := filepath.Join(tmpDir, ".treb")
		require.NoError(t, os.MkdirAll(trebDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(trebDir, "deployments.json"), []byte(`{}`), 0644))

		counts, err := countDeploymentsPerNamespace(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, map[string]int{}, counts)
	})

	t.Run("counts deployments per namespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		trebDir := filepath.Join(tmpDir, ".treb")
		require.NoError(t, os.MkdirAll(trebDir, 0755))

		deploymentsJSON := `{
			"default/31337/Counter": {"namespace": "default"},
			"default/31337/Token": {"namespace": "default"},
			"production/1/Counter": {"namespace": "production"},
			"staging/11155111/Counter": {"namespace": "staging"}
		}`
		require.NoError(t, os.WriteFile(filepath.Join(trebDir, "deployments.json"), []byte(deploymentsJSON), 0644))

		counts, err := countDeploymentsPerNamespace(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, 2, counts["default"])
		assert.Equal(t, 1, counts["production"])
		assert.Equal(t, 1, counts["staging"])
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		trebDir := filepath.Join(tmpDir, ".treb")
		require.NoError(t, os.MkdirAll(trebDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(trebDir, "deployments.json"), []byte(`not json`), 0644))

		_, err := countDeploymentsPerNamespace(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse deployments.json")
	})
}

func TestRunMigrateNamespacePruning(t *testing.T) {
	t.Run("non-interactive keeps all namespaces even with no deployments", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .treb/deployments.json with only default namespace having deployments
		trebDir := filepath.Join(tmpDir, ".treb")
		require.NoError(t, os.MkdirAll(trebDir, 0755))
		deploymentsJSON := `{
			"default/31337/Counter": {"namespace": "default"}
		}`
		require.NoError(t, os.WriteFile(filepath.Join(trebDir, "deployments.json"), []byte(deploymentsJSON), 0644))

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
					"staging": {
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

		// Non-interactive: both namespaces should be kept
		assert.Contains(t, content, "[namespace.default]")
		assert.Contains(t, content, "[namespace.staging]")
	})

	t.Run("non-interactive works when no deployments file exists", func(t *testing.T) {
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

		err := runMigrate(cfg)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "treb.toml"))
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "[namespace.default]")
	})
}

func TestPruneEmptyNamespaces(t *testing.T) {
	t.Run("keeps namespaces with deployments", func(t *testing.T) {
		namespaces := map[string]nsInfo{
			"default":    {profile: "default", roles: map[string]string{"deployer": "key"}},
			"production": {profile: "production", roles: map[string]string{"deployer": "key"}},
		}
		counts := map[string]int{
			"default":    3,
			"production": 1,
		}

		// No prompts should be triggered since all namespaces have deployments
		err := pruneEmptyNamespaces(namespaces, counts)
		require.NoError(t, err)
		assert.Len(t, namespaces, 2)
		assert.Contains(t, namespaces, "default")
		assert.Contains(t, namespaces, "production")
	})

	t.Run("does not prompt when all namespaces have deployments", func(t *testing.T) {
		namespaces := map[string]nsInfo{
			"default": {profile: "default", roles: map[string]string{"deployer": "key"}},
		}
		counts := map[string]int{
			"default": 5,
		}

		err := pruneEmptyNamespaces(namespaces, counts)
		require.NoError(t, err)
		assert.Len(t, namespaces, 1)
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

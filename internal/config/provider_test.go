package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestMergeTrebFileConfig(t *testing.T) {
	t.Run("default namespace only", func(t *testing.T) {
		trebFile := &config.TrebFileConfig{
			Ns: map[string]config.NamespaceConfig{
				"default": {
					Profile: "default",
					Senders: map[string]config.SenderConfig{
						"deployer": {Type: "private_key", PrivateKey: "0x1234"},
					},
				},
			},
		}

		merged, profile := mergeTrebFileConfig(trebFile, "default")

		require.NotNil(t, merged)
		assert.Equal(t, "default", profile)
		assert.Len(t, merged.Senders, 1)
		assert.Equal(t, config.SenderType("private_key"), merged.Senders["deployer"].Type)
	})

	t.Run("active namespace overlays default senders", func(t *testing.T) {
		trebFile := &config.TrebFileConfig{
			Ns: map[string]config.NamespaceConfig{
				"default": {
					Profile: "default",
					Senders: map[string]config.SenderConfig{
						"deployer": {Type: "private_key", PrivateKey: "0x1234"},
						"backup":   {Type: "private_key", PrivateKey: "0x5678"},
					},
				},
				"live": {
					Profile: "production",
					Senders: map[string]config.SenderConfig{
						"deployer": {Type: "ledger", DerivationPath: "m/44'/60'/0'/0/0"},
					},
				},
			},
		}

		merged, profile := mergeTrebFileConfig(trebFile, "live")

		require.NotNil(t, merged)
		assert.Equal(t, "production", profile)
		assert.Len(t, merged.Senders, 2)
		// deployer overridden by live namespace
		assert.Equal(t, config.SenderType("ledger"), merged.Senders["deployer"].Type)
		// backup inherited from default
		assert.Equal(t, config.SenderType("private_key"), merged.Senders["backup"].Type)
	})

	t.Run("profile defaults to namespace name", func(t *testing.T) {
		trebFile := &config.TrebFileConfig{
			Ns: map[string]config.NamespaceConfig{
				"staging": {
					Profile: "staging", // Set by loadTrebConfig default logic
					Senders: map[string]config.SenderConfig{
						"deployer": {Type: "private_key", PrivateKey: "0x1234"},
					},
				},
			},
		}

		_, profile := mergeTrebFileConfig(trebFile, "staging")
		assert.Equal(t, "staging", profile)
	})

	t.Run("namespace not in config uses default senders and namespace as profile", func(t *testing.T) {
		trebFile := &config.TrebFileConfig{
			Ns: map[string]config.NamespaceConfig{
				"default": {
					Profile: "default",
					Senders: map[string]config.SenderConfig{
						"deployer": {Type: "private_key", PrivateKey: "0x1234"},
					},
				},
			},
		}

		merged, profile := mergeTrebFileConfig(trebFile, "unknown")

		require.NotNil(t, merged)
		assert.Equal(t, "unknown", profile)
		assert.Len(t, merged.Senders, 1)
		assert.Equal(t, config.SenderType("private_key"), merged.Senders["deployer"].Type)
	})

	t.Run("empty config returns empty senders", func(t *testing.T) {
		trebFile := &config.TrebFileConfig{
			Ns: map[string]config.NamespaceConfig{},
		}

		merged, profile := mergeTrebFileConfig(trebFile, "default")
		require.NotNil(t, merged)
		assert.Equal(t, "default", profile)
		assert.Empty(t, merged.Senders)
	})
}

func TestMergeFoundryTrebConfig(t *testing.T) {
	t.Run("default profile only", func(t *testing.T) {
		foundryConfig := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"default": {
					Treb: &config.TrebConfig{
						Senders: map[string]config.SenderConfig{
							"deployer": {Type: "private_key", PrivateKey: "0x1234"},
						},
					},
				},
			},
		}

		merged := mergeFoundryTrebConfig(foundryConfig, "default")
		require.NotNil(t, merged)
		assert.Len(t, merged.Senders, 1)
		assert.Equal(t, config.SenderType("private_key"), merged.Senders["deployer"].Type)
	})

	t.Run("specific profile merges with default", func(t *testing.T) {
		foundryConfig := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"default": {
					Treb: &config.TrebConfig{
						Senders: map[string]config.SenderConfig{
							"deployer": {Type: "private_key", PrivateKey: "0x1234"},
							"backup":   {Type: "private_key", PrivateKey: "0x5678"},
						},
					},
				},
				"production": {
					Treb: &config.TrebConfig{
						Senders: map[string]config.SenderConfig{
							"deployer": {Type: "ledger", DerivationPath: "m/44'/60'/0'/0/0"},
						},
					},
				},
			},
		}

		merged := mergeFoundryTrebConfig(foundryConfig, "production")
		require.NotNil(t, merged)
		assert.Len(t, merged.Senders, 2)
		assert.Equal(t, config.SenderType("ledger"), merged.Senders["deployer"].Type)
		assert.Equal(t, config.SenderType("private_key"), merged.Senders["backup"].Type)
	})

	t.Run("no treb config returns nil", func(t *testing.T) {
		foundryConfig := &config.FoundryConfig{
			Profile: map[string]config.ProfileConfig{
				"default": {},
			},
		}

		merged := mergeFoundryTrebConfig(foundryConfig, "default")
		// Should have an empty senders map since default profile exists but Treb is nil
		require.NotNil(t, merged)
		assert.Empty(t, merged.Senders)
	})
}

func TestProviderConfigSource(t *testing.T) {
	t.Run("uses treb.toml when present", func(t *testing.T) {
		dir := t.TempDir()

		// Write minimal foundry.toml (always required)
		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		// Write treb.toml
		trebToml := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml", cfg.ConfigSource)
		assert.Equal(t, "default", cfg.FoundryProfile)
		require.NotNil(t, cfg.TrebConfig)
		assert.Equal(t, config.SenderType("private_key"), cfg.TrebConfig.Senders["deployer"].Type)
	})

	t.Run("falls back to foundry.toml when treb.toml absent", func(t *testing.T) {
		dir := t.TempDir()

		// Write foundry.toml with treb sender config
		foundryToml := `[profile.default]
src = "src"

[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "foundry.toml", cfg.ConfigSource)
		assert.Equal(t, "default", cfg.FoundryProfile)
		require.NotNil(t, cfg.TrebConfig)
		assert.Equal(t, config.SenderType("private_key"), cfg.TrebConfig.Senders["deployer"].Type)
	})

	t.Run("treb.toml profile field sets FoundryProfile", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"

[profile.production]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0x1234"

[ns.live]
profile = "production"

[ns.live.senders.deployer]
type = "ledger"
derivation_path = "m/44'/60'/0'/0/0"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "live")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml", cfg.ConfigSource)
		assert.Equal(t, "production", cfg.FoundryProfile, "should resolve profile from treb.toml")
		assert.Equal(t, config.SenderType("ledger"), cfg.TrebConfig.Senders["deployer"].Type)
	})

	t.Run("foundry.toml fallback sets FoundryProfile to namespace", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"

[profile.staging]
src = "src"

[profile.staging.treb.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "staging")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "foundry.toml", cfg.ConfigSource)
		assert.Equal(t, "staging", cfg.FoundryProfile, "FoundryProfile should equal namespace in legacy mode")
	})

	t.Run("uses treb.toml v2 format with accounts and namespaces", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[accounts.deployer]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

[namespace.default]
deployer = "deployer"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml (v2)", cfg.ConfigSource)
		assert.Equal(t, "default", cfg.FoundryProfile)
		require.NotNil(t, cfg.TrebConfig)
		assert.Equal(t, config.SenderType("private_key"), cfg.TrebConfig.Senders["deployer"].Type)
	})

	t.Run("v2 format resolves namespace profile", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"

[profile.mainnet]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[accounts.prod-deployer]
type = "ledger"
derivation_path = "m/44'/60'/0'/0/0"

[namespace.default]
deployer = "deployer"

[namespace.production]
profile = "mainnet"
deployer = "prod-deployer"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "production")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml (v2)", cfg.ConfigSource)
		assert.Equal(t, "mainnet", cfg.FoundryProfile, "should resolve profile from v2 namespace")
		assert.Equal(t, config.SenderType("ledger"), cfg.TrebConfig.Senders["deployer"].Type)
	})

	t.Run("v2 format reads fork.setup into ForkSetup", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[namespace.default]
deployer = "deployer"

[fork]
setup = "script/SetupFork.s.sol"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml (v2)", cfg.ConfigSource)
		assert.Equal(t, "script/SetupFork.s.sol", cfg.ForkSetup)
	})

	t.Run("v2 format defaults FoundryProfile to namespace when profile not set", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[namespace.default]
deployer = "deployer"

[namespace.staging]
deployer = "deployer"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "staging")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml (v2)", cfg.ConfigSource)
		assert.Equal(t, "staging", cfg.FoundryProfile, "should default to namespace name when profile not set")
	})

	t.Run("v1 treb.toml still works when v2 not detected", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml", cfg.ConfigSource)
		require.NotNil(t, cfg.TrebConfig)
		assert.Equal(t, config.SenderType("private_key"), cfg.TrebConfig.Senders["deployer"].Type)
	})

	t.Run("v1 format reads forkSetup from viper for backwards compat", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")
		// Simulate forkSetup from config.local.json (viper reads this field)
		v.Set("forksetup", "script/SetupFork.s.sol")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml", cfg.ConfigSource)
		assert.Equal(t, "script/SetupFork.s.sol", cfg.ForkSetup)
	})

	t.Run("legacy foundry.toml reads forkSetup from viper for backwards compat", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "default")
		v.Set("forksetup", "script/SetupFork.s.sol")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "foundry.toml", cfg.ConfigSource)
		assert.Equal(t, "script/SetupFork.s.sol", cfg.ForkSetup)
	})

	t.Run("v2 format with hierarchical namespace inheritance", func(t *testing.T) {
		dir := t.TempDir()

		foundryToml := `[profile.default]
src = "src"
`
		err := os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(foundryToml), 0644)
		require.NoError(t, err)

		trebToml := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[accounts.safe-wallet]
type = "safe"
safe = "0xDEAD"
signer = "deployer"

[namespace.default]
deployer = "deployer"

[namespace.production]
profile = "mainnet"

[namespace."production.ntt"]
deployer = "safe-wallet"
`
		err = os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		v := viper.New()
		v.Set("project_root", dir)
		v.Set("namespace", "production.ntt")

		cfg, err := Provider(v)
		require.NoError(t, err)

		assert.Equal(t, "treb.toml (v2)", cfg.ConfigSource)
		assert.Equal(t, "mainnet", cfg.FoundryProfile, "should inherit profile from production")
		assert.Equal(t, config.SenderType("safe"), cfg.TrebConfig.Senders["deployer"].Type, "deployer should be overridden by production.ntt")
	})
}

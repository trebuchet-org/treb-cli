package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestLoadTrebConfig(t *testing.T) {
	t.Run("parses valid treb.toml", func(t *testing.T) {
		dir := t.TempDir()
		trebToml := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

[ns.live]
profile = "production"

[ns.live.senders.safe0]
type = "safe"
safe = "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F"
signer = "deployer"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check default namespace
		defaultNs, ok := cfg.Ns["default"]
		require.True(t, ok)
		assert.Equal(t, config.SenderType("private_key"), defaultNs.Senders["deployer"].Type)
		assert.Equal(t, "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", defaultNs.Senders["deployer"].PrivateKey)

		// Check live namespace
		liveNs, ok := cfg.Ns["live"]
		require.True(t, ok)
		assert.Equal(t, "production", liveNs.Profile)
		assert.Equal(t, config.SenderType("safe"), liveNs.Senders["safe0"].Type)
		assert.Equal(t, "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F", liveNs.Senders["safe0"].Safe)
		assert.Equal(t, "deployer", liveNs.Senders["safe0"].Signer)
	})

	t.Run("profile defaults to namespace name", func(t *testing.T) {
		dir := t.TempDir()
		trebToml := `
[ns.staging.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		stagingNs := cfg.Ns["staging"]
		assert.Equal(t, "staging", stagingNs.Profile, "profile should default to namespace name")
	})

	t.Run("explicit profile overrides default", func(t *testing.T) {
		dir := t.TempDir()
		trebToml := `
[ns.live]
profile = "production"

[ns.live.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		liveNs := cfg.Ns["live"]
		assert.Equal(t, "production", liveNs.Profile, "explicit profile should be preserved")
	})

	t.Run("expands environment variables", func(t *testing.T) {
		dir := t.TempDir()
		trebToml := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "${TEST_TREB_PK}"

[ns.default.senders.safe0]
type = "safe"
safe = "${TEST_TREB_SAFE}"
signer = "deployer"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		t.Setenv("TEST_TREB_PK", "0xdeadbeef")
		t.Setenv("TEST_TREB_SAFE", "0x1111111111111111111111111111111111111111")

		cfg, err := loadTrebConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		deployer := cfg.Ns["default"].Senders["deployer"]
		assert.Equal(t, "0xdeadbeef", deployer.PrivateKey)

		safe0 := cfg.Ns["default"].Senders["safe0"]
		assert.Equal(t, "0x1111111111111111111111111111111111111111", safe0.Safe)
	})

	t.Run("missing file returns nil", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := loadTrebConfig(dir)
		assert.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("invalid TOML returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte("invalid [[ toml"), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfig(dir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse treb.toml")
	})

	t.Run("all sender config string fields are expanded", func(t *testing.T) {
		dir := t.TempDir()
		trebToml := `
[ns.default.senders.hw]
type = "ledger"
address = "${TEST_HW_ADDR}"
derivation_path = "${TEST_HW_PATH}"

[ns.default.senders.gov]
type = "oz_governor"
governor = "${TEST_GOV}"
timelock = "${TEST_TIMELOCK}"
proposer = "hw"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(trebToml), 0644)
		require.NoError(t, err)

		t.Setenv("TEST_HW_ADDR", "0x2222222222222222222222222222222222222222")
		t.Setenv("TEST_HW_PATH", "m/44'/60'/0'/0/0")
		t.Setenv("TEST_GOV", "0x3333333333333333333333333333333333333333")
		t.Setenv("TEST_TIMELOCK", "0x4444444444444444444444444444444444444444")

		cfg, err := loadTrebConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		hw := cfg.Ns["default"].Senders["hw"]
		assert.Equal(t, "0x2222222222222222222222222222222222222222", hw.Address)
		assert.Equal(t, "m/44'/60'/0'/0/0", hw.DerivationPath)

		gov := cfg.Ns["default"].Senders["gov"]
		assert.Equal(t, "0x3333333333333333333333333333333333333333", gov.Governor)
		assert.Equal(t, "0x4444444444444444444444444444444444444444", gov.Timelock)
	})
}

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestDetectTrebConfigFormat(t *testing.T) {
	t.Run("no treb.toml returns None", func(t *testing.T) {
		dir := t.TempDir()

		format, err := detectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatNone, format)
	})

	t.Run("v2 format with accounts section", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		format, err := detectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatV2, format)
	})

	t.Run("v2 format with namespace section", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[namespace.default]
deployer = "deployer"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		format, err := detectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatV2, format)
	})

	t.Run("v1 format with ns section", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		format, err := detectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatV1, format)
	})

	t.Run("empty treb.toml returns None", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(""), 0644)
		require.NoError(t, err)

		format, err := detectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatNone, format)
	})

	t.Run("invalid TOML returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte("invalid [[ toml"), 0644)
		require.NoError(t, err)

		_, err = detectTrebConfigFormat(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse treb.toml")
	})
}

func TestLoadTrebConfigV2(t *testing.T) {
	t.Run("parses basic v2 config", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[accounts.deployer]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

[accounts.safe0]
type = "safe"
safe = "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F"
signer = "deployer"

[namespace.default]
deployer = "deployer"

[namespace.production]
profile = "mainnet"
deployer = "safe0"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check accounts
		assert.Len(t, cfg.Accounts, 2)
		assert.Equal(t, config.SenderType("private_key"), cfg.Accounts["deployer"].Type)
		assert.Equal(t, "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", cfg.Accounts["deployer"].PrivateKey)
		assert.Equal(t, config.SenderType("safe"), cfg.Accounts["safe0"].Type)
		assert.Equal(t, "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F", cfg.Accounts["safe0"].Safe)
		assert.Equal(t, "deployer", cfg.Accounts["safe0"].Signer)

		// Check namespaces
		assert.Len(t, cfg.Namespace, 2)

		defaultNs := cfg.Namespace["default"]
		assert.Equal(t, "", defaultNs.Profile)
		assert.Equal(t, "deployer", defaultNs.Roles["deployer"])

		prodNs := cfg.Namespace["production"]
		assert.Equal(t, "mainnet", prodNs.Profile)
		assert.Equal(t, "safe0", prodNs.Roles["deployer"])
	})

	t.Run("expands environment variables in accounts", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[accounts.deployer]
type = "private_key"
private_key = "${TEST_V2_PK}"

[accounts.hw]
type = "ledger"
address = "${TEST_V2_ADDR}"
derivation_path = "${TEST_V2_PATH}"

[accounts.safe0]
type = "safe"
safe = "${TEST_V2_SAFE}"
signer = "hw"

[accounts.gov]
type = "oz_governor"
governor = "${TEST_V2_GOV}"
timelock = "${TEST_V2_TIMELOCK}"
proposer = "hw"

[namespace.default]
deployer = "deployer"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		t.Setenv("TEST_V2_PK", "0xdeadbeef")
		t.Setenv("TEST_V2_ADDR", "0x1111111111111111111111111111111111111111")
		t.Setenv("TEST_V2_PATH", "m/44'/60'/0'/0/0")
		t.Setenv("TEST_V2_SAFE", "0x2222222222222222222222222222222222222222")
		t.Setenv("TEST_V2_GOV", "0x3333333333333333333333333333333333333333")
		t.Setenv("TEST_V2_TIMELOCK", "0x4444444444444444444444444444444444444444")

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "0xdeadbeef", cfg.Accounts["deployer"].PrivateKey)
		assert.Equal(t, "0x1111111111111111111111111111111111111111", cfg.Accounts["hw"].Address)
		assert.Equal(t, "m/44'/60'/0'/0/0", cfg.Accounts["hw"].DerivationPath)
		assert.Equal(t, "0x2222222222222222222222222222222222222222", cfg.Accounts["safe0"].Safe)
		assert.Equal(t, "0x3333333333333333333333333333333333333333", cfg.Accounts["gov"].Governor)
		assert.Equal(t, "0x4444444444444444444444444444444444444444", cfg.Accounts["gov"].Timelock)
	})

	t.Run("handles quoted dot-namespace keys", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[namespace.default]
deployer = "deployer"

[namespace."production.ntt"]
profile = "production"
deployer = "deployer"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// The dot should be part of the key, not TOML nesting
		nttNs, ok := cfg.Namespace["production.ntt"]
		require.True(t, ok, "namespace 'production.ntt' should exist as a single key")
		assert.Equal(t, "production", nttNs.Profile)
		assert.Equal(t, "deployer", nttNs.Roles["deployer"])
	})

	t.Run("missing file returns nil", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := loadTrebConfigV2(dir)
		assert.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("v1 format returns nil", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[ns.default.senders.deployer]
type = "private_key"
private_key = "0x1234"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		assert.NoError(t, err)
		assert.Nil(t, cfg, "v1 format should return nil from v2 loader")
	})

	t.Run("parses fork config", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[namespace.default]
deployer = "deployer"

[fork]
setup = "script/ForkSetup.s.sol"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "script/ForkSetup.s.sol", cfg.Fork.Setup)
	})

	t.Run("invalid TOML returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte("invalid [[ toml"), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse treb.toml")
	})

	t.Run("multiple roles in namespace", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[accounts.deployer]
type = "private_key"
private_key = "0x1234"

[accounts.safe0]
type = "safe"
safe = "0xaaaa"
signer = "deployer"

[namespace.default]
deployer = "deployer"
admin = "safe0"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		defaultNs := cfg.Namespace["default"]
		assert.Equal(t, "deployer", defaultNs.Roles["deployer"])
		assert.Equal(t, "safe0", defaultNs.Roles["admin"])
		assert.Len(t, defaultNs.Roles, 2)
	})
}

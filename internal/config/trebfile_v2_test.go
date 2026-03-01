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

		format, err := DetectTrebConfigFormat(dir)
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

		format, err := DetectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatV2, format)
	})

	t.Run("v2 format with namespace section", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[namespace.default.senders]
deployer = "deployer"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		format, err := DetectTrebConfigFormat(dir)
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

		format, err := DetectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatV1, format)
	})

	t.Run("fork-only treb.toml returns V2", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[fork]
setup = "script/ForkSetup.s.sol"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		format, err := DetectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatV2, format)
	})

	t.Run("empty treb.toml returns None", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(""), 0644)
		require.NoError(t, err)

		format, err := DetectTrebConfigFormat(dir)
		require.NoError(t, err)
		assert.Equal(t, TrebConfigFormatNone, format)
	})

	t.Run("invalid TOML returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte("invalid [[ toml"), 0644)
		require.NoError(t, err)

		_, err = DetectTrebConfigFormat(dir)
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

[namespace.default.senders]
deployer = "deployer"

[namespace.production]
profile = "mainnet"

[namespace.production.senders]
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
		assert.Equal(t, "deployer", defaultNs.Senders["deployer"])

		prodNs := cfg.Namespace["production"]
		assert.Equal(t, "mainnet", prodNs.Profile)
		assert.Equal(t, "safe0", prodNs.Senders["deployer"])
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

[namespace.default.senders]
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

[namespace.default.senders]
deployer = "deployer"

[namespace."production.ntt"]
profile = "production"

[namespace."production.ntt".senders]
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
		assert.Equal(t, "deployer", nttNs.Senders["deployer"])
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

[namespace.default.senders]
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

	t.Run("fork-only config loads correctly", func(t *testing.T) {
		dir := t.TempDir()
		content := `
[fork]
setup = "script/ForkSetup.s.sol"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "script/ForkSetup.s.sol", cfg.Fork.Setup)
		assert.Empty(t, cfg.Accounts)
		assert.Empty(t, cfg.Namespace)
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

[namespace.default.senders]
deployer = "deployer"
admin = "safe0"
`
		err := os.WriteFile(filepath.Join(dir, "treb.toml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := loadTrebConfigV2(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		defaultNs := cfg.Namespace["default"]
		assert.Equal(t, "deployer", defaultNs.Senders["deployer"])
		assert.Equal(t, "safe0", defaultNs.Senders["admin"])
		assert.Len(t, defaultNs.Senders, 2)
	})
}

func TestResolveNamespace(t *testing.T) {
	t.Run("single-level default resolution", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"deployer": {Type: "private_key", PrivateKey: "0x1234"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Profile: "default",
					Senders:   map[string]string{"deployer": "deployer"},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "default")
		require.NoError(t, err)
		assert.Equal(t, "default", resolved.Profile)
		assert.Len(t, resolved.Accounts, 1)
		assert.Equal(t, config.SenderType("private_key"), resolved.Accounts["deployer"].Type)
		assert.Equal(t, "0x1234", resolved.Accounts["deployer"].PrivateKey)
	})

	t.Run("multi-level inheritance", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"dev-wallet":  {Type: "private_key", PrivateKey: "0xdev"},
				"prod-safe":   {Type: "safe", Safe: "0xsafe", Signer: "dev-wallet"},
				"ntt-deployer": {Type: "private_key", PrivateKey: "0xntt"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Senders: map[string]string{"deployer": "dev-wallet"},
				},
				"production": {
					Profile: "mainnet",
					Senders:   map[string]string{"deployer": "prod-safe"},
				},
				"production.ntt": {
					Senders: map[string]string{"deployer": "ntt-deployer"},
				},
			},
		}

		// Resolving "production.ntt" should walk: default → production → production.ntt
		resolved, err := ResolveNamespace(cfg, "production.ntt")
		require.NoError(t, err)

		// Profile inherited from production
		assert.Equal(t, "mainnet", resolved.Profile)
		// Deployer overridden at production.ntt level
		assert.Equal(t, config.SenderType("private_key"), resolved.Accounts["deployer"].Type)
		assert.Equal(t, "0xntt", resolved.Accounts["deployer"].PrivateKey)
	})

	t.Run("profile inheritance without override", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"deployer": {Type: "private_key", PrivateKey: "0x1234"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Senders: map[string]string{"deployer": "deployer"},
				},
				"production": {
					Profile: "mainnet",
					Senders:   map[string]string{},
				},
				"production.ntt": {
					// No profile set — should inherit "mainnet" from production
					Senders: map[string]string{},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "production.ntt")
		require.NoError(t, err)
		assert.Equal(t, "mainnet", resolved.Profile)
		// deployer inherited from default
		assert.Equal(t, "0x1234", resolved.Accounts["deployer"].PrivateKey)
	})

	t.Run("profile override at child level", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"deployer": {Type: "private_key", PrivateKey: "0x1234"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Senders: map[string]string{"deployer": "deployer"},
				},
				"production": {
					Profile: "mainnet",
					Senders:   map[string]string{},
				},
				"production.ntt": {
					Profile: "ntt-mainnet",
					Senders:   map[string]string{},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "production.ntt")
		require.NoError(t, err)
		assert.Equal(t, "ntt-mainnet", resolved.Profile)
	})

	t.Run("undefined parent is skipped", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"deployer":     {Type: "private_key", PrivateKey: "0xdefault"},
				"ntt-deployer": {Type: "private_key", PrivateKey: "0xntt"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Senders: map[string]string{"deployer": "deployer"},
				},
				// "production" is NOT defined — should be skipped
				"production.ntt": {
					Profile: "mainnet",
					Senders:   map[string]string{"deployer": "ntt-deployer"},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "production.ntt")
		require.NoError(t, err)
		assert.Equal(t, "mainnet", resolved.Profile)
		assert.Equal(t, "0xntt", resolved.Accounts["deployer"].PrivateKey)
	})

	t.Run("unknown account returns error", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"deployer": {Type: "private_key", PrivateKey: "0x1234"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Senders: map[string]string{"deployer": "nonexistent"},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "default")
		assert.Error(t, err)
		assert.Nil(t, resolved)
		assert.Contains(t, err.Error(), "unknown account \"nonexistent\"")
	})

	t.Run("default-only resolution with undefined default", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"deployer": {Type: "private_key", PrivateKey: "0x1234"},
			},
			Namespace: map[string]config.NamespaceRoles{
				// No default namespace defined
				"staging": {
					Profile: "staging",
					Senders:   map[string]string{"deployer": "deployer"},
				},
			},
		}

		// Resolving "default" when it doesn't exist should return empty
		resolved, err := ResolveNamespace(cfg, "default")
		require.NoError(t, err)
		assert.Equal(t, "", resolved.Profile)
		assert.Empty(t, resolved.Accounts)
	})

	t.Run("roles accumulate across levels", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"dev-wallet": {Type: "private_key", PrivateKey: "0xdev"},
				"prod-safe":  {Type: "safe", Safe: "0xsafe"},
				"admin":      {Type: "ledger", DerivationPath: "m/44'/60'/0'/0/0"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Senders: map[string]string{"deployer": "dev-wallet"},
				},
				"production": {
					Profile: "mainnet",
					Senders:   map[string]string{"deployer": "prod-safe", "admin": "admin"},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "production")
		require.NoError(t, err)
		assert.Equal(t, "mainnet", resolved.Profile)
		assert.Len(t, resolved.Accounts, 2)
		// deployer overridden from default
		assert.Equal(t, config.SenderType("safe"), resolved.Accounts["deployer"].Type)
		// admin added at production level
		assert.Equal(t, config.SenderType("ledger"), resolved.Accounts["admin"].Type)
	})

	t.Run("three-level deep resolution", func(t *testing.T) {
		cfg := &config.TrebFileConfigV2{
			Accounts: map[string]config.AccountConfig{
				"dev":   {Type: "private_key", PrivateKey: "0xdev"},
				"prod":  {Type: "private_key", PrivateKey: "0xprod"},
				"ntt":   {Type: "private_key", PrivateKey: "0xntt"},
				"v2":    {Type: "private_key", PrivateKey: "0xv2"},
			},
			Namespace: map[string]config.NamespaceRoles{
				"default": {
					Profile: "default",
					Senders:   map[string]string{"deployer": "dev", "monitor": "dev"},
				},
				"production": {
					Profile: "mainnet",
					Senders:   map[string]string{"deployer": "prod"},
				},
				"production.ntt": {
					Senders: map[string]string{"deployer": "ntt"},
				},
				"production.ntt.v2": {
					Senders: map[string]string{"deployer": "v2"},
				},
			},
		}

		resolved, err := ResolveNamespace(cfg, "production.ntt.v2")
		require.NoError(t, err)
		assert.Equal(t, "mainnet", resolved.Profile) // inherited from production
		assert.Equal(t, "0xv2", resolved.Accounts["deployer"].PrivateKey) // overridden at deepest level
		assert.Equal(t, "0xdev", resolved.Accounts["monitor"].PrivateKey) // inherited from default
	})
}

func TestResolvedNamespaceToTrebConfig(t *testing.T) {
	t.Run("private_key account mapping", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "0x1234"},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "default",
			Accounts: map[string]config.AccountConfig{
				"deployer": accounts["deployer"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		require.NoError(t, err)
		require.NotNil(t, trebCfg)

		assert.Len(t, trebCfg.Senders, 1)
		sender := trebCfg.Senders["deployer"]
		assert.Equal(t, config.SenderTypePrivateKey, sender.Type)
		assert.Equal(t, "0x1234", sender.PrivateKey)
	})

	t.Run("safe account with signer resolution", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"dev-wallet": {Type: config.SenderTypePrivateKey, PrivateKey: "0xdev"},
			"safe0":      {Type: config.SenderTypeSafe, Safe: "0xSafeAddr", Signer: "dev-wallet"},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "production",
			Accounts: map[string]config.AccountConfig{
				"deployer": accounts["safe0"],
				"proposer": accounts["dev-wallet"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		require.NoError(t, err)
		require.NotNil(t, trebCfg)

		assert.Len(t, trebCfg.Senders, 2)

		// Safe sender should have signer set to the account name
		safeSender := trebCfg.Senders["deployer"]
		assert.Equal(t, config.SenderTypeSafe, safeSender.Type)
		assert.Equal(t, "0xSafeAddr", safeSender.Safe)
		assert.Equal(t, "dev-wallet", safeSender.Signer)

		// Signer account should be present as a sender
		signerSender := trebCfg.Senders["proposer"]
		assert.Equal(t, config.SenderTypePrivateKey, signerSender.Type)
		assert.Equal(t, "0xdev", signerSender.PrivateKey)
	})

	t.Run("oz_governor with proposer", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"hw-wallet": {Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
			"gov":       {Type: config.SenderTypeOZGovernor, Governor: "0xGovAddr", Timelock: "0xTimelockAddr", Proposer: "hw-wallet"},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "production",
			Accounts: map[string]config.AccountConfig{
				"governor": accounts["gov"],
				"proposer": accounts["hw-wallet"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		require.NoError(t, err)
		require.NotNil(t, trebCfg)

		assert.Len(t, trebCfg.Senders, 2)

		govSender := trebCfg.Senders["governor"]
		assert.Equal(t, config.SenderTypeOZGovernor, govSender.Type)
		assert.Equal(t, "0xGovAddr", govSender.Governor)
		assert.Equal(t, "0xTimelockAddr", govSender.Timelock)
		assert.Equal(t, "hw-wallet", govSender.Proposer)

		proposerSender := trebCfg.Senders["proposer"]
		assert.Equal(t, config.SenderTypeLedger, proposerSender.Type)
		assert.Equal(t, "m/44'/60'/0'/0/0", proposerSender.DerivationPath)
	})

	t.Run("missing signer account returns error", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"safe0": {Type: config.SenderTypeSafe, Safe: "0xSafeAddr", Signer: "nonexistent"},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "default",
			Accounts: map[string]config.AccountConfig{
				"deployer": accounts["safe0"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		assert.Error(t, err)
		assert.Nil(t, trebCfg)
		assert.Contains(t, err.Error(), "unknown signer account \"nonexistent\"")
	})

	t.Run("missing proposer account returns error", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"gov": {Type: config.SenderTypeOZGovernor, Governor: "0xGovAddr", Proposer: "nonexistent"},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "default",
			Accounts: map[string]config.AccountConfig{
				"governor": accounts["gov"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		assert.Error(t, err)
		assert.Nil(t, trebCfg)
		assert.Contains(t, err.Error(), "unknown proposer account \"nonexistent\"")
	})

	t.Run("all account fields are mapped to sender config", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"full": {
				Type:           config.SenderTypeLedger,
				Address:        "0xAddr",
				DerivationPath: "m/44'/60'/0'/0/0",
			},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "default",
			Accounts: map[string]config.AccountConfig{
				"hw": accounts["full"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		require.NoError(t, err)

		sender := trebCfg.Senders["hw"]
		assert.Equal(t, config.SenderTypeLedger, sender.Type)
		assert.Equal(t, "0xAddr", sender.Address)
		assert.Equal(t, "m/44'/60'/0'/0/0", sender.DerivationPath)
	})

	t.Run("empty resolved namespace produces empty senders", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "0x1234"},
		}
		resolved := &config.ResolvedNamespace{
			Profile:  "default",
			Accounts: map[string]config.AccountConfig{},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		require.NoError(t, err)
		assert.Empty(t, trebCfg.Senders)
	})

	t.Run("multiple roles with mixed types", func(t *testing.T) {
		accounts := map[string]config.AccountConfig{
			"pk":     {Type: config.SenderTypePrivateKey, PrivateKey: "0x1234"},
			"ledger": {Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
			"safe0":  {Type: config.SenderTypeSafe, Safe: "0xSafe", Signer: "ledger"},
		}
		resolved := &config.ResolvedNamespace{
			Profile: "mainnet",
			Accounts: map[string]config.AccountConfig{
				"deployer": accounts["pk"],
				"admin":    accounts["ledger"],
				"multisig": accounts["safe0"],
			},
		}

		trebCfg, err := ResolvedNamespaceToTrebConfig(resolved, accounts)
		require.NoError(t, err)
		assert.Len(t, trebCfg.Senders, 3)
		assert.Equal(t, config.SenderTypePrivateKey, trebCfg.Senders["deployer"].Type)
		assert.Equal(t, config.SenderTypeLedger, trebCfg.Senders["admin"].Type)
		assert.Equal(t, config.SenderTypeSafe, trebCfg.Senders["multisig"].Type)
		assert.Equal(t, "ledger", trebCfg.Senders["multisig"].Signer)
	})
}

func TestBuildNamespaceChain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"default", "default", []string{"default"}},
		{"single level", "production", []string{"default", "production"}},
		{"two levels", "production.ntt", []string{"default", "production", "production.ntt"}},
		{"three levels", "production.ntt.v2", []string{"default", "production", "production.ntt", "production.ntt.v2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildNamespaceChain(tt.input))
		})
	}
}

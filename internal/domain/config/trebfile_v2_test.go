package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountConfig(t *testing.T) {
	t.Run("has same fields as SenderConfig", func(t *testing.T) {
		acc := AccountConfig{
			Type:           SenderTypePrivateKey,
			Address:        "0x1234",
			PrivateKey:     "0xkey",
			Safe:           "0xsafe",
			Signer:         "signer-account",
			DerivationPath: "m/44'/60'/0'/0/0",
			Governor:       "0xgov",
			Timelock:       "0xtime",
			Proposer:       "proposer-account",
		}
		assert.Equal(t, SenderTypePrivateKey, acc.Type)
		assert.Equal(t, "0x1234", acc.Address)
		assert.Equal(t, "0xkey", acc.PrivateKey)
		assert.Equal(t, "0xsafe", acc.Safe)
		assert.Equal(t, "signer-account", acc.Signer)
		assert.Equal(t, "m/44'/60'/0'/0/0", acc.DerivationPath)
		assert.Equal(t, "0xgov", acc.Governor)
		assert.Equal(t, "0xtime", acc.Timelock)
		assert.Equal(t, "proposer-account", acc.Proposer)
	})

	t.Run("supports all sender types", func(t *testing.T) {
		types := []SenderType{
			SenderTypePrivateKey,
			SenderTypeLedger,
			SenderTypeTrezor,
			SenderTypeSafe,
			SenderTypeOZGovernor,
		}
		for _, st := range types {
			acc := AccountConfig{Type: st}
			assert.Equal(t, st, acc.Type)
		}
	})
}

func TestNamespaceRoles(t *testing.T) {
	t.Run("stores profile and role mappings", func(t *testing.T) {
		ns := NamespaceRoles{
			Profile: "production",
			Roles: map[string]string{
				"deployer": "prod-deployer",
				"safe":     "prod-safe",
			},
		}
		assert.Equal(t, "production", ns.Profile)
		assert.Equal(t, "prod-deployer", ns.Roles["deployer"])
		assert.Equal(t, "prod-safe", ns.Roles["safe"])
	})

	t.Run("roles map can be empty", func(t *testing.T) {
		ns := NamespaceRoles{
			Profile: "staging",
			Roles:   map[string]string{},
		}
		assert.Equal(t, "staging", ns.Profile)
		assert.Empty(t, ns.Roles)
	})
}

func TestTrebFileConfigV2(t *testing.T) {
	t.Run("holds accounts, namespaces, and fork config", func(t *testing.T) {
		cfg := TrebFileConfigV2{
			Accounts: map[string]AccountConfig{
				"deployer": {Type: SenderTypePrivateKey, PrivateKey: "0xkey"},
				"hw":       {Type: SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"},
			},
			Namespace: map[string]NamespaceRoles{
				"default": {
					Profile: "default",
					Roles:   map[string]string{"deployer": "deployer"},
				},
				"production": {
					Profile: "mainnet",
					Roles:   map[string]string{"deployer": "hw"},
				},
			},
			Fork: ForkConfig{Setup: "script/ForkSetup.s.sol"},
		}

		assert.Len(t, cfg.Accounts, 2)
		assert.Equal(t, SenderTypePrivateKey, cfg.Accounts["deployer"].Type)
		assert.Equal(t, SenderTypeLedger, cfg.Accounts["hw"].Type)

		assert.Len(t, cfg.Namespace, 2)
		assert.Equal(t, "default", cfg.Namespace["default"].Profile)
		assert.Equal(t, "deployer", cfg.Namespace["default"].Roles["deployer"])
		assert.Equal(t, "mainnet", cfg.Namespace["production"].Profile)
		assert.Equal(t, "hw", cfg.Namespace["production"].Roles["deployer"])

		assert.Equal(t, "script/ForkSetup.s.sol", cfg.Fork.Setup)
	})
}

func TestForkConfig(t *testing.T) {
	t.Run("holds setup path", func(t *testing.T) {
		fc := ForkConfig{Setup: "script/ForkSetup.s.sol"}
		assert.Equal(t, "script/ForkSetup.s.sol", fc.Setup)
	})

	t.Run("setup can be empty", func(t *testing.T) {
		fc := ForkConfig{}
		assert.Empty(t, fc.Setup)
	})
}

func TestResolvedNamespace(t *testing.T) {
	t.Run("holds resolved profile and account mappings", func(t *testing.T) {
		resolved := ResolvedNamespace{
			Profile: "mainnet",
			Accounts: map[string]AccountConfig{
				"deployer": {Type: SenderTypePrivateKey, PrivateKey: "0xkey"},
				"safe":     {Type: SenderTypeSafe, Safe: "0xsafe", Signer: "deployer"},
			},
		}
		assert.Equal(t, "mainnet", resolved.Profile)
		assert.Len(t, resolved.Accounts, 2)
		assert.Equal(t, SenderTypePrivateKey, resolved.Accounts["deployer"].Type)
		assert.Equal(t, SenderTypeSafe, resolved.Accounts["safe"].Type)
		assert.Equal(t, "deployer", resolved.Accounts["safe"].Signer)
	})

	t.Run("accounts map can be empty", func(t *testing.T) {
		resolved := ResolvedNamespace{
			Profile:  "default",
			Accounts: map[string]AccountConfig{},
		}
		assert.Equal(t, "default", resolved.Profile)
		assert.Empty(t, resolved.Accounts)
	})
}

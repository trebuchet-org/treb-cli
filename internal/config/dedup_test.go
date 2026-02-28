package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestDeduplicateSenders(t *testing.T) {
	t.Run("basic dedup of identical private keys", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"},
			},
			"production": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		// Should produce only 1 unique account
		assert.Len(t, accounts, 1)

		// Both namespaces should reference the same account
		assert.Equal(t, mappings["default"]["deployer"], mappings["production"]["deployer"])

		// Account name derived from env var
		accountName := mappings["default"]["deployer"]
		assert.Equal(t, "deployer-key", accountName)
		assert.Equal(t, config.SenderTypePrivateKey, accounts[accountName].Type)
		assert.Equal(t, "${DEPLOYER_PRIVATE_KEY}", accounts[accountName].PrivateKey)
	})

	t.Run("safe dedup across namespaces", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"deployer": {Type: config.SenderTypeSafe, Safe: "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F", Signer: "proposer"},
				"proposer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PROPOSER_KEY}"},
			},
			"staging": {
				"deployer": {Type: config.SenderTypeSafe, Safe: "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F", Signer: "proposer"},
				"proposer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PROPOSER_KEY}"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		assert.Len(t, accounts, 2) // one safe + one private key
		assert.Equal(t, mappings["default"]["deployer"], mappings["staging"]["deployer"])
		assert.Equal(t, mappings["default"]["proposer"], mappings["staging"]["proposer"])
	})

	t.Run("no dedup when configs differ", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEV_KEY}"},
			},
			"production": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${PROD_KEY}"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		assert.Len(t, accounts, 2)
		assert.NotEqual(t, mappings["default"]["deployer"], mappings["production"]["deployer"])
	})

	t.Run("name collision handling", func(t *testing.T) {
		// Both env vars generate the same base name "deployer-key":
		// ${DEPLOYER_PRIVATE_KEY} → remove PRIVATE_ → DEPLOYER_KEY → deployer-key
		// ${DEPLOYER_KEY}         →                    DEPLOYER_KEY → deployer-key
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"},
			},
			"production": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_KEY}"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		assert.Len(t, accounts, 2)
		name1 := mappings["default"]["deployer"]
		name2 := mappings["production"]["deployer"]
		assert.NotEqual(t, name1, name2)

		// First processed gets base name, second gets suffix
		assert.Equal(t, "deployer-key", name1)
		assert.Equal(t, "deployer-key-2", name2)
	})

	t.Run("mixed sender types", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_KEY}"},
			},
			"production": {
				"deployer": {Type: config.SenderTypeSafe, Safe: "0xAABBCCDDEEFF1234567890", Signer: "proposer"},
				"proposer": {Type: config.SenderTypeLedger, Address: "0x1111222233334444555566667777888899990000", DerivationPath: "m/44'/60'/0'/0/0"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		assert.Len(t, accounts, 3) // private_key + safe + ledger

		// All three should have different account names
		names := make(map[string]bool)
		for _, m := range mappings {
			for _, name := range m {
				names[name] = true
			}
		}
		assert.Len(t, names, 3)
	})

	t.Run("safe signer reference updated to account name", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"proposer": {Type: config.SenderTypeLedger, Address: "0x1111222233334444", DerivationPath: "m/44'/60'/0'/0/0"},
				"deployer": {Type: config.SenderTypeSafe, Safe: "0xAABBCC112233", Signer: "proposer"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		assert.Len(t, accounts, 2)
		safeAccountName := mappings["default"]["deployer"]
		proposerAccountName := mappings["default"]["proposer"]

		// Safe account's signer should reference the new account name, not the old sender name
		assert.Equal(t, proposerAccountName, accounts[safeAccountName].Signer)
	})

	t.Run("oz_governor proposer reference updated to account name", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"hw-wallet": {Type: config.SenderTypeLedger, Address: "0x1111222233334444", DerivationPath: "m/44'/60'/0'/0/0"},
				"governor":  {Type: config.SenderTypeOZGovernor, Governor: "0xGOVERNOR11223344", Timelock: "0xTIMELOCK", Proposer: "hw-wallet"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		assert.Len(t, accounts, 2)
		govAccountName := mappings["default"]["governor"]
		hwAccountName := mappings["default"]["hw-wallet"]

		// Governor's proposer should reference the new account name
		assert.Equal(t, hwAccountName, accounts[govAccountName].Proposer)
	})

	t.Run("empty input returns empty maps", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{}

		accounts, mappings := DeduplicateSenders(input)

		assert.Empty(t, accounts)
		assert.Empty(t, mappings)
	})

	t.Run("single namespace single sender", func(t *testing.T) {
		input := map[string]map[string]config.SenderConfig{
			"default": {
				"deployer": {Type: config.SenderTypePrivateKey, PrivateKey: "${MY_KEY}"},
			},
		}

		accounts, mappings := DeduplicateSenders(input)

		require.Len(t, accounts, 1)
		assert.Equal(t, "my-key", mappings["default"]["deployer"])
		assert.Equal(t, config.SenderTypePrivateKey, accounts["my-key"].Type)
	})
}

func TestSenderFingerprint(t *testing.T) {
	t.Run("same private key same fingerprint", func(t *testing.T) {
		s1 := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${KEY}"}
		s2 := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${KEY}"}
		assert.Equal(t, senderFingerprint(s1), senderFingerprint(s2))
	})

	t.Run("different private key different fingerprint", func(t *testing.T) {
		s1 := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${KEY1}"}
		s2 := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${KEY2}"}
		assert.NotEqual(t, senderFingerprint(s1), senderFingerprint(s2))
	})

	t.Run("safe fingerprint includes signer", func(t *testing.T) {
		s1 := config.SenderConfig{Type: config.SenderTypeSafe, Safe: "0xSafe", Signer: "a"}
		s2 := config.SenderConfig{Type: config.SenderTypeSafe, Safe: "0xSafe", Signer: "b"}
		assert.NotEqual(t, senderFingerprint(s1), senderFingerprint(s2))
	})

	t.Run("governor fingerprint includes all three fields", func(t *testing.T) {
		s1 := config.SenderConfig{Type: config.SenderTypeOZGovernor, Governor: "0xGov", Timelock: "0xTL1", Proposer: "p"}
		s2 := config.SenderConfig{Type: config.SenderTypeOZGovernor, Governor: "0xGov", Timelock: "0xTL2", Proposer: "p"}
		assert.NotEqual(t, senderFingerprint(s1), senderFingerprint(s2))
	})

	t.Run("ledger fingerprint includes address and derivation path", func(t *testing.T) {
		s1 := config.SenderConfig{Type: config.SenderTypeLedger, Address: "0x1111", DerivationPath: "m/44'/60'/0'/0/0"}
		s2 := config.SenderConfig{Type: config.SenderTypeLedger, Address: "0x1111", DerivationPath: "m/44'/60'/0'/0/1"}
		assert.NotEqual(t, senderFingerprint(s1), senderFingerprint(s2))
	})

	t.Run("different types different fingerprint", func(t *testing.T) {
		s1 := config.SenderConfig{Type: config.SenderTypeLedger, Address: "0x1111", DerivationPath: "m/44'/60'/0'/0/0"}
		s2 := config.SenderConfig{Type: config.SenderTypeTrezor, Address: "0x1111", DerivationPath: "m/44'/60'/0'/0/0"}
		assert.NotEqual(t, senderFingerprint(s1), senderFingerprint(s2))
	})
}

func TestGenerateAccountName(t *testing.T) {
	t.Run("private key from env var", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_PRIVATE_KEY}"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "deployer-key", name)
	})

	t.Run("safe with address", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypeSafe, Safe: "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "safe-32cb58", name)
	})

	t.Run("governor with address", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypeOZGovernor, Governor: "0xAABBCC112233"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "governor-aabbcc", name)
	})

	t.Run("ledger with address", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypeLedger, Address: "0xDEADBEEF1234"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "ledger-deadbe", name)
	})

	t.Run("trezor with address", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypeTrezor, Address: "0xCAFEBABE5678"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "trezor-cafeba", name)
	})

	t.Run("ledger without address", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypeLedger, DerivationPath: "m/44'/60'/0'/0/0"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "ledger", name)
	})

	t.Run("collision appends suffix", func(t *testing.T) {
		usedNames := map[string]bool{"deployer-key": true}
		s := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_KEY}"}
		name := generateAccountName(s, usedNames)
		assert.Equal(t, "deployer-key-2", name)
	})

	t.Run("multiple collisions increment suffix", func(t *testing.T) {
		usedNames := map[string]bool{"deployer-key": true, "deployer-key-2": true}
		s := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "${DEPLOYER_KEY}"}
		name := generateAccountName(s, usedNames)
		assert.Equal(t, "deployer-key-3", name)
	})

	t.Run("literal key gets generic name", func(t *testing.T) {
		s := config.SenderConfig{Type: config.SenderTypePrivateKey, PrivateKey: "0xdeadbeef"}
		name := generateAccountName(s, map[string]bool{})
		assert.Equal(t, "private-key", name)
	})
}

func TestNameFromPrivateKeyRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"env var with PRIVATE_KEY suffix", "${DEPLOYER_PRIVATE_KEY}", "deployer-key"},
		{"env var with KEY suffix", "${DEPLOYER_KEY}", "deployer-key"},
		{"env var with multi-word prefix", "${MY_WALLET_PRIVATE_KEY}", "my-wallet-key"},
		{"dollar without braces", "$DEPLOYER_KEY", "deployer-key"},
		{"literal hex value", "0xdeadbeef", "private-key"},
		{"bare PRIVATE_KEY", "${PRIVATE_KEY}", "key"},
		{"simple key name", "${MY_KEY}", "my-key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nameFromPrivateKeyRef(tt.input))
		})
	}
}

func TestHexPrefix(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		n        int
		expected string
	}{
		{"standard address", "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F", 6, "32cb58"},
		{"no 0x prefix", "AABBCC112233", 6, "aabbcc"},
		{"short address", "0xAB", 6, "ab"},
		{"empty address", "", 6, ""},
		{"exact length", "0xAABBCC", 6, "aabbcc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hexPrefix(tt.addr, tt.n))
		})
	}
}

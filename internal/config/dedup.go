package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// DeduplicateSenders takes sender configs across namespaces, identifies identical configs
// (same type + same relevant fields), and merges them into unique named accounts.
// Returns the deduplicated accounts and namespace role mappings referencing the new account names.
func DeduplicateSenders(
	namespaceSenders map[string]map[string]config.SenderConfig,
) (map[string]config.AccountConfig, map[string]map[string]string) {
	type origin struct {
		namespace  string
		senderName string
	}

	fingerprintToName := make(map[string]string)
	usedNames := make(map[string]bool)
	accounts := make(map[string]config.AccountConfig)
	namespaceMappings := make(map[string]map[string]string)
	origins := make(map[string]origin) // account name → first origin

	// Phase 1: Fingerprint senders, assign unique account names.
	// Process in sorted order for deterministic output.
	nsNames := sortedMapKeys(namespaceSenders)

	for _, nsName := range nsNames {
		senders := namespaceSenders[nsName]
		namespaceMappings[nsName] = make(map[string]string)

		senderNames := sortedMapKeys(senders)

		for _, senderName := range senderNames {
			sender := senders[senderName]
			fp := senderFingerprint(sender)

			if accountName, exists := fingerprintToName[fp]; exists {
				namespaceMappings[nsName][senderName] = accountName
			} else {
				accountName := generateAccountName(sender, usedNames)
				usedNames[accountName] = true
				fingerprintToName[fp] = accountName
				accounts[accountName] = config.AccountConfig(sender)
				namespaceMappings[nsName][senderName] = accountName
				origins[accountName] = origin{namespace: nsName, senderName: senderName}
			}
		}
	}

	// Phase 2: Update signer/proposer cross-references to use new account names.
	// In the old format these reference sender names within the same profile/namespace;
	// in the new format they must reference account names.
	for accountName, acct := range accounts {
		orig := origins[accountName]
		nsMappings := namespaceMappings[orig.namespace]
		updated := false

		if acct.Signer != "" {
			if newName, exists := nsMappings[acct.Signer]; exists {
				acct.Signer = newName
				updated = true
			}
		}
		if acct.Proposer != "" {
			if newName, exists := nsMappings[acct.Proposer]; exists {
				acct.Proposer = newName
				updated = true
			}
		}

		if updated {
			accounts[accountName] = acct
		}
	}

	return accounts, namespaceMappings
}

// senderFingerprint generates a unique string for a SenderConfig based on its
// type and relevant identifying fields.
func senderFingerprint(s config.SenderConfig) string {
	switch s.Type {
	case config.SenderTypePrivateKey:
		return fmt.Sprintf("private_key:%s", s.PrivateKey)
	case config.SenderTypeSafe:
		return fmt.Sprintf("safe:%s:%s", s.Safe, s.Signer)
	case config.SenderTypeOZGovernor:
		return fmt.Sprintf("oz_governor:%s:%s:%s", s.Governor, s.Timelock, s.Proposer)
	case config.SenderTypeLedger:
		return fmt.Sprintf("ledger:%s:%s", s.Address, s.DerivationPath)
	case config.SenderTypeTrezor:
		return fmt.Sprintf("trezor:%s:%s", s.Address, s.DerivationPath)
	default:
		return fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s",
			s.Type, s.PrivateKey, s.Safe, s.Signer, s.Address, s.DerivationPath,
			s.Governor, s.Timelock, s.Proposer)
	}
}

// generateAccountName creates a human-readable account name from the sender's
// type and key material. Handles collisions by appending a numeric suffix.
func generateAccountName(s config.SenderConfig, usedNames map[string]bool) string {
	var base string
	switch s.Type {
	case config.SenderTypePrivateKey:
		base = nameFromPrivateKeyRef(s.PrivateKey)
	case config.SenderTypeSafe:
		if prefix := hexPrefix(s.Safe, 6); prefix != "" {
			base = "safe-" + prefix
		} else {
			base = "safe"
		}
	case config.SenderTypeOZGovernor:
		if prefix := hexPrefix(s.Governor, 6); prefix != "" {
			base = "governor-" + prefix
		} else {
			base = "governor"
		}
	case config.SenderTypeLedger:
		if prefix := hexPrefix(s.Address, 6); prefix != "" {
			base = "ledger-" + prefix
		} else {
			base = "ledger"
		}
	case config.SenderTypeTrezor:
		if prefix := hexPrefix(s.Address, 6); prefix != "" {
			base = "trezor-" + prefix
		} else {
			base = "trezor"
		}
	default:
		base = "account"
	}

	if base == "" {
		base = "account"
	}

	// Handle name collisions by appending numeric suffix
	name := base
	if usedNames[name] {
		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s-%d", base, i)
			if !usedNames[candidate] {
				name = candidate
				break
			}
		}
	}

	return name
}

// nameFromPrivateKeyRef extracts a human-readable name from a private key env var reference.
// e.g., "${DEPLOYER_PRIVATE_KEY}" → "deployer-key"
func nameFromPrivateKeyRef(value string) string {
	v := value
	if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
		v = v[2 : len(v)-1]
	} else if strings.HasPrefix(v, "$") {
		v = v[1:]
	} else {
		return "private-key"
	}

	// Remove "PRIVATE_" to shorten the name (e.g., DEPLOYER_PRIVATE_KEY → DEPLOYER_KEY)
	v = strings.ReplaceAll(v, "PRIVATE_", "")

	if v == "" {
		return "private-key"
	}

	// Lowercase and replace underscores with hyphens
	return strings.ToLower(strings.ReplaceAll(v, "_", "-"))
}

// hexPrefix returns the first n characters of a hex string after stripping the "0x" prefix.
func hexPrefix(addr string, n int) string {
	s := strings.TrimPrefix(strings.ToLower(addr), "0x")
	if len(s) > n {
		return s[:n]
	}
	return s
}

// sortedMapKeys returns the keys of a map sorted alphabetically.
// Works with any map[string]V type via generics.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

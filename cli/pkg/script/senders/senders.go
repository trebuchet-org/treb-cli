package senders

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// ParseSenderDependencies extracts sender dependencies from a contract artifact's natspec
func ParseSenderDependencies(artifact *contracts.Artifact) ([]string, error) {
	// Extract devdoc from metadata
	var devdoc struct {
		Methods map[string]map[string]interface{} `json:"methods"`
	}

	if err := json.Unmarshal(artifact.Metadata.Output.DevDoc, &devdoc); err != nil {
		return nil, fmt.Errorf("failed to parse devdoc: %w", err)
	}

	// Look for run() method
	runMethod, exists := devdoc.Methods["run()"]
	if !exists {
		return nil, nil // No run() method found
	}

	// Look for custom:senders tag
	customSenders, exists := runMethod["custom:senders"]
	if !exists {
		return nil, nil // No custom:senders tag found
	}

	sendersStr, ok := customSenders.(string)
	if !ok {
		return nil, fmt.Errorf("custom:senders is not a string")
	}

	return parseSendersList(sendersStr)
}

// parseSendersList parses a comma-separated list of sender names
func parseSendersList(sendersStr string) ([]string, error) {
	var senders []string

	// Split by comma and trim spaces
	parts := strings.Split(sendersStr, ",")

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		// Validate name (alphanumeric and underscore)
		if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name) {
			return nil, fmt.Errorf("invalid sender name: %s", name)
		}

		senders = append(senders, name)
	}

	return senders, nil
}

// FilterSenderConfigs filters sender configs based on required senders list
func FilterSenderConfigs(
	allConfigs *config.SenderConfigs,
	requiredSenders []string,
	allSenders map[string]config.SenderConfig,
) (*config.SenderConfigs, error) {
	// If no dependencies specified, return all configs
	if len(requiredSenders) == 0 {
		return allConfigs, nil
	}

	// Create a set of required sender names
	includedSenders := make(map[string]bool)

	// First, validate all required senders exist and add them to included set
	for _, name := range requiredSenders {
		found := false
		for _, config := range allConfigs.Configs {
			if config.Name == name {
				found = true
				includedSenders[name] = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("required sender '%s' not found in configuration", name)
		}
	}

	// Now we need to recursively include dependencies (e.g., Safe signers)
	isDependency := make(map[string]bool)
	changed := true
	for changed {
		changed = false
		for _, cfg := range allConfigs.Configs {
			if includedSenders[cfg.Name] {
				// Check if this is a Safe that depends on a signer
				if cfg.SenderType == config.SENDER_TYPE_GNOSIS_SAFE {
					// Get the original sender config to find the signer
					senderConfig := allSenders[cfg.Name]
					if senderConfig.Signer != "" && !includedSenders[senderConfig.Signer] {
						isDependency[senderConfig.Signer] = true
						includedSenders[senderConfig.Signer] = true
						changed = true
					}
				}
			}
		}
	}

	// Build filtered configs maintaining the original order
	filtered := &config.SenderConfigs{
		Configs: []config.SenderInitConfig{},
	}

	for _, cfg := range allConfigs.Configs {
		if includedSenders[cfg.Name] {
			if isDependency[cfg.Name] && (cfg.SenderType == config.SENDER_TYPE_LEDGER || cfg.SenderType == config.SENDER_TYPE_TREZOR) {
				cfg.CanBroadcast = false
			}
			filtered.Configs = append(filtered.Configs, cfg)
		}
	}

	return filtered, nil
}

// RequiresLedgerFlag determines if the forge script needs --ledger flag
func RequiresLedgerFlag(configs *config.SenderConfigs) bool {
	for _, cfg := range configs.Configs {
		if cfg.SenderType == config.SENDER_TYPE_LEDGER && cfg.CanBroadcast {
			return true
		}
	}
	return false
}

// RequiresTrezorFlag determines if the forge script needs --trezor flag
func RequiresTrezorFlag(configs *config.SenderConfigs) bool {
	for _, cfg := range configs.Configs {
		if cfg.SenderType == config.SENDER_TYPE_TREZOR && cfg.CanBroadcast {
			return true
		}
	}
	return false
}

// GetDerivationPaths returns the derivation paths for the hardware wallets
func GetDerivationPaths(configs *config.SenderConfigs) []string {
	derivationPaths := []string{}
	for _, cfg := range configs.Configs {
		if (cfg.SenderType == config.SENDER_TYPE_TREZOR || cfg.SenderType == config.SENDER_TYPE_LEDGER) && cfg.CanBroadcast {
			derivationPaths = append(derivationPaths, cfg.BaseConfig.DerivationPath)
		}
	}
	return derivationPaths
}

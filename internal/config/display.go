package config

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// LoadRawSenders loads sender configurations for the given namespace
// without expanding environment variables, so raw references like
// ${DEPLOYER_PRIVATE_KEY} are preserved for display.
func LoadRawSenders(projectRoot, namespace, configSource string) (map[string]config.SenderConfig, error) {
	switch configSource {
	case "treb.toml (v2)":
		return loadRawSendersV2(projectRoot, namespace)
	case "treb.toml":
		return loadRawSendersV1(projectRoot, namespace)
	case "foundry.toml":
		return loadRawSendersFoundry(projectRoot, namespace)
	default:
		return nil, nil
	}
}

func loadRawSendersV2(projectRoot, namespace string) (map[string]config.SenderConfig, error) {
	trebPath := filepath.Join(projectRoot, "treb.toml")

	var raw trebFileV2Raw
	if _, err := toml.DecodeFile(trebPath, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse treb.toml: %w", err)
	}

	cfg := &config.TrebFileConfigV2{
		Accounts:  raw.Accounts,
		Namespace: raw.Namespace,
	}
	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]config.AccountConfig)
	}
	if cfg.Namespace == nil {
		cfg.Namespace = make(map[string]config.NamespaceRoles)
	}

	// Resolve namespace WITHOUT env var expansion
	resolved, err := ResolveNamespace(cfg, namespace, io.Discard)
	if err != nil {
		return nil, err
	}

	senders := make(map[string]config.SenderConfig, len(resolved.Accounts))
	for role, acct := range resolved.Accounts {
		senders[role] = config.SenderConfig(acct)
	}

	return senders, nil
}

func loadRawSendersV1(projectRoot, namespace string) (map[string]config.SenderConfig, error) {
	trebPath := filepath.Join(projectRoot, "treb.toml")

	var cfg config.TrebFileConfig
	if _, err := toml.DecodeFile(trebPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse treb.toml: %w", err)
	}

	// Default profile to namespace name when omitted
	for nsName, nsCfg := range cfg.Ns {
		if nsCfg.Profile == "" {
			nsCfg.Profile = nsName
			cfg.Ns[nsName] = nsCfg
		}
	}

	// Merge default + active namespace senders (without env expansion)
	senders := make(map[string]config.SenderConfig)
	if defaultNs, ok := cfg.Ns["default"]; ok {
		for k, v := range defaultNs.Senders {
			senders[k] = v
		}
	}
	if namespace != "default" {
		if activeNs, ok := cfg.Ns[namespace]; ok {
			for k, v := range activeNs.Senders {
				senders[k] = v
			}
		}
	}

	return senders, nil
}

func loadRawSendersFoundry(projectRoot, namespace string) (map[string]config.SenderConfig, error) {
	foundryPath := filepath.Join(projectRoot, "foundry.toml")

	var cfg config.FoundryConfig
	if _, err := toml.DecodeFile(foundryPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	senders := make(map[string]config.SenderConfig)

	// Start with default profile senders
	if defaultProfile, ok := cfg.Profile["default"]; ok && defaultProfile.Treb != nil {
		for k, v := range defaultProfile.Treb.Senders {
			senders[k] = v
		}
	}

	// Overlay active profile senders
	if namespace != "default" {
		if profile, ok := cfg.Profile[namespace]; ok && profile.Treb != nil {
			for k, v := range profile.Treb.Senders {
				senders[k] = v
			}
		}
	}

	if len(senders) == 0 {
		return nil, nil
	}

	return senders, nil
}

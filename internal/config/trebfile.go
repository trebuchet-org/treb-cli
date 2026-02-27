package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// loadTrebConfig loads and parses treb.toml if it exists.
// Returns (nil, nil) when treb.toml does not exist.
func loadTrebConfig(projectRoot string) (*config.TrebFileConfig, error) {
	trebPath := filepath.Join(projectRoot, "treb.toml")

	if _, err := os.Stat(trebPath); os.IsNotExist(err) {
		return nil, nil
	}

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

	// Expand environment variables in all sender config string fields
	for nsName, nsCfg := range cfg.Ns {
		for senderName, sender := range nsCfg.Senders {
			sender.PrivateKey = os.ExpandEnv(sender.PrivateKey)
			sender.Safe = os.ExpandEnv(sender.Safe)
			sender.Address = os.ExpandEnv(sender.Address)
			sender.Signer = os.ExpandEnv(sender.Signer)
			sender.DerivationPath = os.ExpandEnv(sender.DerivationPath)
			sender.Governor = os.ExpandEnv(sender.Governor)
			sender.Timelock = os.ExpandEnv(sender.Timelock)
			sender.Proposer = os.ExpandEnv(sender.Proposer)
			nsCfg.Senders[senderName] = sender
		}
		cfg.Ns[nsName] = nsCfg
	}

	return &cfg, nil
}

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// TrebConfig represents the treb-specific configuration within a profile
type TrebConfig struct {
	Senders         map[string]SenderConfig `toml:"senders"`
	LibraryDeployer string                  `toml:"library_deployer,omitempty"`
}

// FoundryProfileConfig represents a complete profile configuration including treb
type FoundryProfileConfig struct {
	Treb TrebConfig `toml:"treb"`
	// Other Foundry profile settings can be added here
}

// FoundryFullConfig represents the complete foundry.toml structure
type FoundryFullConfig struct {
	Profile map[string]FoundryProfileConfig `toml:"profile"`
}

// LoadTrebConfig loads the treb configuration from foundry.toml
func LoadTrebConfig(projectPath string) (*FoundryFullConfig, error) {
	// Load environment variables from .env files
	envFiles := []string{
		fmt.Sprintf("%s/.env", projectPath),
		fmt.Sprintf("%s/.env.local", projectPath),
	}

	if err := LoadEnvFiles(envFiles...); err != nil {
		return nil, fmt.Errorf("failed to load .env files: %w", err)
	}

	foundryPath := fmt.Sprintf("%s/foundry.toml", projectPath)

	// Read and parse the file
	var config FoundryFullConfig
	if _, err := toml.DecodeFile(foundryPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	// Expand environment variables in senders
	for profileName, profile := range config.Profile {
		for senderName, sender := range profile.Treb.Senders {
			if sender.PrivateKey != "" {
				sender.PrivateKey = expandEnvVar(sender.PrivateKey)
			}
			if sender.Safe != "" {
				sender.Safe = expandEnvVar(sender.Safe)
			}
			if sender.Signer != "" {
				sender.Signer = expandEnvVar(sender.Signer)
			}
			if sender.DerivationPath != "" {
				sender.DerivationPath = expandEnvVar(sender.DerivationPath)
			}
			if sender.Address != "" {
				sender.Address = expandEnvVar(sender.Address)
			}
			// Update the sender in the map
			profile.Treb.Senders[senderName] = sender
		}
		// Update the profile
		config.Profile[profileName] = profile
	}

	return &config, nil
}

// GetProfileTrebConfig returns the treb config for a specific profile
func (fc *FoundryFullConfig) GetProfileTrebConfig(profileName string) (*TrebConfig, error) {
	if profile, exists := fc.Profile[profileName]; exists {
		return &profile.Treb, nil
	}
	return nil, fmt.Errorf("profile '%s' not found", profileName)
}

// expandEnvVar expands environment variables in a string
func expandEnvVar(s string) string {
	return os.ExpandEnv(s)
}

// GetSenderNameByAddress looks up a sender name by its address
func (tc *TrebConfig) GetSenderNameByAddress(address string) (string, error) {
	if tc == nil || tc.Senders == nil {
		return "", fmt.Errorf("no senders configured")
	}

	// Normalize the address for comparison
	address = strings.ToLower(address)

	for name, sender := range tc.Senders {
		switch sender.Type {
		case "safe":
			if strings.ToLower(sender.Safe) == address {
				return name, nil
			}
		case "private_key":
			// For private key senders, we need to derive the address
			addr, err := GetAddressFromPrivateKey(sender.PrivateKey)
			if err == nil && strings.ToLower(addr) == address {
				return name, nil
			}
		case "ledger":
			// For ledger, we can't easily derive the address without the device
			// Skip for now
		}
	}

	return "", fmt.Errorf("sender not found for address: %s", address)
}

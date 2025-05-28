package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type DeployConfig struct {
	Profile map[string]ProfileConfig `toml:"profile"`
}

type ProfileConfig struct {
	// New structure: senders are defined at profile level
	Senders         map[string]SenderConfig `toml:"senders"`
	LibraryDeployer string                  `toml:"library_deployer,omitempty"`
}

type SenderConfig struct {
	Type           string `toml:"type"`
	PrivateKey     string `toml:"private_key,omitempty"`
	Safe           string `toml:"safe,omitempty"`
	Signer         string `toml:"signer,omitempty"` // For Safe senders
	DerivationPath string `toml:"derivation_path,omitempty"` // For Ledger senders
}

// LoadDeployConfig loads deploy configuration from foundry.toml
func LoadDeployConfig(projectPath string) (*DeployConfig, error) {
	// Load environment variables from .env files
	envFiles := []string{
		fmt.Sprintf("%s/.env", projectPath),
		fmt.Sprintf("%s/.env.local", projectPath),
	}
	
	if err := LoadEnvFiles(envFiles...); err != nil {
		return nil, fmt.Errorf("failed to load .env files: %w", err)
	}

	foundryPath := fmt.Sprintf("%s/foundry.toml", projectPath)
	
	// Check if foundry.toml exists
	if _, err := os.Stat(foundryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("foundry.toml not found at %s", foundryPath)
	}

	// Read and parse the file
	var config DeployConfig
	if _, err := toml.DecodeFile(foundryPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	// Expand environment variables
	if err := expandEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}

	return &config, nil
}

// GetProfileConfig returns the deploy config for a specific profile (usually "treb")
func (dc *DeployConfig) GetProfileConfig(profile string) (*ProfileConfig, error) {
	if profileConfig, exists := dc.Profile[profile]; exists {
		return &profileConfig, nil
	}
	return nil, fmt.Errorf("profile configuration '%s' not found", profile)
}

// GetSender returns the sender configuration for a specific sender name
func (dc *DeployConfig) GetSender(profile, senderName string) (*SenderConfig, error) {
	profileConfig, err := dc.GetProfileConfig(profile)
	if err != nil {
		return nil, err
	}
	
	if sender, exists := profileConfig.Senders[senderName]; exists {
		return &sender, nil
	}
	return nil, fmt.Errorf("sender '%s' not found in profile '%s'", senderName, profile)
}

// Validate checks if the deploy configuration is valid for a given sender
func (dc *DeployConfig) Validate(namespace string) error {
	// For now, we don't validate namespace since it's created on-demand
	// We'll validate the sender when it's actually used
	return nil
}

// ValidateSender checks if a specific sender configuration is valid
func (dc *DeployConfig) ValidateSender(profile, senderName string) error {
	sender, err := dc.GetSender(profile, senderName)
	if err != nil {
		return err
	}
	
	switch sender.Type {
	case "private_key":
		if sender.PrivateKey == "" {
			return fmt.Errorf("private_key sender requires 'private_key' field")
		}
		// Check if environment variable wasn't expanded
		if strings.Contains(sender.PrivateKey, "${") {
			return fmt.Errorf("environment variable not expanded in private_key: %s", sender.PrivateKey)
		}
		// Validate private key format (basic check)
		if !strings.HasPrefix(sender.PrivateKey, "0x") || len(sender.PrivateKey) != 66 {
			return fmt.Errorf("invalid private key format (should be 0x... with 64 hex chars), got: '%s' (length: %d)", sender.PrivateKey, len(sender.PrivateKey))
		}
	case "safe":
		if sender.Safe == "" {
			return fmt.Errorf("safe sender requires 'safe' field")
		}
		if sender.Signer == "" {
			return fmt.Errorf("safe sender requires 'signer' field")
		}
	case "ledger":
		if sender.DerivationPath == "" {
			return fmt.Errorf("ledger sender requires 'derivation_path' field")
		}
	default:
		return fmt.Errorf("unsupported sender type: %s", sender.Type)
	}

	return nil
}

// GenerateEnvVars generates environment variables for the forge script
func (dc *DeployConfig) GenerateEnvVars(namespace string) (map[string]string, error) {
	// For now, we'll use the "treb" profile by default
	// In the future, this could be configurable
	envVars := make(map[string]string)
	
	// Set namespace (previously environment)
	envVars["DEPLOYMENT_NAMESPACE"] = namespace
	
	// Note: Sender configuration will be handled separately when needed
	// since it's no longer tied to namespace
	
	return envVars, nil
}

// GenerateSenderEnvVars generates environment variables for a specific sender
func (dc *DeployConfig) GenerateSenderEnvVars(profile, senderName string) (map[string]string, error) {
	sender, err := dc.GetSender(profile, senderName)
	if err != nil {
		return nil, err
	}
	
	envVars := make(map[string]string)

	switch sender.Type {
	case "private_key":
		envVars["SENDER_TYPE"] = "private_key"
		envVars["DEPLOYER_PRIVATE_KEY"] = sender.PrivateKey
		
		// Derive address from private key
		address, err := GetAddressFromPrivateKey(sender.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to derive address from private key: %w", err)
		}
		envVars["SENDER_ADDRESS"] = address
	case "safe":
		envVars["SENDER_TYPE"] = "safe"
		envVars["SENDER_ADDRESS"] = sender.Safe
		envVars["SENDER_SIGNER"] = sender.Signer
	case "ledger":
		envVars["SENDER_TYPE"] = "ledger"
		envVars["LEDGER_DERIVATION_PATH"] = sender.DerivationPath
		
		// Resolve address dynamically using cast
		address, err := GetLedgerAddress(sender.DerivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve ledger address: %w", err)
		}
		envVars["SENDER_ADDRESS"] = address
	}

	return envVars, nil
}

// expandEnvVars expands ${VAR} patterns in the config
func expandEnvVars(config *DeployConfig) error {
	envVarRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	for profileName, profileConfig := range config.Profile {
		// Expand library deployer
		if profileConfig.LibraryDeployer != "" {
			profileConfig.LibraryDeployer = expandString(profileConfig.LibraryDeployer, envVarRegex)
		}
		
		// Expand sender fields
		for senderName, sender := range profileConfig.Senders {
			if sender.PrivateKey != "" {
				sender.PrivateKey = expandString(sender.PrivateKey, envVarRegex)
			}
			if sender.Safe != "" {
				sender.Safe = expandString(sender.Safe, envVarRegex)
			}
			if sender.Signer != "" {
				sender.Signer = expandString(sender.Signer, envVarRegex)
			}
			if sender.DerivationPath != "" {
				sender.DerivationPath = expandString(sender.DerivationPath, envVarRegex)
			}
			
			// Update the sender in the map
			profileConfig.Senders[senderName] = sender
		}
		
		// Update the config
		config.Profile[profileName] = profileConfig
	}
	
	return nil
}

func expandString(s string, regex *regexp.Regexp) string {
	return regex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]
		value := os.Getenv(varName)
		if value == "" {
			// For validation, we should fail if a required env var is not set
			// But for now, let's return the match to make it clear what's missing
			return match
		}
		return value
	})
}
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
	Deployer DeployerConfig `toml:"deployer"`
}

type DeployerConfig struct {
	Type       string          `toml:"type"`
	PrivateKey string          `toml:"private_key,omitempty"`
	Safe       string          `toml:"safe,omitempty"`
	Proposer   *ProposerConfig `toml:"proposer,omitempty"`
}

type ProposerConfig struct {
	Type           string `toml:"type"`
	PrivateKey     string `toml:"private_key,omitempty"`
	Address        string `toml:"address,omitempty"`
	DerivationPath string `toml:"derivation_path,omitempty"`
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

// GetEnvironmentConfig returns the deploy config for a specific environment
func (dc *DeployConfig) GetEnvironmentConfig(env string) (*ProfileConfig, error) {
	if profileConfig, exists := dc.Profile[env]; exists {
		return &profileConfig, nil
	}
	return nil, fmt.Errorf("profile configuration for environment '%s' not found", env)
}

// GetDeployer returns the deployer configuration for a specific environment
func (dc *DeployConfig) GetDeployer(env string) *DeployerConfig {
	if profileConfig, exists := dc.Profile[env]; exists {
		return &profileConfig.Deployer
	}
	return nil
}

// Validate checks if the deploy configuration is valid
func (dc *DeployConfig) Validate(env string) error {
	envConfig, err := dc.GetEnvironmentConfig(env)
	if err != nil {
		return err
	}

	deployer := envConfig.Deployer
	
	switch deployer.Type {
	case "private_key":
		if deployer.PrivateKey == "" {
			return fmt.Errorf("private_key deployer requires 'private_key' field")
		}
		// Check if environment variable wasn't expanded
		if strings.Contains(deployer.PrivateKey, "${") {
			return fmt.Errorf("environment variable not expanded in private_key: %s", deployer.PrivateKey)
		}
		// Validate private key format (basic check)
		if !strings.HasPrefix(deployer.PrivateKey, "0x") || len(deployer.PrivateKey) != 66 {
			return fmt.Errorf("invalid private key format (should be 0x... with 64 hex chars), got: '%s' (length: %d)", deployer.PrivateKey, len(deployer.PrivateKey))
		}
	case "safe":
		if deployer.Safe == "" {
			return fmt.Errorf("safe deployer requires 'safe' field")
		}
		if deployer.Proposer == nil {
			return fmt.Errorf("safe deployer requires 'proposer' configuration")
		}
		
		// Validate proposer
		switch deployer.Proposer.Type {
		case "private_key":
			if deployer.Proposer.PrivateKey == "" {
				return fmt.Errorf("private_key proposer requires 'private_key' field")
			}
		case "ledger":
			if deployer.Proposer.DerivationPath == "" {
				return fmt.Errorf("ledger proposer requires 'derivation_path' field")
			}
		default:
			return fmt.Errorf("unsupported proposer type: %s", deployer.Proposer.Type)
		}
	default:
		return fmt.Errorf("unsupported deployer type: %s", deployer.Type)
	}

	return nil
}

// GenerateEnvVars generates environment variables for the forge script
func (dc *DeployConfig) GenerateEnvVars(env string) (map[string]string, error) {
	envConfig, err := dc.GetEnvironmentConfig(env)
	if err != nil {
		return nil, err
	}

	envVars := make(map[string]string)
	deployer := envConfig.Deployer

	// Set common environment
	envVars["DEPLOYMENT_ENV"] = env

	switch deployer.Type {
	case "private_key":
		envVars["DEPLOYER_TYPE"] = "private_key"
		envVars["DEPLOYER_PRIVATE_KEY"] = deployer.PrivateKey
	case "safe":
		envVars["DEPLOYER_TYPE"] = "safe"
		envVars["DEPLOYER_SAFE_ADDRESS"] = deployer.Safe
		
		// Set proposer information
		if deployer.Proposer != nil {
			switch deployer.Proposer.Type {
			case "private_key":
				envVars["PROPOSER_TYPE"] = "private_key"
				envVars["PROPOSER_PRIVATE_KEY"] = deployer.Proposer.PrivateKey
			case "ledger":
				envVars["PROPOSER_TYPE"] = "ledger"
				envVars["PROPOSER_DERIVATION_PATH"] = deployer.Proposer.DerivationPath
				
				// Dynamically resolve the ledger address
				if deployer.Proposer.Address != "" {
					// Use explicitly provided address if available
					envVars["PROPOSER_ADDRESS"] = deployer.Proposer.Address
				} else {
					// Resolve address dynamically using cast
					address, err := GetLedgerAddress(deployer.Proposer.DerivationPath)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve ledger address: %w", err)
					}
					envVars["PROPOSER_ADDRESS"] = address
				}
			}
		}
	}

	return envVars, nil
}

// expandEnvVars expands ${VAR} patterns in the config
func expandEnvVars(config *DeployConfig) error {
	envVarRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	for profileName, profileConfig := range config.Profile {
		// Expand deployer fields
		if profileConfig.Deployer.PrivateKey != "" {
			profileConfig.Deployer.PrivateKey = expandString(profileConfig.Deployer.PrivateKey, envVarRegex)
		}
		if profileConfig.Deployer.Safe != "" {
			profileConfig.Deployer.Safe = expandString(profileConfig.Deployer.Safe, envVarRegex)
		}
		
		// Expand proposer fields
		if profileConfig.Deployer.Proposer != nil {
			if profileConfig.Deployer.Proposer.PrivateKey != "" {
				profileConfig.Deployer.Proposer.PrivateKey = expandString(profileConfig.Deployer.Proposer.PrivateKey, envVarRegex)
			}
			if profileConfig.Deployer.Proposer.Address != "" {
				profileConfig.Deployer.Proposer.Address = expandString(profileConfig.Deployer.Proposer.Address, envVarRegex)
			}
			if profileConfig.Deployer.Proposer.DerivationPath != "" {
				profileConfig.Deployer.Proposer.DerivationPath = expandString(profileConfig.Deployer.Proposer.DerivationPath, envVarRegex)
			}
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
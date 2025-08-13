package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

// FoundryTOML represents the raw foundry.toml structure
type FoundryTOML struct {
	RpcEndpoints map[string]string            `toml:"rpc_endpoints"`
	Etherscan    map[string]map[string]string `toml:"etherscan"`
	Profile      map[string]map[string]any    `toml:"profile"`
}

// loadFoundryConfig loads and parses foundry.toml
func loadFoundryConfig(projectRoot string) (*FoundryConfig, error) {
	// Load .env files first for variable expansion
	envFiles := []string{
		filepath.Join(projectRoot, ".env"),
		filepath.Join(projectRoot, ".env.local"),
	}

	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err != nil {
				// Log warning but don't fail
				fmt.Fprintf(os.Stderr, "Warning: Failed to load %s: %v\n", envFile, err)
			}
		}
	}

	// Load foundry.toml
	foundryPath := filepath.Join(projectRoot, "foundry.toml")
	var raw FoundryTOML

	if _, err := toml.DecodeFile(foundryPath, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	// Convert to our config structure
	cfg := &FoundryConfig{
		RpcEndpoints: make(map[string]string),
		Etherscan:    make(map[string]EtherscanConfig),
		Profiles:     make(map[string]ProfileConfig),
	}

	// Process RPC endpoints
	for name, url := range raw.RpcEndpoints {
		cfg.RpcEndpoints[name] = os.ExpandEnv(url)
	}

	// Process etherscan configs
	for network, ethConfig := range raw.Etherscan {
		ec := EtherscanConfig{}
		if url, ok := ethConfig["url"]; ok {
			ec.URL = os.ExpandEnv(url)
		}
		if key, ok := ethConfig["key"]; ok {
			ec.APIKey = os.ExpandEnv(key)
		}
		cfg.Etherscan[network] = ec
	}

	// Process profiles
	for profileName, profileData := range raw.Profile {
		profile := ProfileConfig{}

		// Look for treb section
		if trebData, ok := profileData["treb"]; ok {
			if trebMap, ok := trebData.(map[string]any); ok {
				trebConfig, err := parseTrebConfig(trebMap)
				if err != nil {
					return nil, fmt.Errorf("failed to parse treb config for profile %s: %w", profileName, err)
				}
				profile.Treb = *trebConfig
			}
		}

		cfg.Profiles[profileName] = profile
	}

	return cfg, nil
}

// parseTrebConfig parses the treb section of a profile
func parseTrebConfig(data map[string]any) (*TrebConfig, error) {
	cfg := &TrebConfig{
		Senders: make(map[string]SenderConfig),
	}

	// Parse senders
	if sendersData, ok := data["senders"]; ok {
		if sendersMap, ok := sendersData.(map[string]any); ok {
			for name, senderData := range sendersMap {
				if senderMap, ok := senderData.(map[string]any); ok {
					sender, err := parseSenderConfig(name, senderMap)
					if err != nil {
						return nil, fmt.Errorf("failed to parse sender %s: %w", name, err)
					}
					cfg.Senders[name] = *sender
				}
			}
		}
	}

	// Parse library deployer
	if deployer, ok := data["library_deployer"]; ok {
		if deployerStr, ok := deployer.(string); ok {
			cfg.LibraryDeployer = deployerStr
		}
	}

	return cfg, nil
}

// parseSenderConfig parses a single sender configuration
func parseSenderConfig(name string, data map[string]any) (*SenderConfig, error) {
	sender := &SenderConfig{
		Name: name,
	}

	// Type is required
	if senderType, ok := data["type"]; ok {
		if typeStr, ok := senderType.(string); ok {
			sender.Type = typeStr
		} else {
			return nil, fmt.Errorf("sender type must be a string")
		}
	} else {
		return nil, fmt.Errorf("sender type is required")
	}

	// Parse type-specific fields
	switch sender.Type {
	case "private_key":
		if pk, ok := data["private_key"]; ok {
			if pkStr, ok := pk.(string); ok {
				sender.PrivateKey = pkStr
			}
		}
	case "safe":
		if safe, ok := data["safe"]; ok {
			if safeStr, ok := safe.(string); ok {
				sender.Safe = safeStr
			}
		}
		if proposer, ok := data["proposer"]; ok {
			if proposerStr, ok := proposer.(string); ok {
				sender.Proposer = proposerStr
			}
		}
	case "ledger":
		if path, ok := data["derivation_path"]; ok {
			if pathStr, ok := path.(string); ok {
				sender.DerivationPath = pathStr
			}
		}
	}

	// Optional address field
	if addr, ok := data["address"]; ok {
		if addrStr, ok := addr.(string); ok {
			sender.Address = addrStr
		}
	}

	return sender, nil
}

// loadTrebConfig extracts treb config for a specific profile
func loadTrebConfig(foundryConfig *FoundryConfig, profile string) (*TrebConfig, error) {
	if foundryConfig == nil || foundryConfig.Profiles == nil {
		return nil, fmt.Errorf("no profiles found in foundry config")
	}

	profileConfig, exists := foundryConfig.Profiles[profile]
	if !exists {
		return nil, fmt.Errorf("profile %s not found", profile)
	}

	// Return a copy to avoid mutation
	trebConfig := profileConfig.Treb
	return &trebConfig, nil
}


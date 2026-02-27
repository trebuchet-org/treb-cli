package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// envVarPattern matches ${VAR_NAME} patterns in TOML values
var envVarPattern = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)

// DetectEnvVar checks if a raw TOML value is a simple ${VAR_NAME} reference.
// Returns the variable name and true if the value is a pure env var reference.
func DetectEnvVar(rawValue string) (string, bool) {
	matches := envVarPattern.FindStringSubmatch(rawValue)
	if len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

// GenerateEnvVarName generates a conventional env var name for a network's RPC URL.
// Convention: uppercase, dashes/dots to underscores, append _RPC_URL.
// Examples: sepolia -> SEPOLIA_RPC_URL, celo-sepolia -> CELO_SEPOLIA_RPC_URL
func GenerateEnvVarName(networkName string) string {
	name := strings.ToUpper(networkName)
	name = strings.NewReplacer("-", "_", ".", "_").Replace(name)
	return name + "_RPC_URL"
}

// LoadRawRPCEndpoints reads foundry.toml and returns RPC endpoints without env var expansion.
func LoadRawRPCEndpoints(projectRoot string) (map[string]string, error) {
	foundryPath := filepath.Join(projectRoot, "foundry.toml")

	var cfg config.FoundryConfig
	if _, err := toml.DecodeFile(foundryPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	return cfg.RpcEndpoints, nil
}

// LoadRawRPCEndpoint reads a single raw RPC endpoint value from foundry.toml (before env var expansion).
func LoadRawRPCEndpoint(projectRoot string, networkName string) (string, error) {
	endpoints, err := LoadRawRPCEndpoints(projectRoot)
	if err != nil {
		return "", err
	}

	raw, ok := endpoints[networkName]
	if !ok {
		return "", fmt.Errorf("network '%s' not found in foundry.toml [rpc_endpoints]", networkName)
	}

	return raw, nil
}

// MigrateRPCEndpoint replaces a hardcoded RPC URL in foundry.toml with an env var reference
// and appends the env var assignment to .env.
func MigrateRPCEndpoint(projectRoot, networkName, rawURL string) error {
	envVarName := GenerateEnvVarName(networkName)

	// Update foundry.toml
	if err := updateFoundryTOML(projectRoot, networkName, rawURL, envVarName); err != nil {
		return fmt.Errorf("failed to update foundry.toml: %w", err)
	}

	// Append to .env
	if err := appendToEnvFile(projectRoot, envVarName, rawURL); err != nil {
		return fmt.Errorf("failed to update .env: %w", err)
	}

	return nil
}

// updateFoundryTOML replaces the RPC endpoint value for the given network in foundry.toml.
func updateFoundryTOML(projectRoot, networkName, oldValue, envVarName string) error {
	foundryPath := filepath.Join(projectRoot, "foundry.toml")

	data, err := os.ReadFile(foundryPath) //nolint:gosec // internal path
	if err != nil {
		return fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	content := string(data)

	// Build the old line pattern: networkName = "oldValue"
	// TOML uses key = "value" format for string values
	oldEntry := fmt.Sprintf(`%s = "%s"`, networkName, oldValue)
	newEntry := fmt.Sprintf(`%s = "${%s}"`, networkName, envVarName)

	if !strings.Contains(content, oldEntry) {
		return fmt.Errorf("could not find entry '%s' in foundry.toml", oldEntry)
	}

	content = strings.Replace(content, oldEntry, newEntry, 1)

	if err := os.WriteFile(foundryPath, []byte(content), 0644); err != nil { //nolint:gosec // internal path
		return fmt.Errorf("failed to write foundry.toml: %w", err)
	}

	return nil
}

// appendToEnvFile appends an env var assignment to the .env file.
func appendToEnvFile(projectRoot, envVarName, value string) error {
	envPath := filepath.Join(projectRoot, ".env")

	// Read existing content to check for duplicates and trailing newline
	existing, err := os.ReadFile(envPath) //nolint:gosec // internal path
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .env: %w", err)
	}

	entry := fmt.Sprintf("%s=%s", envVarName, value)

	// Check if already exists
	if strings.Contains(string(existing), envVarName+"=") {
		return nil // Already present
	}

	// Open for appending (create if not exists)
	f, err := os.OpenFile(envPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // internal path
	if err != nil {
		return fmt.Errorf("failed to open .env: %w", err)
	}
	defer f.Close()

	// Ensure we start on a new line if file has content
	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}

	if _, err := fmt.Fprintf(f, "%s%s\n", prefix, entry); err != nil {
		return fmt.Errorf("failed to write to .env: %w", err)
	}

	return nil
}

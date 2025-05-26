package network

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// NetworkInfo contains resolved network details
type NetworkInfo struct {
	Name   string
	RpcUrl string
}

// Resolver handles network resolution from foundry.toml and chain ID extraction
type Resolver struct {
	projectRoot string
}

var chainIDMap = make(map[string]uint64)

// NewResolver creates a new network resolver
func NewResolver(projectRoot string) *Resolver {
	// Try to load .env file from project root
	envPath := filepath.Join(projectRoot, ".env")
	if _, err := os.Stat(envPath); err == nil {
		// Load .env file, but don't fail if it doesn't exist
		godotenv.Load(envPath)
	}

	return &Resolver{
		projectRoot: projectRoot,
	}
}

// ResolveNetwork resolves a network name to RPC URL and chain ID
func (r *Resolver) ResolveNetwork(network string) (*NetworkInfo, error) {
	// Get RPC URL from foundry.toml
	rpcUrl, err := r.getRpcUrlFromFoundry(network)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network %s: %w", network, err)
	}

	return &NetworkInfo{
		Name:   network,
		RpcUrl: rpcUrl,
	}, nil
}

func (n *NetworkInfo) ChainID() uint64 {
	if chainIDMap[n.Name] == 0 {
		chainID, err := n.getChainID()
		if err != nil {
			panic(err)
		}
		chainIDMap[n.Name] = chainID
	}
	return chainIDMap[n.Name]
}

// getRpcUrlFromFoundry extracts RPC URL from foundry.toml
func (r *Resolver) getRpcUrlFromFoundry(network string) (string, error) {
	// For now, use a simple approach - in production, you'd want to parse foundry.toml properly
	// or use forge config command
	return r.getFoundryRpcUrl(network)
}

// getFoundryRpcUrl reads foundry.toml and resolves RPC URL with env var substitution
func (r *Resolver) getFoundryRpcUrl(network string) (string, error) {
	foundryToml := fmt.Sprintf("%s/foundry.toml", r.projectRoot)

	content, err := os.ReadFile(foundryToml)
	if err != nil {
		return "", fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	// Simple TOML parsing for rpc_endpoints
	lines := strings.Split(string(content), "\n")
	inRpcSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "[rpc_endpoints]" {
			inRpcSection = true
			continue
		}

		if strings.HasPrefix(line, "[") && line != "[rpc_endpoints]" {
			inRpcSection = false
			continue
		}

		if inRpcSection && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if key == network {
					// Remove quotes
					value = strings.Trim(value, `"`)

					// Substitute environment variables
					return r.substituteEnvVars(value), nil
				}
			}
		}
	}

	return "", fmt.Errorf("network %s not found in foundry.toml", network)
}

// substituteEnvVars replaces ${VAR_NAME} with environment variable values
func (r *Resolver) substituteEnvVars(value string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	result := re.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}
		// If env var not found, return the original placeholder
		// This will help identify missing env vars
		return match
	})

	// Check if there are still unresolved variables
	if strings.Contains(result, "${") {
		// Log a warning but don't fail - let the RPC call fail with a clearer error
		fmt.Fprintf(os.Stderr, "Warning: Unresolved environment variables in RPC URL: %s\n", result)
	}

	return result
}

// getChainID extracts chain ID from RPC endpoint
func (n *NetworkInfo) getChainID() (uint64, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Prepare eth_chainId JSON-RPC request
	requestBody := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`

	resp, err := client.Post(n.RpcUrl, "application/json", strings.NewReader(requestBody))
	if err != nil {
		return 0, fmt.Errorf("failed to make RPC request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON-RPC response
	var rpcResponse struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return 0, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if rpcResponse.Error != nil {
		return 0, fmt.Errorf("RPC error: %s", rpcResponse.Error.Message)
	}

	// Parse hex chain ID
	chainIDStr := strings.TrimPrefix(rpcResponse.Result, "0x")
	chainID, err := strconv.ParseUint(chainIDStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse chain ID: %w", err)
	}

	return chainID, nil
}

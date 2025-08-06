package network

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

// NetworkInfo contains resolved network details
type NetworkInfo struct {
	Name    string
	RpcUrl  string
	ChainID uint64
}

// ChainIDCache represents the cached chain ID mappings
type ChainIDCache struct {
	// NetworkName -> ChainID mapping
	Networks map[string]uint64 `json:"networks"`
	// RPC URL -> ChainID mapping (for custom RPCs)
	RPCs map[string]uint64 `json:"rpcs"`
	// ChainID -> Network names mapping (for reverse lookup)
	ChainNames map[uint64][]string `json:"chainNames"`
	// Timestamp of last update
	UpdatedAt time.Time `json:"updatedAt"`
}

// Resolver is a network resolver that uses foundry.toml as source of truth
type Resolver struct {
	projectRoot   string
	foundryConfig *config.FoundryConfig
	cache         *ChainIDCache
	cachePath     string
	mu            sync.RWMutex
	httpClient    *http.Client
}

// NewResolver creates a new network resolver
func NewResolver(projectRoot string) (*Resolver, error) {
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Creating resolver for project: %s\n", projectRoot)
	}

	// Load .env file if it exists (for environment variable expansion)
	envPath := filepath.Join(projectRoot, ".env")
	if _, err := os.Stat(envPath); err == nil {
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] Loading .env file: %s\n", envPath)
		}
		if err := godotenv.Load(envPath); err != nil {
			// Log warning but don't fail - .env might have syntax issues
			fmt.Fprintf(os.Stderr, "Warning: Failed to load .env file: %v\n", err)
		}
	}

	// Load foundry config
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Loading foundry config\n")
	}
	foundryConfig, err := config.LoadFoundryConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load foundry config: %w", err)
	}

	// Ensure cache directory exists
	cacheDir := filepath.Join(projectRoot, "cache")
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Creating cache directory: %s\n", cacheDir)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := filepath.Join(cacheDir, "chainIds.json")

	resolver := &Resolver{
		projectRoot:   projectRoot,
		foundryConfig: foundryConfig,
		cachePath:     cachePath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Load cache
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Loading cache from: %s\n", cachePath)
	}
	resolver.loadCache()

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Resolver created successfully\n")
	}

	return resolver, nil
}

// GetNetworks returns all configured networks from foundry.toml
func (r *Resolver) GetNetworks() []string {
	networks := []string{}
	for network := range r.foundryConfig.RpcEndpoints {
		networks = append(networks, network)
	}
	return networks
}

// GetExplorerURL returns the explorer URL for a network if configured
func (r *Resolver) GetExplorerURL(networkName string) (string, error) {
	if r.foundryConfig.Etherscan != nil {
		if etherscan, exists := r.foundryConfig.Etherscan[networkName]; exists && etherscan.URL != "" {
			// Expand environment variables in explorer URL
			return r.expandEnvVars(etherscan.URL), nil
		}
	}
	
	// Fallback to common defaults for well-known networks
	info, err := r.ResolveNetwork(networkName)
	if err != nil {
		return "", fmt.Errorf("network not found: %s", networkName)
	}
	
	// Return default explorers for common chains
	switch info.ChainID {
	case 1:
		return "https://etherscan.io", nil
	case 5:
		return "https://goerli.etherscan.io", nil
	case 11155111:
		return "https://sepolia.etherscan.io", nil
	case 10:
		return "https://optimistic.etherscan.io", nil
	case 137:
		return "https://polygonscan.com", nil
	case 8453:
		return "https://basescan.org", nil
	case 42161:
		return "https://arbiscan.io", nil
	case 43114:
		return "https://snowtrace.io", nil
	case 56:
		return "https://bscscan.com", nil
	case 250:
		return "https://ftmscan.com", nil
	case 1101:
		return "https://zkevm.polygonscan.com", nil
	case 324:
		return "https://explorer.zksync.io", nil
	case 42220:
		return "https://celoscan.io", nil
	case 44787:
		return "https://alfajores.celoscan.io", nil
	default:
		return "", fmt.Errorf("no explorer configured for network: %s", networkName)
	}
}

// ResolveNetwork resolves a network name to its configuration
func (r *Resolver) ResolveNetwork(networkName string) (*NetworkInfo, error) {
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Resolving network: %s\n", networkName)
	}

	// Check if network exists in foundry.toml
	rpcURL, exists := r.foundryConfig.RpcEndpoints[networkName]
	if !exists {
		return nil, fmt.Errorf("network '%s' not found in foundry.toml [rpc_endpoints]", networkName)
	}

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Found RPC URL: %s\n", rpcURL)
	}

	// Expand environment variables in RPC URL
	expandedURL := r.expandEnvVars(rpcURL)
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Expanded RPC URL: %s\n", expandedURL)
	}
	
	// Check if environment variable expansion failed (variable not set)
	if strings.Contains(expandedURL, "${") || (strings.Contains(rpcURL, "$") && expandedURL == rpcURL) {
		// Only warn if it looks like a variable that didn't expand
		missingVar := ""
		if start := strings.Index(expandedURL, "${"); start != -1 {
			if end := strings.Index(expandedURL[start:], "}"); end != -1 {
				missingVar = expandedURL[start+2 : start+end]
			}
		}
		if missingVar != "" {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Environment variable '%s' not set for network '%s'\n", missingVar, networkName)
		}
	}
	
	rpcURL = expandedURL

	// Check cache first
	r.mu.RLock()
	chainID, cached := r.cache.Networks[networkName]
	r.mu.RUnlock()

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Cache lookup for %s: cached=%v, chainID=%d\n", networkName, cached, chainID)
	}

	if !cached {
		// Fetch chain ID from RPC
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] Fetching chain ID from RPC: %s\n", rpcURL)
		}
		fetchedChainID, err := r.fetchChainID(rpcURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chain ID for network %s: %w", networkName, err)
		}
		chainID = fetchedChainID

		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] Fetched chain ID: %d\n", chainID)
		}

		// Update cache
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] Updating cache for network %s\n", networkName)
		}
		r.updateCache(networkName, rpcURL, chainID)
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] Cache updated successfully\n")
		}
	}

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Returning network info for %s\n", networkName)
	}

	return &NetworkInfo{
		Name:    networkName,
		RpcUrl:  rpcURL,
		ChainID: chainID,
	}, nil
}

// ResolveByChainID finds network names for a given chain ID
func (r *Resolver) ResolveByChainID(chainID uint64) ([]string, error) {
	r.mu.RLock()
	names, exists := r.cache.ChainNames[chainID]
	r.mu.RUnlock()

	if exists && len(names) > 0 {
		return names, nil
	}

	// If not in cache, scan all networks
	var foundNetworks []string
	for networkName := range r.foundryConfig.RpcEndpoints {
		info, err := r.ResolveNetwork(networkName)
		if err == nil && info.ChainID == chainID {
			foundNetworks = append(foundNetworks, networkName)
		}
	}

	if len(foundNetworks) == 0 {
		return nil, fmt.Errorf("no networks found for chain ID %d", chainID)
	}

	return foundNetworks, nil
}

// GetPreferredNetwork returns the preferred network name for a chain ID
// This is useful when multiple networks point to the same chain
func (r *Resolver) GetPreferredNetwork(chainID uint64) (string, error) {
	networks, err := r.ResolveByChainID(chainID)
	if err != nil {
		return "", err
	}

	// Prefer common network names
	preferredOrder := []string{"mainnet", "ethereum", "sepolia", "goerli", "arbitrum", "optimism", "polygon", "base"}
	
	for _, preferred := range preferredOrder {
		for _, network := range networks {
			if network == preferred {
				return network, nil
			}
		}
	}

	// Return the first one if no preferred match
	return networks[0], nil
}

// expandEnvVars expands environment variables in the format ${VAR_NAME}
func (r *Resolver) expandEnvVars(value string) string {
	// Handle ${VAR_NAME} format
	for strings.Contains(value, "${") {
		start := strings.Index(value, "${")
		end := strings.Index(value[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varName := value[start+2 : end]
		envValue := os.Getenv(varName)
		value = value[:start] + envValue + value[end+1:]
	}

	// Also handle $VAR_NAME format (without braces)
	parts := strings.Fields(value)
	for i, part := range parts {
		if strings.HasPrefix(part, "$") && !strings.HasPrefix(part, "${") {
			varName := part[1:]
			if envValue := os.Getenv(varName); envValue != "" {
				parts[i] = envValue
			}
		}
	}
	
	return strings.Join(parts, " ")
}

// fetchChainID fetches the chain ID from an RPC endpoint
func (r *Resolver) fetchChainID(rpcURL string) (uint64, error) {
	// Check RPC cache first
	r.mu.RLock()
	if chainID, exists := r.cache.RPCs[rpcURL]; exists {
		r.mu.RUnlock()
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] Found chain ID in RPC cache: %d\n", chainID)
		}
		return chainID, nil
	}
	r.mu.RUnlock()

	// Prepare eth_chainId JSON-RPC request
	requestBody := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] Making RPC request to: %s\n", rpcURL)
		fmt.Fprintf(os.Stderr, "[NETWORK] Request body: %s\n", requestBody)
	}

	resp, err := r.httpClient.Post(rpcURL, "application/json", strings.NewReader(requestBody))
	if err != nil {
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] RPC request failed: %v\n", err)
		}
		return 0, fmt.Errorf("failed to make RPC request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] RPC response: %s\n", string(body))
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

	if rpcResponse.Result == "" {
		return 0, fmt.Errorf("empty chain ID response")
	}

	// Parse hex chain ID
	chainIDStr := strings.TrimPrefix(rpcResponse.Result, "0x")
	chainID, err := strconv.ParseUint(chainIDStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse chain ID: %w", err)
	}

	return chainID, nil
}

// loadCache loads the chain ID cache from disk
func (r *Resolver) loadCache() {
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: acquiring lock\n")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: initializing empty cache\n")
	}

	// Initialize empty cache
	r.cache = &ChainIDCache{
		Networks:   make(map[string]uint64),
		RPCs:       make(map[string]uint64),
		ChainNames: make(map[uint64][]string),
		UpdatedAt:  time.Now(),
	}

	// Try to load from file
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: reading file %s\n", r.cachePath)
	}
	data, err := os.ReadFile(r.cachePath)
	if err != nil {
		// Cache doesn't exist yet, that's fine
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: file not found or error: %v\n", err)
		}
		return
	}

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: loaded %d bytes from cache file\n", len(data))
	}

	// Parse cache
	if err := json.Unmarshal(data, &r.cache); err != nil {
		// Invalid cache, start fresh
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: failed to parse cache: %v\n", err)
		}
		r.cache = &ChainIDCache{
			Networks:   make(map[string]uint64),
			RPCs:       make(map[string]uint64),
			ChainNames: make(map[uint64][]string),
			UpdatedAt:  time.Now(),
		}
	} else {
		if os.Getenv("TREB_DEBUG_NETWORK") != "" {
			fmt.Fprintf(os.Stderr, "[NETWORK] loadCache: successfully loaded cache with %d networks, %d RPCs\n", 
				len(r.cache.Networks), len(r.cache.RPCs))
		}
	}
}


// saveCacheInternal saves the cache without acquiring locks (must be called with lock held)
func (r *Resolver) saveCacheInternal() error {
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] saveCacheInternal: marshaling cache\n")
	}
	
	data, err := json.MarshalIndent(r.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] saveCacheInternal: writing %d bytes to %s\n", len(data), r.cachePath)
	}

	err = os.WriteFile(r.cachePath, data, 0644)
	
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[NETWORK] saveCacheInternal: write failed: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[NETWORK] saveCacheInternal: write successful\n")
		}
	}
	
	return err
}

// updateCache updates the cache with new chain ID information
func (r *Resolver) updateCache(networkName, rpcURL string, chainID uint64) {
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] updateCache: acquiring lock\n")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] updateCache: lock acquired, updating mappings\n")
	}

	// Update network mapping
	r.cache.Networks[networkName] = chainID

	// Update RPC mapping
	r.cache.RPCs[rpcURL] = chainID

	// Update reverse mapping
	if r.cache.ChainNames[chainID] == nil {
		r.cache.ChainNames[chainID] = []string{}
	}

	// Add network name if not already present
	found := false
	for _, name := range r.cache.ChainNames[chainID] {
		if name == networkName {
			found = true
			break
		}
	}
	if !found {
		r.cache.ChainNames[chainID] = append(r.cache.ChainNames[chainID], networkName)
	}

	r.cache.UpdatedAt = time.Now()

	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		fmt.Fprintf(os.Stderr, "[NETWORK] updateCache: saving cache to disk\n")
	}

	// Save to disk (ignore errors, cache is just for performance)
	// Use internal version since we already hold the lock
	err := r.saveCacheInternal()
	if os.Getenv("TREB_DEBUG_NETWORK") != "" {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[NETWORK] updateCache: failed to save cache: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[NETWORK] updateCache: cache saved successfully\n")
		}
	}
}

// InvalidateCache clears the cache for a specific network or all networks
func (r *Resolver) InvalidateCache(networkName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if networkName == "" {
		// Clear entire cache
		r.cache = &ChainIDCache{
			Networks:   make(map[string]uint64),
			RPCs:       make(map[string]uint64),
			ChainNames: make(map[uint64][]string),
			UpdatedAt:  time.Now(),
		}
	} else {
		// Clear specific network
		if chainID, exists := r.cache.Networks[networkName]; exists {
			delete(r.cache.Networks, networkName)
			
			// Remove from chain names
			if names, exists := r.cache.ChainNames[chainID]; exists {
				newNames := []string{}
				for _, name := range names {
					if name != networkName {
						newNames = append(newNames, name)
					}
				}
				if len(newNames) > 0 {
					r.cache.ChainNames[chainID] = newNames
				} else {
					delete(r.cache.ChainNames, chainID)
				}
			}
		}
	}

	// Use internal version since we already hold the lock
	_ = r.saveCacheInternal()
}
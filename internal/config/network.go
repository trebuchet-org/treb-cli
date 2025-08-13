package config

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
)

// NetworkResolver resolves network names to configurations with caching
type NetworkResolver struct {
	projectRoot   string
	foundryConfig *FoundryConfig
	cache         *NetworkCache
	httpClient    *http.Client
	mu            sync.RWMutex
}

// NetworkCache caches chain ID lookups
type NetworkCache struct {
	Networks    map[string]uint64   `json:"networks"`    // name -> chainID
	RPCs        map[string]uint64   `json:"rpcs"`        // rpcURL -> chainID
	ChainNames  map[uint64][]string `json:"chainNames"`  // chainID -> names
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// NewNetworkResolver creates a new network resolver
func NewNetworkResolver(projectRoot string, foundryConfig *FoundryConfig) *NetworkResolver {
	r := &NetworkResolver{
		projectRoot:   projectRoot,
		foundryConfig: foundryConfig,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	
	// Load cache
	r.loadCache()
	
	return r
}

// Resolve resolves a network name to its configuration
func (r *NetworkResolver) Resolve(networkName string) (*NetworkConfig, error) {
	// Check if network exists in foundry.toml
	rpcURL, exists := r.foundryConfig.RpcEndpoints[networkName]
	if !exists {
		return nil, fmt.Errorf("network '%s' not found in foundry.toml [rpc_endpoints]", networkName)
	}

	// Check cache first
	r.mu.RLock()
	chainID, cached := r.cache.Networks[networkName]
	r.mu.RUnlock()

	if !cached {
		// Fetch chain ID from RPC
		fetchedChainID, err := r.fetchChainID(rpcURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chain ID for network %s: %w", networkName, err)
		}
		chainID = fetchedChainID

		// Update cache
		r.updateCache(networkName, rpcURL, chainID)
	}

	// Get explorer URL
	explorer := r.getExplorerURL(networkName, chainID)

	return &NetworkConfig{
		Name:       networkName,
		RpcUrl:     rpcURL,
		ChainID:    chainID,
		Explorer:   explorer,
		Configured: true,
	}, nil
}

// fetchChainID fetches the chain ID from an RPC endpoint
func (r *NetworkResolver) fetchChainID(rpcURL string) (uint64, error) {
	// Check RPC cache first
	r.mu.RLock()
	if chainID, exists := r.cache.RPCs[rpcURL]; exists {
		r.mu.RUnlock()
		return chainID, nil
	}
	r.mu.RUnlock()

	// Prepare eth_chainId JSON-RPC request
	requestBody := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`

	resp, err := r.httpClient.Post(rpcURL, "application/json", strings.NewReader(requestBody))
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

// getExplorerURL returns the explorer URL for a network
func (r *NetworkResolver) getExplorerURL(networkName string, chainID uint64) string {
	// Check if configured in foundry.toml
	if etherscan, exists := r.foundryConfig.Etherscan[networkName]; exists && etherscan.URL != "" {
		return etherscan.URL
	}

	// Fallback to common defaults
	switch chainID {
	case 1:
		return "https://etherscan.io"
	case 5:
		return "https://goerli.etherscan.io"
	case 11155111:
		return "https://sepolia.etherscan.io"
	case 10:
		return "https://optimistic.etherscan.io"
	case 137:
		return "https://polygonscan.com"
	case 8453:
		return "https://basescan.org"
	case 42161:
		return "https://arbiscan.io"
	case 43114:
		return "https://snowtrace.io"
	case 56:
		return "https://bscscan.com"
	case 250:
		return "https://ftmscan.com"
	case 1101:
		return "https://zkevm.polygonscan.com"
	case 324:
		return "https://explorer.zksync.io"
	case 42220:
		return "https://celoscan.io"
	case 44787:
		return "https://alfajores.celoscan.io"
	default:
		return ""
	}
}

// loadCache loads the chain ID cache from disk
func (r *NetworkResolver) loadCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize empty cache
	r.cache = &NetworkCache{
		Networks:   make(map[string]uint64),
		RPCs:       make(map[string]uint64),
		ChainNames: make(map[uint64][]string),
		UpdatedAt:  time.Now(),
	}

	// Try to load from file
	cacheDir := filepath.Join(r.projectRoot, "cache")
	cachePath := filepath.Join(cacheDir, "chainIds.json")
	
	data, err := os.ReadFile(cachePath)
	if err != nil {
		// Cache doesn't exist yet, that's fine
		return
	}

	// Parse cache
	if err := json.Unmarshal(data, &r.cache); err != nil {
		// Invalid cache, start fresh
		r.cache = &NetworkCache{
			Networks:   make(map[string]uint64),
			RPCs:       make(map[string]uint64),
			ChainNames: make(map[uint64][]string),
			UpdatedAt:  time.Now(),
		}
	}
}

// updateCache updates the cache with new chain ID information
func (r *NetworkResolver) updateCache(networkName, rpcURL string, chainID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

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

	// Save to disk (ignore errors, cache is just for performance)
	r.saveCache()
}

// saveCache saves the cache to disk
func (r *NetworkResolver) saveCache() error {
	cacheDir := filepath.Join(r.projectRoot, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, "chainIds.json")
	data, err := json.MarshalIndent(r.cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}
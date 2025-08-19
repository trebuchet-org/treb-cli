package network

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Resolver handles network configuration resolution
type Resolver struct {
	networks      map[string]*domain.Network
	chainIDLookup map[uint64]string // chainID -> network name
}

// NewResolver creates a new network resolver
func NewResolver() *Resolver {
	r := &Resolver{
		networks:      make(map[string]*domain.Network),
		chainIDLookup: make(map[uint64]string),
	}

	// Initialize with default networks
	r.initializeDefaultNetworks()

	return r
}

// initializeDefaultNetworks sets up well-known networks
func (r *Resolver) initializeDefaultNetworks() {
	defaultNetworks := []domain.Network{
		{ChainID: 1, Name: "mainnet", RPCURL: "", ExplorerURL: "https://etherscan.io"},
		{ChainID: 11155111, Name: "sepolia", RPCURL: "", ExplorerURL: "https://sepolia.etherscan.io"},
		{ChainID: 10, Name: "optimism", RPCURL: "", ExplorerURL: "https://optimistic.etherscan.io"},
		{ChainID: 42161, Name: "arbitrum", RPCURL: "", ExplorerURL: "https://arbiscan.io"},
		{ChainID: 137, Name: "polygon", RPCURL: "", ExplorerURL: "https://polygonscan.com"},
		{ChainID: 8453, Name: "base", RPCURL: "", ExplorerURL: "https://basescan.org"},
		{ChainID: 43114, Name: "avalanche", RPCURL: "", ExplorerURL: "https://snowtrace.io"},
		{ChainID: 250, Name: "fantom", RPCURL: "", ExplorerURL: "https://ftmscan.com"},
		{ChainID: 56, Name: "bsc", RPCURL: "", ExplorerURL: "https://bscscan.com"},
		{ChainID: 42220, Name: "celo", RPCURL: "", ExplorerURL: "https://celoscan.io"},
		{ChainID: 31337, Name: "localhost", RPCURL: "http://localhost:8545", ExplorerURL: ""},
		{ChainID: 31337, Name: "anvil", RPCURL: "http://localhost:8545", ExplorerURL: ""},
	}

	for _, network := range defaultNetworks {
		r.addNetwork(&network)
	}
}

// addNetwork adds a network configuration
func (r *Resolver) addNetwork(network *domain.Network) {
	r.networks[network.Name] = network
	r.networks[strings.ToLower(network.Name)] = network // Case-insensitive lookup
	r.chainIDLookup[network.ChainID] = network.Name
}

// LoadNetworks loads additional network configurations
func (r *Resolver) LoadNetworks(networks map[string]*domain.Network) {
	for name, network := range networks {
		// Ensure name is set
		if network.Name == "" {
			network.Name = name
		}
		r.addNetwork(network)
	}
}

// ResolveNetwork resolves a network by name, chain ID, or RPC URL
func (r *Resolver) ResolveNetwork(ctx context.Context, input string) (*domain.Network, error) {
	// Empty input
	if input == "" {
		return nil, fmt.Errorf("network not specified")
	}

	// Direct name lookup
	if network, ok := r.networks[input]; ok {
		return network, nil
	}

	// Case-insensitive name lookup
	if network, ok := r.networks[strings.ToLower(input)]; ok {
		return network, nil
	}

	// Try to parse as chain ID
	if chainID, err := strconv.ParseUint(input, 10, 64); err == nil {
		if name, ok := r.chainIDLookup[chainID]; ok {
			return r.networks[name], nil
		}
		// Create ad-hoc network for unknown chain ID
		return &domain.Network{
			ChainID: chainID,
			Name:    fmt.Sprintf("chain-%d", chainID),
			RPCURL:  "",
		}, nil
	}

	// Check if it's an RPC URL
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") || strings.HasPrefix(input, "ws://") || strings.HasPrefix(input, "wss://") {
		// Create ad-hoc network for RPC URL
		return &domain.Network{
			Name:   "custom",
			RPCURL: input,
		}, nil
	}

	return nil, fmt.Errorf("unknown network: %s", input)
}

// GetNetworkByChainID retrieves a network by its chain ID
func (r *Resolver) GetNetworkByChainID(chainID uint64) (*domain.Network, error) {
	if name, ok := r.chainIDLookup[chainID]; ok {
		return r.networks[name], nil
	}

	// Return a generic network for unknown chain IDs
	return &domain.Network{
		ChainID: chainID,
		Name:    fmt.Sprintf("chain-%d", chainID),
	}, nil
}

// GetPreferredNetwork returns the preferred network name for a chain ID
func (r *Resolver) GetPreferredNetwork(chainID uint64) (string, error) {
	if name, ok := r.chainIDLookup[chainID]; ok {
		return name, nil
	}
	return "", fmt.Errorf("no preferred network for chain ID %d", chainID)
}

// ListNetworks returns all configured networks
func (r *Resolver) ListNetworks() []*domain.Network {
	// Deduplicate networks (since we store both original and lowercase)
	seen := make(map[uint64]bool)
	var networks []*domain.Network

	for _, network := range r.networks {
		if !seen[network.ChainID] {
			networks = append(networks, network)
			seen[network.ChainID] = true
		}
	}

	return networks
}

// GetExplorerURL returns the explorer URL for a network
func (r *Resolver) GetExplorerURL(network string) (string, error) {
	net, err := r.ResolveNetwork(context.Background(), network)
	if err != nil {
		return "", err
	}

	if net.ExplorerURL == "" {
		return "", fmt.Errorf("no explorer URL configured for network %s", net.Name)
	}

	return net.ExplorerURL, nil
}

// GetRPCURL returns the RPC URL for a network
func (r *Resolver) GetRPCURL(network string) (string, error) {
	net, err := r.ResolveNetwork(context.Background(), network)
	if err != nil {
		return "", err
	}

	if net.RPCURL == "" {
		return "", fmt.Errorf("no RPC URL configured for network %s", net.Name)
	}

	return net.RPCURL, nil
}

// NetworkConfig holds the complete network configuration
type NetworkConfig struct {
	Networks         map[string]*domain.Network
	RPCEndpoints     map[string]string
	EtherscanAPIKeys map[string]string
}

// LoadFromConfig loads network configuration from various sources
func (r *Resolver) LoadFromConfig(config *NetworkConfig) error {
	// Load network definitions
	if config.Networks != nil {
		r.LoadNetworks(config.Networks)
	}

	// Update RPC URLs from endpoints
	for name, rpcURL := range config.RPCEndpoints {
		if network, ok := r.networks[name]; ok {
			network.RPCURL = rpcURL
		} else {
			// Create new network entry
			r.addNetwork(&domain.Network{
				Name:   name,
				RPCURL: rpcURL,
			})
		}
	}

	return nil
}


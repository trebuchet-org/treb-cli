package network

import (
	"fmt"
	"strings"
)

// GetNetworkDisplay returns a user-friendly display of a network
func GetNetworkDisplay(networkInfo *NetworkInfo) string {
	return fmt.Sprintf("%s (chain ID: %d)", networkInfo.Name, networkInfo.ChainID)
}

// GetNetworkByChainID attempts to find a network name for a given chain ID
func GetNetworkByChainID(projectRoot string, chainID uint64) (string, error) {
	resolver, err := NewResolver(projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to create resolver: %w", err)
	}

	return resolver.GetPreferredNetwork(chainID)
}

// ListAvailableNetworks returns all networks configured in foundry.toml
func ListAvailableNetworks(projectRoot string) ([]string, error) {
	resolver, err := NewResolver(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	return resolver.GetNetworks(), nil
}

// ValidateNetwork checks if a network exists in foundry.toml
func ValidateNetwork(projectRoot string, networkName string) error {
	networks, err := ListAvailableNetworks(projectRoot)
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network == networkName {
			return nil
		}
	}

	// Provide helpful error message with available networks
	return fmt.Errorf("network '%s' not found in foundry.toml. Available networks: %s", 
		networkName, strings.Join(networks, ", "))
}
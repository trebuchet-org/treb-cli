package network

// GetEtherscanConfig returns the Etherscan configuration for a network if configured
func (r *Resolver) GetEtherscanConfig(networkName string) (url string, apiKey string, configured bool) {
	if r.foundryConfig.Etherscan != nil {
		if etherscan, exists := r.foundryConfig.Etherscan[networkName]; exists {
			// Return the configured etherscan with expanded environment variables
			url = ""
			if etherscan.URL != "" {
				url = r.expandEnvVars(etherscan.URL)
			}
			apiKey = ""
			if etherscan.Key != "" {
				apiKey = r.expandEnvVars(etherscan.Key)
			}
			return url, apiKey, true
		}
	}
	return "", "", false
}

// GetExplorerAPIKey returns the API key for a network's explorer if configured
func (r *Resolver) GetExplorerAPIKey(networkName string) string {
	if r.foundryConfig.Etherscan != nil {
		if etherscan, exists := r.foundryConfig.Etherscan[networkName]; exists && etherscan.Key != "" {
			// Expand environment variables in API key
			return r.expandEnvVars(etherscan.Key)
		}
	}

	// Fallback to common environment variable names
	// This maintains backward compatibility with existing setups
	switch networkName {
	case "mainnet", "sepolia", "goerli":
		return r.expandEnvVars("${ETHERSCAN_API_KEY}")
	case "optimism":
		return r.expandEnvVars("${OPTIMISM_ETHERSCAN_API_KEY}")
	case "arbitrum":
		return r.expandEnvVars("${ARBISCAN_API_KEY}")
	case "polygon":
		return r.expandEnvVars("${POLYGONSCAN_API_KEY}")
	case "base":
		return r.expandEnvVars("${BASESCAN_API_KEY}")
	default:
		// Try generic ETHERSCAN_API_KEY as fallback
		return r.expandEnvVars("${ETHERSCAN_API_KEY}")
	}
}

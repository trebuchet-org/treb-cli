package network

// GetChainID returns the chain ID for a given network name
func GetChainID(network string) uint64 {
	chainIDs := map[string]uint64{
		"mainnet":     1,
		"ethereum":    1,
		"sepolia":     11155111,
		"goerli":      5,
		"polygon":     137,
		"mumbai":      80001,
		"arbitrum":    42161,
		"arbitrum-sepolia": 421614,
		"optimism":    10,
		"optimism-sepolia": 11155420,
		"base":        8453,
		"base-sepolia": 84532,
		"basesepolia": 84532,
		"avalanche":   43114,
		"avalanche-fuji": 43113,
		"bsc":         56,
		"bsc-testnet": 97,
		"gnosis":      100,
		"local":       31337,
		"anvil":       31337,
		"hardhat":     31337,
	}

	if chainID, exists := chainIDs[network]; exists {
		return chainID
	}

	// Default to local network
	return 31337
}
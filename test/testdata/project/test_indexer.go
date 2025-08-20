package main

import (
	"fmt"
	"github.com/trebuchet-org/treb-cli/internal/adapters/contracts"
)

func main() {
	indexer := contracts.NewInternalIndexer(".")
	if err := indexer.Index(); err != nil {
		fmt.Printf("Failed to index: %v\n", err)
		return
	}

	// Try different key formats
	keys := []string{
		"Counter",
		"src/Counter.sol:Counter",
		"Counter.sol:Counter",
	}

	for _, key := range keys {
		contract, err := indexer.GetContract(key)
		if err != nil {
			fmt.Printf("Not found: %s\n", key)
		} else {
			fmt.Printf("Found: %s => %s\n", key, contract.Path)
		}
	}

	// List all contracts
	fmt.Println("\nAll indexed contracts:")
	for _, infos := range indexer.GetAllContracts() {
		for _, info := range infos {
			fmt.Printf("  %s:%s\n", info.Path, info.Name)
		}
	}
}

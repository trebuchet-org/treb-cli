package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// Test different possible event signatures
	signatures := []string{
		"ContractDeployed(address,address,bytes32,bytes32,bytes32,bytes32,bytes,string)",
		"ContractDeployed(address indexed,address indexed,bytes32 indexed,bytes32,bytes32,bytes32,bytes,string)",
		"ContractDeployed(address indexed deployer,address indexed location,bytes32 indexed bundleId,bytes32 salt,bytes32 bytecodeHash,bytes32 initCodeHash,bytes constructorArgs,string createStrategy)",
	}
	
	target := "0x76961918061572de13392698f5554c60d00bd312650938636371e9a356dc0f9e"
	
	for _, sig := range signatures {
		hash := crypto.Keccak256Hash([]byte(sig))
		fmt.Printf("Signature: %s\n", sig)
		fmt.Printf("Hash:      %s\n", hash.Hex())
		fmt.Printf("Match:     %t\n\n", hash.Hex() == target)
	}
}
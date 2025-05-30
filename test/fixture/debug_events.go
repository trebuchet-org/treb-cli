package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	// Calculate topic hashes
	deployingContract := crypto.Keccak256Hash([]byte("DeployingContract(string,string,bytes32)"))
	contractDeployed := crypto.Keccak256Hash([]byte("ContractDeployed(address,address,bytes32,bytes32,bytes32,bytes,string)"))
	
	fmt.Printf("DeployingContract topic: %s\n", deployingContract.Hex())
	fmt.Printf("ContractDeployed topic: %s\n", contractDeployed.Hex())
	
	// Test topic from our logs
	testTopic := common.HexToHash("0x8f4dc992827388d6f9546363611ad0a09a82022d386aa9110c021a68bbc2a3e9")
	fmt.Printf("Test topic: %s\n", testTopic.Hex())
	fmt.Printf("Matches ContractDeployed: %v\n", contractDeployed == testTopic)
}
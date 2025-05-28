package config

import (
	"crypto/ecdsa"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// GetLedgerAddress gets the address for a given derivation path using cast
func GetLedgerAddress(derivationPath string) (string, error) {
	// Use cast to get the address from the ledger
	cmd := exec.Command("cast", "wallet", "address", "--ledger", "--mnemonic-derivation-path", derivationPath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get ledger address: %w (output: %s)", err, string(output))
	}
	
	address := strings.TrimSpace(string(output))
	
	// Basic validation - should be a valid Ethereum address
	if !strings.HasPrefix(address, "0x") || len(address) != 42 {
		return "", fmt.Errorf("invalid address returned from ledger: %s", address)
	}
	
	return address, nil
}

// GetAddressFromPrivateKey derives an Ethereum address from a private key
func GetAddressFromPrivateKey(privateKeyHex string) (string, error) {
	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	
	// Parse the private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	
	// Get the public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to cast public key to ECDSA")
	}
	
	// Derive the address
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	
	return address.Hex(), nil
}
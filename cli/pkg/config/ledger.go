package config

import (
	"fmt"
	"os/exec"
	"strings"
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
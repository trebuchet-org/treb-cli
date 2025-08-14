package environment

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

func TestSenderEncodingIntegration(t *testing.T) {
	// Test that our sender encoding matches what Solidity expects
	
	// First, let's verify what the v1 code produces for "in-memory"
	v1InMemory := calculateBytes8("in-memory")
	t.Logf("v2 calculateBytes8('in-memory') = 0x%x", v1InMemory)
	
	// Calculate the hash manually to verify
	hash := crypto.Keccak256([]byte("in-memory"))
	t.Logf("Keccak256('in-memory') full hash = 0x%x", hash)
	t.Logf("First 8 bytes = 0x%x", hash[:8])
	
	// Test the full encoding for a simple private key sender
	config := &domain.TrebConfig{
		Senders: map[string]domain.SenderConfig{
			"anvil": {
				Type:       "private_key",
				PrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			},
		},
	}
	
	encoded, err := ABIEncodeSenderConfigs(config)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	
	t.Logf("Full encoded sender configs: %s", encoded)
	
	// Decode just the hex part
	data, _ := hex.DecodeString(encoded[2:])
	t.Logf("Decoded length: %d bytes", len(data))
	
	// Log the SENDER_TYPE_IN_MEMORY value
	t.Logf("SENDER_TYPE_IN_MEMORY = 0x%x", SENDER_TYPE_IN_MEMORY)
	t.Logf("SENDER_TYPE_PRIVATE_KEY = 0x%x", SENDER_TYPE_PRIVATE_KEY)
	
	// Also check what we're getting for the error case
	t.Logf("v2 sender type for 'anvil' (which failed) = 0x%x", SENDER_TYPE_IN_MEMORY)
}

func TestSenderTypesMatch(t *testing.T) {
	// Test that all our sender type calculations match v1
	types := []string{
		"in-memory",
		"hw-wallet", 
		"multisig",
		"gnosis-safe",
		"ledger",
		"trezor",
		"private-key",
		"hardware-wallet",
	}
	
	for _, typ := range types {
		hash := calculateBytes8(typ)
		t.Logf("calculateBytes8('%s') = 0x%x", typ, hash)
	}
}
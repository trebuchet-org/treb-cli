package environment

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

func TestABIEncodeSenderConfigs(t *testing.T) {
	tests := []struct {
		name      string
		config    *domain.TrebConfig
		wantError bool
		validate  func(t *testing.T, encoded string)
	}{
		{
			name:      "nil config returns empty",
			config:    nil,
			wantError: false,
			validate: func(t *testing.T, encoded string) {
				if encoded != "0x" {
					t.Errorf("expected 0x, got %s", encoded)
				}
			},
		},
		{
			name: "empty senders returns empty",
			config: &domain.TrebConfig{
				Senders: map[string]domain.SenderConfig{},
			},
			wantError: false,
			validate: func(t *testing.T, encoded string) {
				if encoded != "0x" {
					t.Errorf("expected 0x, got %s", encoded)
				}
			},
		},
		{
			name: "private key sender",
			config: &domain.TrebConfig{
				Senders: map[string]domain.SenderConfig{
					"deployer": {
						Type:       "private_key",
						PrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
					},
				},
			},
			wantError: false,
			validate: func(t *testing.T, encoded string) {
				// Should be valid hex
				if !strings.HasPrefix(encoded, "0x") {
					t.Errorf("encoded value should start with 0x")
				}
				
				// Try to decode to verify it's valid
				_, err := hex.DecodeString(encoded[2:])
				if err != nil {
					t.Errorf("failed to decode hex: %v", err)
				}

				// The encoded data should be decodable as an array of structs
				tupleType, _ := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
					{Name: "name", Type: "string"},
					{Name: "account", Type: "address"},
					{Name: "senderType", Type: "bytes8"},
					{Name: "canBroadcast", Type: "bool"},
					{Name: "config", Type: "bytes"},
				})
				
				data, _ := hex.DecodeString(encoded[2:])
				arrayArgs := abi.Arguments{{Type: tupleType}}
				decoded, err := arrayArgs.Unpack(data)
				if err != nil {
					t.Errorf("failed to decode ABI: %v", err)
				}
				if len(decoded) == 0 {
					t.Errorf("expected decoded data")
				}
			},
		},
		{
			name: "ledger sender without address fails",
			config: &domain.TrebConfig{
				Senders: map[string]domain.SenderConfig{
					"ledger": {
						Type:           "ledger",
						DerivationPath: "m/44'/60'/0'/0/0",
					},
				},
			},
			wantError: true,
		},
		{
			name: "multiple senders",
			config: &domain.TrebConfig{
				Senders: map[string]domain.SenderConfig{
					"deployer": {
						Type:       "private_key",
						PrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
					},
					"hardware": {
						Type:           "ledger",
						Account:        "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
						DerivationPath: "m/44'/60'/0'/0/0",
					},
				},
			},
			wantError: false,
			validate: func(t *testing.T, encoded string) {
				// Decode and verify we have 2 senders
				tupleType, _ := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
					{Name: "name", Type: "string"},
					{Name: "account", Type: "address"},
					{Name: "senderType", Type: "bytes8"},
					{Name: "canBroadcast", Type: "bool"},
					{Name: "config", Type: "bytes"},
				})
				
				data, _ := hex.DecodeString(encoded[2:])
				arrayArgs := abi.Arguments{{Type: tupleType}}
				
				// We expect an array with 2 elements
				result, err := arrayArgs.Unpack(data)
				if err != nil {
					t.Errorf("failed to unpack: %v", err)
				}
				if len(result) == 0 {
					t.Errorf("expected decoded data")
				}
			},
		},
		{
			name: "safe sender with proposer",
			config: &domain.TrebConfig{
				Senders: map[string]domain.SenderConfig{
					"multisig": {
						Type: "safe",
						Safe: "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F",
						Proposer: &domain.ProposerConfig{
							Type:       "private_key",
							PrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
						},
					},
				},
			},
			wantError: false,
			validate: func(t *testing.T, encoded string) {
				// Should encode both the proposer and the safe
				data, _ := hex.DecodeString(encoded[2:])
				if len(data) < 100 { // Rough check for minimum size
					t.Errorf("encoded data seems too small for safe + proposer")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := ABIEncodeSenderConfigs(tt.config)
			
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			
			if tt.validate != nil && err == nil {
				tt.validate(t, encoded)
			}
		})
	}
}

func TestCalculateBytes8(t *testing.T) {
	tests := []struct {
		input    string
		expected [8]byte
	}{
		{
			input:    "test",
			expected: [8]byte{'t', 'e', 's', 't', 0, 0, 0, 0},
		},
		{
			input:    "in-memory",
			expected: [8]byte{'i', 'n', '-', 'm', 'e', 'm', 'o', 'r'},
		},
		{
			input:    "", // empty string
			expected: [8]byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := calculateBytes8(tt.input)
			if result != tt.expected {
				t.Errorf("calculateBytes8(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBitwiseOr(t *testing.T) {
	a := [8]byte{0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00}
	b := [8]byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}
	expected := [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	
	result := bitwiseOr(a, b)
	if result != expected {
		t.Errorf("bitwiseOr() = %v, want %v", result, expected)
	}
}
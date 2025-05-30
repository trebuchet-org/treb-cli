package abi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestFormatTokenAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   *big.Int
		expected string
	}{
		{
			name:     "nil amount",
			amount:   nil,
			expected: "0",
		},
		{
			name:     "zero amount",
			amount:   big.NewInt(0),
			expected: "0",
		},
		{
			name:     "1 token with 18 decimals",
			amount:   new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			expected: "1 * 10^18",
		},
		{
			name:     "1000 tokens with 18 decimals",
			amount:   new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			expected: "1,000 * 10^18",
		},
		{
			name:     "1 token with 6 decimals",
			amount:   new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil),
			expected: "1 * 10^6",
		},
		{
			name:     "non-standard amount",
			amount:   big.NewInt(123456789),
			expected: "123,456,789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTokenAmount(tt.amount)
			if result != tt.expected {
				t.Errorf("FormatTokenAmount(%v) = %s, want %s", tt.amount, result, tt.expected)
			}
		})
	}
}

func TestDecodeConstructorArgs_CommonPatterns(t *testing.T) {
	parser := NewParser(".")
	
	tests := []struct {
		name         string
		contractName string
		args         []byte
		expected     string
		shouldError  bool
	}{
		{
			name:         "empty args",
			contractName: "Counter",
			args:         []byte{},
			expected:     "",
			shouldError:  false,
		},
		{
			name:         "simple address pattern",
			contractName: "Ownable",
			args:         common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(),
			expected:     "0x1234567890123456789012345678901234567890",
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.DecodeConstructorArgs(tt.contractName, tt.args)
			
			if tt.shouldError && err == nil {
				t.Errorf("DecodeConstructorArgs() expected error but got none")
			}
			
			if !tt.shouldError && err != nil {
				t.Errorf("DecodeConstructorArgs() unexpected error: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("DecodeConstructorArgs() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestAddCommas(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"1234", "1,234"},
		{"1234567", "1,234,567"},
		{"1234567890", "1,234,567,890"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := addCommas(tt.input)
			if result != tt.expected {
				t.Errorf("addCommas(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
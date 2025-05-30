package script

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

// SenderInitConfig represents a single sender configuration for the new Senders library
// This matches the Solidity SenderInitConfig struct in Senders.sol
type SenderInitConfig struct {
	Name       string
	Account    common.Address
	SenderType [8]byte // bytes8 magic constant
	Config     []byte  // ABI-encoded config data
}

// SenderConfigs represents the complete array of sender configurations
type SenderConfigs struct {
	Configs []SenderInitConfig
}

// Magic constants for sender types - these match the constants in Senders.sol
var (
	// Base types
	SENDER_TYPE_CUSTOM          = calculateBytes8("custom")
	SENDER_TYPE_PRIVATE_KEY     = calculateBytes8("private-key")
	SENDER_TYPE_MULTISIG        = calculateBytes8("multisig")
	SENDER_TYPE_HARDWARE_WALLET = bitwiseOr(calculateBytes8("hardware-wallet"), SENDER_TYPE_PRIVATE_KEY)

	// Composite types
	SENDER_TYPE_IN_MEMORY   = bitwiseOr(calculateBytes8("in-memory"), SENDER_TYPE_PRIVATE_KEY)
	SENDER_TYPE_GNOSIS_SAFE = bitwiseOr(SENDER_TYPE_MULTISIG, calculateBytes8("gnosis-safe"))
	SENDER_TYPE_LEDGER      = bitwiseOr(calculateBytes8("ledger"), SENDER_TYPE_HARDWARE_WALLET)
	SENDER_TYPE_TREZOR      = bitwiseOr(calculateBytes8("trezor"), SENDER_TYPE_HARDWARE_WALLET)
)

// BuildSenderConfigs builds sender configurations from the treb config
func BuildSenderConfigs(trebConfig *config.TrebConfig) (*SenderConfigs, error) {
	configs := &SenderConfigs{
		Configs: []SenderInitConfig{},
	}

	// Debug: check if senders exist
	if trebConfig.Senders == nil || len(trebConfig.Senders) == 0 {
		return nil, fmt.Errorf("no senders configured in profile")
	}

	// Get sorted sender IDs for consistent ordering
	var senderIDs []string
	for id := range trebConfig.Senders {
		senderIDs = append(senderIDs, id)
	}
	sort.Strings(senderIDs)

	// Process all senders from the profile in sorted order
	for _, id := range senderIDs {
		sender := trebConfig.Senders[id]
		senderConfig, err := buildSenderInitConfig(id, sender, trebConfig.Senders)
		if err != nil {
			return nil, fmt.Errorf("failed to build sender %s: %w", id, err)
		}

		configs.Configs = append(configs.Configs, *senderConfig)
	}

	return configs, nil
}

// buildSenderInitConfig builds a single sender configuration using the new format
func buildSenderInitConfig(id string, sender config.SenderConfig, allSenders map[string]config.SenderConfig) (*SenderInitConfig, error) {
	switch sender.Type {
	case "private_key":
		// Parse private key to get address
		key, err := parsePrivateKey(sender.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}

		// For private key senders, config contains the private key as uint256
		uint256Type, _ := abi.NewType("uint256", "", nil)
		args := abi.Arguments{{Type: uint256Type}}
		configData, err := args.Pack(key.PrivateKey.D)
		if err != nil {
			return nil, fmt.Errorf("failed to encode private key config: %w", err)
		}

		return &SenderInitConfig{
			Name:       id,
			Account:    key.Address,
			SenderType: SENDER_TYPE_IN_MEMORY, // Use in-memory for private key senders
			Config:     configData,
		}, nil

	case "safe":
		// Parse Safe address
		safe := common.HexToAddress(sender.Safe)
		if safe == (common.Address{}) {
			return nil, fmt.Errorf("invalid Safe address: %s", sender.Safe)
		}

		// Validate signer is provided
		if sender.Signer == "" {
			return nil, fmt.Errorf("Safe sender requires a signer (proposer) to be specified")
		}

		// Validate that the signer exists in the sender configs
		if _, exists := allSenders[sender.Signer]; !exists {
			return nil, fmt.Errorf("Safe signer '%s' not found in sender configurations", sender.Signer)
		}

		// For Safe senders, config contains the proposer name as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.Signer)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Safe config: %w", err)
		}

		return &SenderInitConfig{
			Name:       id,
			Account:    safe,
			SenderType: SENDER_TYPE_GNOSIS_SAFE,
			Config:     configData,
		}, nil

	case "ledger":
		// For hardware wallets, we can't derive the address without the device
		// We'll use a zero address as placeholder - the hardware wallet will provide the actual address
		placeholderAddress := common.Address{}

		// For hardware wallet senders, config contains the derivation path as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.DerivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Ledger config: %w", err)
		}

		return &SenderInitConfig{
			Name:       id,
			Account:    placeholderAddress,
			SenderType: SENDER_TYPE_LEDGER,
			Config:     configData,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported sender type: %s", sender.Type)
	}
}

// EncodeSenderConfigs encodes the sender configs for passing as environment variable
func EncodeSenderConfigs(configs *SenderConfigs) (string, error) {
	// Use standard ABI encoding for array of structs - this is much more reliable than manual encoding
	tupleType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "name", Type: "string"},
		{Name: "account", Type: "address"},
		{Name: "senderType", Type: "bytes8"},
		{Name: "config", Type: "bytes"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create tuple type: %w", err)
	}

	arrayArgs := abi.Arguments{{Type: tupleType}}
	encoded, err := arrayArgs.Pack(configs.Configs)
	if err != nil {
		return "", fmt.Errorf("failed to encode sender configs: %w", err)
	}

	return "0x" + hex.EncodeToString(encoded), nil
}

// Helper functions for magic constants

// calculateBytes8 calculates the bytes8 hash for a type string (first 8 bytes of keccak256)
func calculateBytes8(typeString string) [8]byte {
	hash := crypto.Keccak256([]byte(typeString))
	var result [8]byte
	copy(result[:], hash[:8])
	return result
}

// bitwiseOr performs bitwise OR on two bytes8 values
func bitwiseOr(a, b [8]byte) [8]byte {
	var result [8]byte
	for i := 0; i < 8; i++ {
		result[i] = a[i] | b[i]
	}
	return result
}

// Helper functions for encoding constructor arguments

// parsePrivateKey parses a private key string and returns the address
func parsePrivateKey(privateKeyHex string) (*struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}, error) {
	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Decode hex string
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Create private key from bytes
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key: %w", err)
	}

	// Get address from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to get public key")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &struct {
		PrivateKey *ecdsa.PrivateKey
		Address    common.Address
	}{
		PrivateKey: privateKey,
		Address:    address,
	}, nil
}

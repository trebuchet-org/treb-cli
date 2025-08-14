package environment

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Magic sender type constants matching Senders.sol
// Note: These must be calculated at runtime since they depend on each other
var (
	SENDER_TYPE_CUSTOM          [8]byte
	SENDER_TYPE_PRIVATE_KEY     [8]byte
	SENDER_TYPE_MULTISIG        [8]byte
	SENDER_TYPE_HARDWARE_WALLET [8]byte
	SENDER_TYPE_IN_MEMORY       [8]byte
	SENDER_TYPE_GNOSIS_SAFE     [8]byte
	SENDER_TYPE_LEDGER          [8]byte
	SENDER_TYPE_TREZOR          [8]byte
)

func init() {
	// Base types
	SENDER_TYPE_CUSTOM = calculateBytes8("custom")
	SENDER_TYPE_PRIVATE_KEY = calculateBytes8("private-key")
	SENDER_TYPE_MULTISIG = calculateBytes8("multisig")
	SENDER_TYPE_HARDWARE_WALLET = bitwiseOr(calculateBytes8("hardware-wallet"), SENDER_TYPE_PRIVATE_KEY)

	// Composite types
	SENDER_TYPE_IN_MEMORY = bitwiseOr(calculateBytes8("in-memory"), SENDER_TYPE_PRIVATE_KEY)
	SENDER_TYPE_GNOSIS_SAFE = bitwiseOr(SENDER_TYPE_MULTISIG, calculateBytes8("gnosis-safe"))
	SENDER_TYPE_LEDGER = bitwiseOr(calculateBytes8("ledger"), SENDER_TYPE_HARDWARE_WALLET)
	SENDER_TYPE_TREZOR = bitwiseOr(calculateBytes8("trezor"), SENDER_TYPE_HARDWARE_WALLET)
}

// SenderInitConfig represents a sender configuration matching Solidity struct
type SenderInitConfig struct {
	Name         string
	Account      common.Address
	SenderType   [8]byte
	CanBroadcast bool
	Config       []byte
}

// ABIEncodeSenderConfigs properly ABI encodes sender configurations
func ABIEncodeSenderConfigs(trebConfig *domain.TrebConfig) (string, error) {
	if trebConfig == nil || len(trebConfig.Senders) == 0 {
		return "0x", nil
	}

	var configs []SenderInitConfig
	proposerSenders := make(map[string]domain.SenderConfig)

	// Process senders in deterministic order
	var senderNames []string
	for name := range trebConfig.Senders {
		senderNames = append(senderNames, name)
	}
	// Sort for consistent ordering
	for i := 0; i < len(senderNames); i++ {
		for j := i + 1; j < len(senderNames); j++ {
			if senderNames[i] > senderNames[j] {
				senderNames[i], senderNames[j] = senderNames[j], senderNames[i]
			}
		}
	}

	// First pass: build non-Safe senders and collect proposers
	var safeSenders []string
	for _, name := range senderNames {
		sender := trebConfig.Senders[name]
		if sender.Type == "safe" {
			safeSenders = append(safeSenders, name)
			// If this safe has a proposer, add it to our proposer map
			if sender.Proposer != nil {
				proposerName := fmt.Sprintf("%s_proposer", name)
				proposerSender := domain.SenderConfig{
					Type:           sender.Proposer.Type,
					PrivateKey:     sender.Proposer.PrivateKey,
					DerivationPath: sender.Proposer.DerivationPath,
				}
				proposerSenders[proposerName] = proposerSender
			}
			continue
		}

		config, err := buildSenderInitConfig(name, sender, trebConfig.Senders)
		if err != nil {
			return "", fmt.Errorf("failed to build config for sender %s: %w", name, err)
		}
		configs = append(configs, *config)
	}

	// Add proposer senders
	for name, sender := range proposerSenders {
		config, err := buildSenderInitConfig(name, sender, trebConfig.Senders)
		if err != nil {
			return "", fmt.Errorf("failed to build config for proposer %s: %w", name, err)
		}
		configs = append(configs, *config)
	}

	// Second pass: build Safe senders (after their signers are registered)
	for _, name := range safeSenders {
		sender := trebConfig.Senders[name]
		config, err := buildSenderInitConfig(name, sender, trebConfig.Senders)
		if err != nil {
			return "", fmt.Errorf("failed to build config for safe sender %s: %w", name, err)
		}
		configs = append(configs, *config)
	}

	// ABI encode the array of structs
	tupleType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "name", Type: "string"},
		{Name: "account", Type: "address"},
		{Name: "senderType", Type: "bytes8"},
		{Name: "canBroadcast", Type: "bool"},
		{Name: "config", Type: "bytes"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create tuple type: %w", err)
	}

	arrayArgs := abi.Arguments{{Type: tupleType}}
	encoded, err := arrayArgs.Pack(configs)
	if err != nil {
		return "", fmt.Errorf("failed to encode sender configs: %w", err)
	}

	return "0x" + common.Bytes2Hex(encoded), nil
}

// buildSenderInitConfig builds a single sender configuration
func buildSenderInitConfig(name string, sender domain.SenderConfig, allSenders map[string]domain.SenderConfig) (*SenderInitConfig, error) {
	switch sender.Type {
	case "private_key":
		// Parse private key
		privateKey := strings.TrimPrefix(sender.PrivateKey, "0x")
		key, err := crypto.HexToECDSA(privateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}

		address := crypto.PubkeyToAddress(key.PublicKey)

		// Encode private key as uint256
		uint256Type, _ := abi.NewType("uint256", "", nil)
		args := abi.Arguments{{Type: uint256Type}}
		configData, err := args.Pack(key.D)
		if err != nil {
			return nil, fmt.Errorf("failed to encode private key config: %w", err)
		}

		return &SenderInitConfig{
			Name:         name,
			Account:      address,
			SenderType:   SENDER_TYPE_IN_MEMORY,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	case "ledger":
		if sender.Account == "" {
			return nil, fmt.Errorf("ledger sender requires an account address")
		}

		// Encode derivation path as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.DerivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to encode ledger config: %w", err)
		}

		return &SenderInitConfig{
			Name:         name,
			Account:      common.HexToAddress(sender.Account),
			SenderType:   SENDER_TYPE_LEDGER,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	case "trezor":
		if sender.Account == "" {
			return nil, fmt.Errorf("trezor sender requires an account address")
		}

		// Encode derivation path as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.DerivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to encode trezor config: %w", err)
		}

		return &SenderInitConfig{
			Name:         name,
			Account:      common.HexToAddress(sender.Account),
			SenderType:   SENDER_TYPE_TREZOR,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	case "safe":
		if sender.Safe == "" {
			return nil, fmt.Errorf("safe sender requires a safe address")
		}

		// For v2, we need to handle proposer config
		proposerName := ""
		if sender.Proposer != nil {
			// Find or create a proposer sender name
			// In v1, this is referenced by signer name
			proposerName = fmt.Sprintf("%s_proposer", name)
		} else if sender.Signer != "" {
			// Handle legacy v1 format where signer references another sender
			proposerName = sender.Signer
		}

		if proposerName == "" {
			return nil, fmt.Errorf("safe sender requires a proposer")
		}

		// Encode proposer name as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(proposerName)
		if err != nil {
			return nil, fmt.Errorf("failed to encode safe config: %w", err)
		}

		return &SenderInitConfig{
			Name:         name,
			Account:      common.HexToAddress(sender.Safe),
			SenderType:   SENDER_TYPE_GNOSIS_SAFE,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported sender type: %s", sender.Type)
	}
}

// calculateBytes8 converts a string to bytes8 for magic constants
func calculateBytes8(s string) [8]byte {
	hash := crypto.Keccak256([]byte(s))
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

// parsePrivateKey parses a hex private key and returns the key and address
func parsePrivateKey(privateKeyHex string) (*parsedKey, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}

	address := crypto.PubkeyToAddress(key.PublicKey)
	return &parsedKey{
		PrivateKey: key,
		Address:    address,
	}, nil
}

type parsedKey struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}
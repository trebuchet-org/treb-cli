package config

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// SenderInitConfig represents a single sender configuration for the new Senders library
// This matches the Solidity SenderInitConfig struct in Senders.sol
type SendersManager struct {
	config *config.TrebConfig
}

// Magic constants for sender types - these match the constants in Senders.sol
var (
	// Base types
	SENDER_TYPE_CUSTOM          = calculateBytes8("custom")
	SENDER_TYPE_PRIVATE_KEY     = calculateBytes8("private-key")
	SENDER_TYPE_MULTISIG        = calculateBytes8("multisig")
	SENDER_TYPE_HARDWARE_WALLET = bitwiseOr(calculateBytes8("hardware-wallet"), SENDER_TYPE_PRIVATE_KEY)
	SENDER_TYPE_GOVERNANCE      = calculateBytes8("governance")

	// Composite types
	SENDER_TYPE_IN_MEMORY   = bitwiseOr(calculateBytes8("in-memory"), SENDER_TYPE_PRIVATE_KEY)
	SENDER_TYPE_GNOSIS_SAFE = bitwiseOr(SENDER_TYPE_MULTISIG, calculateBytes8("gnosis-safe"))
	SENDER_TYPE_LEDGER      = bitwiseOr(calculateBytes8("ledger"), SENDER_TYPE_HARDWARE_WALLET)
	SENDER_TYPE_TREZOR      = bitwiseOr(calculateBytes8("trezor"), SENDER_TYPE_HARDWARE_WALLET)
	SENDER_TYPE_OZ_GOVERNOR = bitwiseOr(SENDER_TYPE_GOVERNANCE, calculateBytes8("oz-governor"))
)

func NewSendersManager(config *config.RuntimeConfig) *SendersManager {
	return &SendersManager{
		config: config.TrebConfig,
	}
}

func (m *SendersManager) BuildSenderScriptConfig(
	script *models.Artifact,
) (*config.SenderScriptConfig, error) {
	var err error
	var senders []string
	if senders, err = m.getSendersFromScript(script); err != nil {
		return nil, err

	}
	if len(senders) == 0 {
		return &config.SenderScriptConfig{}, nil
	}

	executionHWConfig := m.getSendersHWConfig(senders)
	if executionHWConfig.UseLedger && executionHWConfig.UseTrezor {
		return nil, fmt.Errorf("can not use ledger and trezor senders in the same script, configure @custom:senders")
	}

	var safeSigners []string
	if m.config != nil && m.config.Senders != nil {
		for _, senderKey := range senders {
			if sender, exists := m.config.Senders[senderKey]; exists && sender.Type == "safe" {
				safeSigners = append(safeSigners, sender.Signer)
			}
		}
	}
	signersHWConfig := m.getSendersHWConfig(safeSigners)

	if signersHWConfig.UseLedger && executionHWConfig.UseLedger {
		return nil, fmt.Errorf("can not use ledger in both main sender and safe signer, configure @custom:senders")
	}

	if signersHWConfig.UseTrezor && executionHWConfig.UseTrezor {
		return nil, fmt.Errorf("can not use ledger in both main sender and safe signer, configure @custom:senders")
	}

	sort.Strings(safeSigners)
	sort.Strings(senders)
	allSenders := append(slices.Clone(safeSigners), senders...)

	var senderInitConfigs []config.SenderInitConfig
	if senderInitConfigs, err = m.buildSenderInitConfigs(allSenders); err != nil {
		return nil, err
	}
	var encodedSenderInitConfigs string
	if encodedSenderInitConfigs, err = m.encodeSenderInitConfigs(senderInitConfigs); err != nil {
		return nil, err
	}

	return &config.SenderScriptConfig{
		UseLedger:         executionHWConfig.UseLedger,
		UseTrezor:         executionHWConfig.UseTrezor,
		DerivationPaths:   executionHWConfig.DerivationPaths,
		EncodedConfig:     encodedSenderInitConfigs,
		SenderInitConfigs: senderInitConfigs,
		Senders:           senders,
	}, nil
}

func (m *SendersManager) getSendersFromScript(script *models.Artifact) ([]string, error) {
	// Extract devdoc from metadata
	var devdoc struct {
		Methods map[string]map[string]any `json:"methods"`
	}

	if err := json.Unmarshal(script.Metadata.Output.DevDoc, &devdoc); err != nil {
		return nil, fmt.Errorf("failed to parse devdoc: %w", err)
	}

	// Look for run() method
	runMethod, exists := devdoc.Methods["run()"]
	if !exists {
		return nil, nil // No run() method found
	}

	// Look for custom:senders tag
	customSenders, exists := runMethod["custom:senders"]
	if !exists {
		return nil, nil // No custom:senders tag found
	}

	sendersStr, ok := customSenders.(string)
	if !ok {
		return nil, fmt.Errorf("custom:senders is not a string")
	}

	var senders []string

	// Split by comma and trim spaces
	parts := strings.Split(sendersStr, ",")

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		// Validate name (alphanumeric and underscore)
		if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name) {
			return nil, fmt.Errorf("invalid sender name: %s", name)
		}

		senders = append(senders, name)
	}

	return senders, nil
}

func (m *SendersManager) getSendersHWConfig(senders []string) config.SenderHWConfig {
	hwConfig := config.SenderHWConfig{
		UseLedger:       false,
		UseTrezor:       false,
		DerivationPaths: []string{},
	}

	// Check if config is nil or has no senders
	if m.config == nil || m.config.Senders == nil {
		return hwConfig
	}

	for _, senderKey := range senders {
		sender, exists := m.config.Senders[senderKey]
		if !exists {
			continue
		}
		hwConfig.UseTrezor = hwConfig.UseTrezor || sender.Type == "trezor"
		hwConfig.UseLedger = hwConfig.UseLedger || sender.Type == "ledger"

		if sender.Type == "trezor" || sender.Type == "ledger" {
			hwConfig.DerivationPaths = append(hwConfig.DerivationPaths, sender.DerivationPath)
		}

	}
	return hwConfig
}

func (m *SendersManager) buildSenderInitConfigs(senders []string) ([]config.SenderInitConfig, error) {
	configs := []config.SenderInitConfig{}
	for _, sender := range senders {
		var config *config.SenderInitConfig
		var err error
		if config, err = m.buildSenderInitConfig(sender); err != nil {
			return nil, err
		}

		configs = append(configs, *config)
	}
	return configs, nil
}

func (m *SendersManager) buildSenderInitConfig(senderKey string) (*config.SenderInitConfig, error) {
	if m.config == nil || m.config.Senders == nil {
		return nil, fmt.Errorf("no sender configuration available")
	}

	sender, exists := m.config.Senders[senderKey]
	if !exists {
		return nil, fmt.Errorf("sender '%s' not found in configuration", senderKey)
	}

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

		return &config.SenderInitConfig{
			Name:         senderKey,
			Account:      key.Address,
			SenderType:   SENDER_TYPE_IN_MEMORY, // Use in-memory for private key senders
			CanBroadcast: true,
			Config:       configData,
			BaseConfig:   sender,
		}, nil

	case "safe":
		// Parse Safe address
		safe := common.HexToAddress(sender.Safe)
		if safe == (common.Address{}) {
			return nil, fmt.Errorf("invalid Safe address: %s", sender.Safe)
		}

		// Validate signer is provided
		if sender.Signer == "" {
			return nil, fmt.Errorf("safe sender requires a signer (proposer) to be specified")
		}

		// Validate that the signer exists in the sender configs
		_, exists := m.config.Senders[sender.Signer]
		if !exists {
			return nil, fmt.Errorf("safe signer '%s' not found in sender configurations", sender.Signer)
		}

		// For Safe senders, config contains the proposer name as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.Signer)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Safe config: %w", err)
		}

		return &config.SenderInitConfig{
			Name:         senderKey,
			Account:      safe,
			SenderType:   SENDER_TYPE_GNOSIS_SAFE,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	case "ledger":
		// Validate address is provided
		if sender.Address == "" {
			return nil, fmt.Errorf("ledger sender requires an address to be specified")
		}

		// Parse the address to ensure it's valid
		address := common.HexToAddress(sender.Address)
		if address == (common.Address{}) {
			return nil, fmt.Errorf("invalid ledger address: %s", sender.Address)
		}

		// For hardware wallet senders, config contains the derivation path as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.DerivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Ledger config: %w", err)
		}

		return &config.SenderInitConfig{
			Name:         senderKey,
			Account:      address,
			SenderType:   SENDER_TYPE_LEDGER,
			CanBroadcast: true,
			Config:       configData,
		}, nil
	case "trezor":
		// Validate address is provided
		if sender.Address == "" {
			return nil, fmt.Errorf("trezor sender requires an address to be specified")
		}

		// For hardware wallet senders, config contains the derivation path as string
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{{Type: stringType}}
		configData, err := args.Pack(sender.DerivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Ledger config: %w", err)
		}

		return &config.SenderInitConfig{
			Name:         senderKey,
			Account:      common.HexToAddress(sender.Address),
			SenderType:   SENDER_TYPE_LEDGER,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	case "oz_governor":
		// Parse Governor address
		governor := common.HexToAddress(sender.Governor)
		if governor == (common.Address{}) {
			return nil, fmt.Errorf("invalid Governor address: %s", sender.Governor)
		}

		// Parse optional Timelock address (zero address if not provided)
		var timelock common.Address
		if sender.Timelock != "" {
			timelock = common.HexToAddress(sender.Timelock)
		}

		// Validate proposer is provided
		if sender.Proposer == "" {
			return nil, fmt.Errorf("oz_governor sender requires a proposer to be specified")
		}

		// Validate that the proposer exists in the sender configs
		_, exists := m.config.Senders[sender.Proposer]
		if !exists {
			return nil, fmt.Errorf("oz_governor proposer '%s' not found in sender configurations", sender.Proposer)
		}

		// Account is timelock if provided, otherwise governor (for correct vm.prank behavior)
		account := governor
		if timelock != (common.Address{}) {
			account = timelock
		}

		// For OZGovernor senders, config contains (address governor, address timelock, string proposerName)
		addressType, _ := abi.NewType("address", "", nil)
		stringType, _ := abi.NewType("string", "", nil)
		args := abi.Arguments{
			{Type: addressType},
			{Type: addressType},
			{Type: stringType},
		}
		configData, err := args.Pack(governor, timelock, sender.Proposer)
		if err != nil {
			return nil, fmt.Errorf("failed to encode OZGovernor config: %w", err)
		}

		return &config.SenderInitConfig{
			Name:         senderKey,
			Account:      account,
			SenderType:   SENDER_TYPE_OZ_GOVERNOR,
			CanBroadcast: true,
			Config:       configData,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported sender type: %s", sender.Type)
	}
}

func (m *SendersManager) encodeSenderInitConfigs(senderInitConfigs []config.SenderInitConfig) (string, error) {
	// Use standard ABI encoding for array of structs - this is much more reliable than manual encoding
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
	encoded, err := arrayArgs.Pack(senderInitConfigs)
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

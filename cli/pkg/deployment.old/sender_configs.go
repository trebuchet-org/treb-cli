package deployment

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

// SenderConfigs represents the configuration for senders
// This matches the Solidity struct in Dispatcher.sol
type SenderConfigs struct {
	Ids             []string
	Artifacts       []string
	ConstructorArgs [][]byte
}

// SenderConfig represents a single sender configuration
type SenderConfig struct {
	ID           string
	Type         string // "private_key", "safe", "ledger"
	Artifact     string
	Args         []byte
}

// PrivateKey represents a parsed private key with its address
type PrivateKey struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}

// Context represents a minimal deployment context for sender configuration
type Context struct {
	Config  *config.Config
	Profile *config.Profile
	Network string
	RPC     string
	DryRun  bool
}

// BuildSenderConfigs builds sender configurations from the deployment context
func (ctx *Context) BuildSenderConfigs() (*SenderConfigs, error) {
	configs := &SenderConfigs{
		Ids:             []string{},
		Artifacts:       []string{},
		ConstructorArgs: [][]byte{},
	}
	
	// Always add a default sender based on the profile
	defaultSender, err := ctx.buildDefaultSender()
	if err != nil {
		return nil, fmt.Errorf("failed to build default sender: %w", err)
	}
	
	configs.Ids = append(configs.Ids, defaultSender.ID)
	configs.Artifacts = append(configs.Artifacts, defaultSender.Artifact)
	configs.ConstructorArgs = append(configs.ConstructorArgs, defaultSender.Args)
	
	// Add additional senders if needed (e.g., for Safe proposer)
	if ctx.Profile.Deployer.Type == "safe" && ctx.Profile.Deployer.Proposer != nil {
		proposerSender, err := ctx.buildProposerSender()
		if err != nil {
			return nil, fmt.Errorf("failed to build proposer sender: %w", err)
		}
		
		configs.Ids = append(configs.Ids, proposerSender.ID)
		configs.Artifacts = append(configs.Artifacts, proposerSender.Artifact)
		configs.ConstructorArgs = append(configs.ConstructorArgs, proposerSender.Args)
	}
	
	return configs, nil
}

// buildDefaultSender builds the default sender configuration
func (ctx *Context) buildDefaultSender() (*SenderConfig, error) {
	deployer := ctx.Profile.Deployer
	
	switch deployer.Type {
	case "private_key":
		return ctx.buildPrivateKeySender("default", deployer.PrivateKey)
		
	case "safe":
		return ctx.buildSafeSender("default", deployer.Safe, deployer.Proposer)
		
	case "ledger":
		return ctx.buildHardwareWalletSender("default", deployer.DerivationPath)
		
	default:
		return nil, fmt.Errorf("unsupported deployer type: %s", deployer.Type)
	}
}

// buildProposerSender builds the proposer sender for Safe deployments
func (ctx *Context) buildProposerSender() (*SenderConfig, error) {
	proposer := ctx.Profile.Deployer.Proposer
	if proposer == nil {
		return nil, fmt.Errorf("proposer configuration is required for Safe deployments")
	}
	
	switch proposer.Type {
	case "private_key":
		return ctx.buildPrivateKeySender("proposer", proposer.PrivateKey)
		
	case "ledger":
		return ctx.buildHardwareWalletSender("proposer", proposer.DerivationPath)
		
	default:
		return nil, fmt.Errorf("unsupported proposer type: %s", proposer.Type)
	}
}

// buildPrivateKeySender builds a private key sender configuration
func (ctx *Context) buildPrivateKeySender(id, privateKey string) (*SenderConfig, error) {
	// Parse private key to get address
	key, err := parsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	
	// Encode constructor arguments (address)
	args, err := encodeAddress(key.Address)
	if err != nil {
		return nil, err
	}
	
	return &SenderConfig{
		ID:       id,
		Type:     "private_key",
		Artifact: "PrivateKeySender.sol:PrivateKeySender",
		Args:     args,
	}, nil
}

// buildSafeSender builds a Safe sender configuration
func (ctx *Context) buildSafeSender(id, safeAddress string, proposer *config.ProposerConfig) (*SenderConfig, error) {
	// Parse Safe address
	safe := common.HexToAddress(safeAddress)
	if safe == (common.Address{}) {
		return nil, fmt.Errorf("invalid Safe address: %s", safeAddress)
	}
	
	// For Safe sender, we need the Safe address in constructor
	args, err := encodeAddress(safe)
	if err != nil {
		return nil, err
	}
	
	return &SenderConfig{
		ID:       id,
		Type:     "safe",
		Artifact: "SafeSender.sol:SafeSender",
		Args:     args,
	}, nil
}

// buildHardwareWalletSender builds a hardware wallet sender configuration
func (ctx *Context) buildHardwareWalletSender(id, derivationPath string) (*SenderConfig, error) {
	// For hardware wallet, we need to get the address from the device
	// This would typically involve interacting with the hardware wallet
	// For now, we'll use a placeholder
	
	// Parse derivation path to get account index
	// Expected format: m/44'/60'/0'/0/0
	parts := strings.Split(derivationPath, "/")
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid derivation path: %s", derivationPath)
	}
	
	// Extract account index (last part)
	accountStr := strings.TrimSuffix(parts[5], "'")
	accountIndex := new(big.Int)
	if _, ok := accountIndex.SetString(accountStr, 10); !ok {
		accountIndex = big.NewInt(0) // Default to 0 if parsing fails
	}
	
	// Encode constructor arguments (uint256 accountIndex)
	args, err := encodeUint256(accountIndex)
	if err != nil {
		return nil, err
	}
	
	return &SenderConfig{
		ID:       id,
		Type:     "ledger",
		Artifact: "HardwareWalletSender.sol:HardwareWalletSender",
		Args:     args,
	}, nil
}

// EncodeSenderConfigs encodes the sender configs for passing as environment variable
func EncodeSenderConfigs(configs *SenderConfigs) (string, error) {
	// Define the ABI for SenderConfigs struct
	senderConfigsABI := `[{
		"components": [
			{"name": "ids", "type": "string[]"},
			{"name": "artifacts", "type": "string[]"},
			{"name": "constructorArgs", "type": "bytes[]"}
		],
		"name": "configs",
		"type": "tuple"
	}]`
	
	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(senderConfigsABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}
	
	// Get the method (using the struct name)
	method := parsedABI.Methods[""]
	if len(parsedABI.Methods) == 0 {
		// For struct encoding, we need to use the type directly
		// Create a temporary method for encoding
		method = abi.NewMethod("", "", abi.Function, "", false, false, 
			[]abi.Argument{{Name: "configs", Type: parsedABI.Methods[""].Inputs[0].Type}},
			nil)
	}
	
	// Pack the data
	packed, err := packStruct(configs)
	if err != nil {
		return "", fmt.Errorf("failed to pack sender configs: %w", err)
	}
	
	// Return as hex string with 0x prefix
	return "0x" + hex.EncodeToString(packed), nil
}

// packStruct manually packs the SenderConfigs struct
func packStruct(configs *SenderConfigs) ([]byte, error) {
	// Create ABI encoder
	uint256Type, _ := abi.NewType("uint256", "", nil)
	stringArrayType, _ := abi.NewType("string[]", "", nil)
	bytesArrayType, _ := abi.NewType("bytes[]", "", nil)
	
	// Create arguments for encoding
	args := abi.Arguments{
		{Type: stringArrayType},
		{Type: stringArrayType},
		{Type: bytesArrayType},
	}
	
	// Pack the values
	packed, err := args.Pack(configs.Ids, configs.Artifacts, configs.ConstructorArgs)
	if err != nil {
		return nil, err
	}
	
	// For struct encoding, we need to add the offset
	// The packed data should start with a 32-byte offset pointing to the struct data
	offset := make([]byte, 32)
	offset[31] = 0x20 // Offset to the struct data (32 bytes)
	
	return append(offset, packed...), nil
}

// Helper functions for encoding constructor arguments

func encodeAddress(addr common.Address) ([]byte, error) {
	addressType, _ := abi.NewType("address", "", nil)
	args := abi.Arguments{{Type: addressType}}
	return args.Pack(addr)
}

func encodeUint256(value *big.Int) ([]byte, error) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{{Type: uint256Type}}
	return args.Pack(value)
}

// parsePrivateKey parses a private key string and returns the crypto.PrivateKey
func parsePrivateKey(privateKeyHex string) (*PrivateKey, error) {
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
	
	return &PrivateKey{
		PrivateKey: privateKey,
		Address:    address,
	}, nil
}
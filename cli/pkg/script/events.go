package script

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Event signatures (keccak256 of event signature)
var (
	// From Deployer.sol
	// DeployingContract(string what, string label, bytes32 initCodeHash)
	DeployingContractTopic = crypto.Keccak256Hash([]byte("DeployingContract(string,string,bytes32)"))
	
	// ContractDeployed(address indexed deployer, address indexed location, bytes32 indexed bundleId, bytes32 salt, bytes32 bytecodeHash, bytes32 initCodeHash, bytes constructorArgs, string createStrategy)
	ContractDeployedTopic = crypto.Keccak256Hash([]byte("ContractDeployed(address,address,bytes32,bytes32,bytes32,bytes32,bytes,string)"))

	// From GnosisSafeSender.sol
	// SafeTransactionQueued(address indexed safe, address indexed proposer, bytes32 indexed bundleId, bytes32 safeTxHash, uint256 nonce)
	SafeTransactionQueuedTopic = crypto.Keccak256Hash([]byte("SafeTransactionQueued(address,address,bytes32,bytes32,uint256)"))
	
	// From Senders.sol
	// BundleSent(address indexed sender, bytes32 indexed bundleId, BundleStatus status, RichTransaction[] transactions)
	BundleSentTopic = crypto.Keccak256Hash([]byte("BundleSent(address,bytes32,uint8,((string,address,bytes,uint256),bytes,bytes)[]))"))
)

// Event types
type EventType string

const (
	EventTypeDeployingContract EventType = "DeployingContract"
	EventTypeContractDeployed  EventType = "ContractDeployed"
	EventTypeSafeTransaction   EventType = "SafeTransactionQueued"
	EventTypeBundleSent        EventType = "BundleSent"
)

// ParsedEvent is the interface for all parsed events
type ParsedEvent interface {
	Type() EventType
	String() string
}

// DeployingContractEvent represents a contract being deployed
type DeployingContractEvent struct {
	What         string
	Label        string
	InitCodeHash common.Hash
}

func (e DeployingContractEvent) Type() EventType { return EventTypeDeployingContract }
func (e DeployingContractEvent) String() string {
	return fmt.Sprintf("Deploying %s (label: %s, initCodeHash: %s)", e.What, e.Label, e.InitCodeHash.Hex()[:10]+"...")
}

// ContractDeployedEvent represents a deployed contract
type ContractDeployedEvent struct {
	Deployer       common.Address
	Location       common.Address
	BundleID       common.Hash
	Salt           common.Hash
	BytecodeHash   common.Hash  // This is the actual bytecode hash from the event
	InitCodeHash   common.Hash  // Kept for backward compatibility
	ConstructorArgs []byte
	CreateStrategy string
}

func (e ContractDeployedEvent) Type() EventType { return EventTypeContractDeployed }
func (e ContractDeployedEvent) String() string {
	return fmt.Sprintf("Deployed contract at %s by %s (strategy: %s, salt: %s)", 
		e.Location.Hex(), e.Deployer.Hex()[:10]+"...", e.CreateStrategy, e.Salt.Hex()[:10]+"...")
}

// SafeTransactionQueuedEvent represents a Safe transaction queued
type SafeTransactionQueuedEvent struct {
	Safe        common.Address
	Proposer    common.Address
	BundleID    common.Hash
	SafeTxHash  common.Hash
	Nonce       uint64
}

func (e SafeTransactionQueuedEvent) Type() EventType { return EventTypeSafeTransaction }
func (e SafeTransactionQueuedEvent) String() string {
	return fmt.Sprintf("Safe transaction queued: safe=%s, nonce=%d, txHash=%s", 
		e.Safe.Hex()[:10]+"...", e.Nonce, e.SafeTxHash.Hex()[:10]+"...")
}

// BundleSentEvent represents a bundle of transactions sent
type BundleSentEvent struct {
	Sender          common.Address
	BundleID        common.Hash
	Status          uint8
	TransactionCount int
}

func (e BundleSentEvent) Type() EventType { return EventTypeBundleSent }
func (e BundleSentEvent) String() string {
	status := "QUEUED"
	if e.Status == 1 {
		status = "EXECUTED"
	}
	return fmt.Sprintf("Bundle sent by %s (status: %s, transactions: %d)", 
		e.Sender.Hex()[:10]+"...", status, e.TransactionCount)
}

// Legacy event types for backward compatibility
type DeploymentEvent = ContractDeployedEvent

// Log represents a log entry from the script output
type Log struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    string         `json:"data"`
}

// parseDeployingContractEvent parses a DeployingContract event from a log
func parseDeployingContractEvent(log Log) (*DeployingContractEvent, error) {
	// No indexed parameters, all in data
	// Parameters: string what, string label, bytes32 initCodeHash
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	stringType, _ := abi.NewType("string", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)

	args := abi.Arguments{
		{Type: stringType, Name: "what"},
		{Type: stringType, Name: "label"},
		{Type: bytes32Type, Name: "initCodeHash"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 3 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	what, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast what")
	}

	label, ok := values[1].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast label")
	}

	initCodeHash, ok := values[2].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast init code hash")
	}

	return &DeployingContractEvent{
		What:         what,
		Label:        label,
		InitCodeHash: common.BytesToHash(initCodeHash[:]),
	}, nil
}

// parseContractDeployedEvent parses a ContractDeployed event from a log
func parseContractDeployedEvent(log Log) (*ContractDeployedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for ContractDeployed event: got %d", len(log.Topics))
	}

	// Topics: [eventSig, deployer (indexed), location (indexed), bundleId (indexed)]
	deployer := common.HexToAddress(log.Topics[1].Hex())
	location := common.HexToAddress(log.Topics[2].Hex())
	bundleID := log.Topics[3]

	// Decode non-indexed parameters from data
	// Parameters: bytes32 salt, bytes32 bytecodeHash, bytes32 initCodeHash, bytes constructorArgs, string createStrategy
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	stringType, _ := abi.NewType("string", "", nil)

	args := abi.Arguments{
		{Type: bytes32Type, Name: "salt"},
		{Type: bytes32Type, Name: "bytecodeHash"},
		{Type: bytes32Type, Name: "initCodeHash"},
		{Type: bytesType, Name: "constructorArgs"},
		{Type: stringType, Name: "createStrategy"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 5 {
		return nil, fmt.Errorf("unexpected number of values unpacked: got %d, expected 5", len(values))
	}

	salt, ok := values[0].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast salt")
	}

	bytecodeHash, ok := values[1].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast bytecode hash")
	}

	initCodeHash, ok := values[2].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast init code hash")
	}

	constructorArgs, ok := values[3].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast constructor args")
	}

	createStrategy, ok := values[4].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast create strategy")
	}

	return &ContractDeployedEvent{
		Deployer:        deployer,
		Location:        location,
		BundleID:        bundleID,
		Salt:            common.BytesToHash(salt[:]),
		BytecodeHash:    common.BytesToHash(bytecodeHash[:]),
		InitCodeHash:    common.BytesToHash(initCodeHash[:]),
		ConstructorArgs: constructorArgs,
		CreateStrategy:  createStrategy,
	}, nil
}


// parseSafeTransactionQueuedEvent parses a SafeTransactionQueued event from a log
func parseSafeTransactionQueuedEvent(log Log) (*SafeTransactionQueuedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for SafeTransactionQueued event")
	}

	// Topics: [eventSig, safe (indexed), proposer (indexed), bundleId (indexed)]
	safe := common.HexToAddress(log.Topics[1].Hex())
	proposer := common.HexToAddress(log.Topics[2].Hex())
	bundleID := log.Topics[3]

	// Decode non-indexed parameters from data
	// Parameters: bytes32 safeTxHash, uint256 nonce
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{
		{Type: bytes32Type, Name: "safeTxHash"},
		{Type: uint256Type, Name: "nonce"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 2 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	safeTxHash, ok := values[0].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast safeTxHash")
	}

	nonce, ok := values[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to cast nonce")
	}

	return &SafeTransactionQueuedEvent{
		Safe:       safe,
		Proposer:   proposer,
		BundleID:   bundleID,
		SafeTxHash: common.BytesToHash(safeTxHash[:]),
		Nonce:      nonce.Uint64(),
	}, nil
}

// parseBundleSentEvent parses a BundleSent event from a log
func parseBundleSentEvent(log Log) (*BundleSentEvent, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("invalid number of topics for BundleSent event")
	}

	// Topics: [eventSig, sender (indexed), bundleId (indexed)]
	sender := common.HexToAddress(log.Topics[1].Hex())
	bundleID := log.Topics[2]

	// Decode non-indexed parameters from data
	// Parameters: BundleStatus status, RichTransaction[] transactions
	// For now, we'll just extract status and transaction count
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// The data starts with offset to dynamic data (32 bytes), then status (32 bytes)
	if len(data) < 64 {
		return nil, fmt.Errorf("insufficient data for BundleSent event")
	}

	// Read status from bytes 32-63
	status := uint8(data[63])

	// For transaction count, we need to decode the array
	// This is complex due to the nested structure, so for now we'll skip it
	// and just report that transactions were sent
	transactionCount := -1 // Unknown

	return &BundleSentEvent{
		Sender:           sender,
		BundleID:         bundleID,
		Status:           status,
		TransactionCount: transactionCount,
	}, nil
}
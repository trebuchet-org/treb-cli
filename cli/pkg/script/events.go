package script

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// Event signatures (keccak256 of event signature)
var (
	// From Deployer.sol
	// DeployingContract(string what, string label, bytes32 initCodeHash)
	DeployingContractTopic = crypto.Keccak256Hash([]byte("DeployingContract(string,string,bytes32)"))

	// ContractDeployed(address indexed deployer, address indexed location, bytes32 indexed transactionId, (string,string,string,bytes32,bytes32,bytes32,bytes,string) deployment)
	ContractDeployedTopic = crypto.Keccak256Hash([]byte("ContractDeployed(address,address,bytes32,(string,string,string,bytes32,bytes32,bytes32,bytes,string))"))

	// From Senders.sol - New transaction events
	// TransactionSimulated(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label, bytes returnData)
	TransactionSimulatedTopic = crypto.Keccak256Hash([]byte("TransactionSimulated(bytes32,address,address,uint256,bytes,string,bytes)"))

	// TransactionFailed(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label)
	TransactionFailedTopic = crypto.Keccak256Hash([]byte("TransactionFailed(bytes32,address,address,uint256,bytes,string)"))

	// TransactionBroadcast(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label, bytes returnData)
	TransactionBroadcastTopic = crypto.Keccak256Hash([]byte("TransactionBroadcast(bytes32,address,address,uint256,bytes,string,bytes)"))

	// From SafeSender.sol
	// SafeTransactionQueued(bytes32 indexed safeTxHash, address indexed safe, address indexed proposer, (address,bytes,uint256,string,bytes32,bytes32,uint8,bytes,bytes)[] transactions)
	SafeTransactionQueuedTopic = crypto.Keccak256Hash([]byte("SafeTransactionQueued(bytes32,address,address,((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes)[])"))

	// ERC1967 Proxy Events
	// Upgraded(address indexed implementation)
	UpgradedTopic = crypto.Keccak256Hash([]byte("Upgraded(address)"))

	// AdminChanged(address previousAdmin, address newAdmin)
	AdminChangedTopic = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))

	// BeaconUpgraded(address indexed beacon)
	BeaconUpgradedTopic = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
)

// Re-export types from events package for backward compatibility
type (
	EventType                  = events.EventType
	ParsedEvent                = events.ParsedEvent
	DeployingContractEvent     = events.DeployingContractEvent
	EventDeployment            = events.EventDeployment
	ContractDeployedEvent      = events.ContractDeployedEvent
	Transaction                = events.Transaction
	RichTransaction            = events.RichTransaction
	SafeTransactionQueuedEvent = events.SafeTransactionQueuedEvent
	TransactionSimulatedEvent  = events.TransactionSimulatedEvent
	TransactionFailedEvent     = events.TransactionFailedEvent
	TransactionBroadcastEvent  = events.TransactionBroadcastEvent
	UpgradedEvent              = events.UpgradedEvent
	AdminChangedEvent          = events.AdminChangedEvent
	BeaconUpgradedEvent        = events.BeaconUpgradedEvent
	DeploymentEvent            = events.DeploymentEvent
	Log                        = events.Log
)

// Re-export event type constants for backward compatibility
const (
	EventTypeDeployingContract    = events.EventTypeDeployingContract
	EventTypeContractDeployed     = events.EventTypeContractDeployed
	EventTypeSafeTransaction      = events.EventTypeSafeTransactionQueued
	EventTypeTransactionSimulated = events.EventTypeTransactionSimulated
	EventTypeTransactionFailed    = events.EventTypeTransactionFailed
	EventTypeTransactionBroadcast = events.EventTypeTransactionBroadcast
	EventTypeUpgraded             = events.EventTypeUpgraded
	EventTypeAdminChanged         = events.EventTypeAdminChanged
	EventTypeBeaconUpgraded       = events.EventTypeBeaconUpgraded
)

// parseDeployingContractEvent parses a DeployingContract event from a log
func parseDeployingContractEvent(log events.Log) (*events.DeployingContractEvent, error) {
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

	return &events.DeployingContractEvent{
		What:         what,
		Label:        label,
		InitCodeHash: common.BytesToHash(initCodeHash[:]),
	}, nil
}

// parseContractDeployedEvent parses a ContractDeployed event from a log
func parseContractDeployedEvent(log events.Log) (*events.ContractDeployedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for ContractDeployed event: got %d", len(log.Topics))
	}

	// Topics: [eventSig, deployer (indexed), location (indexed), transactionId (indexed)]
	deployer := common.HexToAddress(log.Topics[1].Hex())
	location := common.HexToAddress(log.Topics[2].Hex())
	transactionID := log.Topics[3]

	// Decode non-indexed parameters from data
	// Parameters: EventDeployment struct (string,string,string,bytes32,bytes32,bytes32,bytes,string)
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding the EventDeployment struct
	deploymentStructType, err := abi.NewType("tuple", "struct EventDeployment", []abi.ArgumentMarshaling{
		{Name: "artifact", Type: "string"},
		{Name: "label", Type: "string"},
		{Name: "entropy", Type: "string"},
		{Name: "salt", Type: "bytes32"},
		{Name: "bytecodeHash", Type: "bytes32"},
		{Name: "initCodeHash", Type: "bytes32"},
		{Name: "constructorArgs", Type: "bytes"},
		{Name: "createStrategy", Type: "string"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EventDeployment struct type: %w", err)
	}

	args := abi.Arguments{
		{Type: deploymentStructType, Name: "deployment"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 1 {
		return nil, fmt.Errorf("unexpected number of values unpacked: got %d, expected 1", len(values))
	}

	// Extract the struct value using reflect
	structValue := values[0]

	// Use type assertion to access the fields by reflection
	structReflect := reflect.ValueOf(structValue)
	if structReflect.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", structReflect.Kind())
	}

	if structReflect.NumField() != 8 {
		return nil, fmt.Errorf("expected 8 fields in struct, got %d", structReflect.NumField())
	}

	// Extract fields by index
	artifact := structReflect.Field(0).String()
	label := structReflect.Field(1).String()
	entropy := structReflect.Field(2).String()
	salt := structReflect.Field(3).Interface().([32]byte)
	bytecodeHash := structReflect.Field(4).Interface().([32]byte)
	initCodeHash := structReflect.Field(5).Interface().([32]byte)
	constructorArgs := structReflect.Field(6).Interface().([]byte)
	createStrategy := structReflect.Field(7).String()

	return &events.ContractDeployedEvent{
		Deployer:      deployer,
		Location:      location,
		TransactionID: transactionID,
		Deployment: events.EventDeployment{
			Artifact:        artifact,
			Label:           label,
			Entropy:         entropy,
			Salt:            common.BytesToHash(salt[:]),
			BytecodeHash:    common.BytesToHash(bytecodeHash[:]),
			InitCodeHash:    common.BytesToHash(initCodeHash[:]),
			ConstructorArgs: constructorArgs,
			CreateStrategy:  createStrategy,
		},
	}, nil
}

// parseSafeTransactionQueuedEvent parses a SafeTransactionQueued event from a log
func parseSafeTransactionQueuedEvent(log events.Log) (*events.SafeTransactionQueuedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for SafeTransactionQueued event")
	}

	// Topics: [eventSig, safeTxHash (indexed), safe (indexed), proposer (indexed)]
	safeTxHash := log.Topics[1]
	safe := common.HexToAddress(log.Topics[2].Hex())
	proposer := common.HexToAddress(log.Topics[3].Hex())

	// Decode non-indexed parameters from data
	// Parameters: RichTransaction[] transactions
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding the RichTransaction array
	// Now define the RichTransaction struct type
	richTransactionType, err := abi.NewType("tuple", "struct RichTransaction", []abi.ArgumentMarshaling{
		{Name: "transaction", Type: "tuple", Components: []abi.ArgumentMarshaling{
			{Name: "label", Type: "string"},
			{Name: "to", Type: "address"},
			{Name: "data", Type: "bytes"},
			{Name: "value", Type: "uint256"},
		}},
		{Name: "transactionId", Type: "bytes32"},
		{Name: "senderId", Type: "bytes32"},
		{Name: "status", Type: "uint8"},
		{Name: "simulatedReturnData", Type: "bytes"},
		{Name: "executedReturnData", Type: "bytes"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create RichTransaction type: %w", err)
	}

	// Create array type
	richTransactionArrayType, err := abi.NewType("tuple[]", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create array type: %w", err)
	}
	richTransactionArrayType.Elem = &richTransactionType

	args := abi.Arguments{
		{Type: richTransactionArrayType, Name: "transactions"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 1 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	// Parse the transactions using reflection
	var transactions []events.RichTransaction
	txArray := reflect.ValueOf(values[0])

	for i := 0; i < txArray.Len(); i++ {
		txValue := txArray.Index(i)
		if txValue.Kind() != reflect.Struct {
			continue
		}

		// Extract RichTransaction fields
		transactionField := txValue.Field(0)
		transactionIdField := txValue.Field(1).Interface().([32]byte)
		senderIdField := txValue.Field(2).Interface().([32]byte)
		statusField := uint8(txValue.Field(3).Uint())
		simulatedReturnDataField := txValue.Field(4).Interface().([]byte)
		executedReturnDataField := txValue.Field(5).Interface().([]byte)

		// Extract Transaction struct fields
		label := transactionField.Field(0).String()
		to := transactionField.Field(1).Interface().(common.Address)
		txData := transactionField.Field(2).Interface().([]byte)
		value := transactionField.Field(3).Interface().(*big.Int)

		richTx := events.RichTransaction{
			Transaction: events.Transaction{
				Label: label,
				To:    to,
				Data:  txData,
				Value: value,
			},
			TransactionID:       common.BytesToHash(transactionIdField[:]),
			SenderID:            common.BytesToHash(senderIdField[:]),
			Status:              statusField,
			SimulatedReturnData: simulatedReturnDataField,
			ExecutedReturnData:  executedReturnDataField,
		}

		transactions = append(transactions, richTx)
	}

	return &events.SafeTransactionQueuedEvent{
		SafeTxHash:   safeTxHash,
		Safe:         safe,
		Proposer:     proposer,
		Transactions: transactions,
	}, nil
}

// parseTransactionSimulatedEvent parses a TransactionSimulated event from a log
func parseTransactionSimulatedEvent(log events.Log) (*events.TransactionSimulatedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for TransactionSimulated event")
	}

	// Topics: [eventSig, transactionId (indexed), sender (indexed), to (indexed)]
	transactionID := log.Topics[1]
	sender := common.HexToAddress(log.Topics[2].Hex())
	to := common.HexToAddress(log.Topics[3].Hex())

	// Decode non-indexed parameters from data
	// Parameters: uint256 value, bytes data, string label, bytes returnData
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	stringType, _ := abi.NewType("string", "", nil)

	args := abi.Arguments{
		{Type: uint256Type, Name: "value"},
		{Type: bytesType, Name: "data"},
		{Type: stringType, Name: "label"},
		{Type: bytesType, Name: "returnData"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 4 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to cast value")
	}

	txData, ok := values[1].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast data")
	}

	label, ok := values[2].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast label")
	}

	returnData, ok := values[3].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast return data")
	}

	return &events.TransactionSimulatedEvent{
		TransactionID: transactionID,
		Sender:        sender,
		To:            to,
		Value:         value,
		Data:          txData,
		Label:         label,
		ReturnData:    returnData,
	}, nil
}

// parseTransactionFailedEvent parses a TransactionFailed event from a log
func parseTransactionFailedEvent(log events.Log) (*events.TransactionFailedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for TransactionFailed event")
	}

	// Topics: [eventSig, transactionId (indexed), sender (indexed), to (indexed)]
	transactionID := log.Topics[1]
	sender := common.HexToAddress(log.Topics[2].Hex())
	to := common.HexToAddress(log.Topics[3].Hex())

	// Decode non-indexed parameters from data
	// Parameters: uint256 value, bytes data, string label
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	stringType, _ := abi.NewType("string", "", nil)

	args := abi.Arguments{
		{Type: uint256Type, Name: "value"},
		{Type: bytesType, Name: "data"},
		{Type: stringType, Name: "label"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 3 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to cast value")
	}

	txData, ok := values[1].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast data")
	}

	label, ok := values[2].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast label")
	}

	return &events.TransactionFailedEvent{
		TransactionID: transactionID,
		Sender:        sender,
		To:            to,
		Value:         value,
		Data:          txData,
		Label:         label,
	}, nil
}

// parseTransactionBroadcastEvent parses a TransactionBroadcast event from a log
func parseTransactionBroadcastEvent(log events.Log) (*events.TransactionBroadcastEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("invalid number of topics for TransactionBroadcast event")
	}

	// Topics: [eventSig, transactionId (indexed), sender (indexed), to (indexed)]
	transactionID := log.Topics[1]
	sender := common.HexToAddress(log.Topics[2].Hex())
	to := common.HexToAddress(log.Topics[3].Hex())

	// Decode non-indexed parameters from data
	// Parameters: uint256 value, bytes data, string label, bytes returnData
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	stringType, _ := abi.NewType("string", "", nil)

	args := abi.Arguments{
		{Type: uint256Type, Name: "value"},
		{Type: bytesType, Name: "data"},
		{Type: stringType, Name: "label"},
		{Type: bytesType, Name: "returnData"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 4 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to cast value")
	}

	txData, ok := values[1].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast data")
	}

	label, ok := values[2].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast label")
	}

	returnData, ok := values[3].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast return data")
	}

	return &events.TransactionBroadcastEvent{
		TransactionID: transactionID,
		Sender:        sender,
		To:            to,
		Value:         value,
		Data:          txData,
		Label:         label,
		ReturnData:    returnData,
	}, nil
}

// parseUpgradedEvent parses an Upgraded event from a log
func parseUpgradedEvent(log events.Log) (*events.UpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid number of topics for Upgraded event")
	}

	// Topics: [eventSig, implementation (indexed)]
	implementation := common.HexToAddress(log.Topics[1].Hex())

	// The proxy address is the address that emitted the event
	proxyAddress := log.Address

	return &events.UpgradedEvent{
		ProxyAddress:   proxyAddress,
		Implementation: implementation,
		TransactionID:  common.Hash{}, // Will be filled in by context
	}, nil
}

// parseAdminChangedEvent parses an AdminChanged event from a log
func parseAdminChangedEvent(log events.Log) (*events.AdminChangedEvent, error) {
	if len(log.Topics) < 1 {
		return nil, fmt.Errorf("invalid number of topics for AdminChanged event")
	}

	// Decode non-indexed parameters from data
	// Parameters: address previousAdmin, address newAdmin
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	addressType, _ := abi.NewType("address", "", nil)
	args := abi.Arguments{
		{Type: addressType, Name: "previousAdmin"},
		{Type: addressType, Name: "newAdmin"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 2 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	previousAdmin, ok := values[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("failed to cast previousAdmin")
	}

	newAdmin, ok := values[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf("failed to cast newAdmin")
	}

	return &events.AdminChangedEvent{
		ProxyAddress:  log.Address,
		PreviousAdmin: previousAdmin,
		NewAdmin:      newAdmin,
		TransactionID: common.Hash{}, // Will be filled in by context
	}, nil
}

// parseBeaconUpgradedEvent parses a BeaconUpgraded event from a log
func parseBeaconUpgradedEvent(log events.Log) (*events.BeaconUpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid number of topics for BeaconUpgraded event")
	}

	// Topics: [eventSig, beacon (indexed)]
	beacon := common.HexToAddress(log.Topics[1].Hex())

	return &events.BeaconUpgradedEvent{
		ProxyAddress:  log.Address,
		Beacon:        beacon,
		TransactionID: common.Hash{}, // Will be filled in by context
	}, nil
}

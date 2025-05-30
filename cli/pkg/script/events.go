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
	SafeTransactionQueuedTopic = crypto.Keccak256Hash([]byte("SafeTransactionQueued(bytes32,address,address,(address,bytes,uint256,string,bytes32,bytes32,uint8,bytes,bytes)[])"))

	// ERC1967 Proxy Events
	// Upgraded(address indexed implementation)
	UpgradedTopic = crypto.Keccak256Hash([]byte("Upgraded(address)"))

	// AdminChanged(address previousAdmin, address newAdmin)
	AdminChangedTopic = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))

	// BeaconUpgraded(address indexed beacon)
	BeaconUpgradedTopic = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
)

// Event types
type EventType string

const (
	EventTypeDeployingContract    EventType = "DeployingContract"
	EventTypeContractDeployed     EventType = "ContractDeployed"
	EventTypeSafeTransaction      EventType = "SafeTransactionQueued"
	EventTypeTransactionSimulated EventType = "TransactionSimulated"
	EventTypeTransactionFailed    EventType = "TransactionFailed"
	EventTypeTransactionBroadcast EventType = "TransactionBroadcast"
	EventTypeUpgraded             EventType = "Upgraded"
	EventTypeAdminChanged         EventType = "AdminChanged"
	EventTypeBeaconUpgraded       EventType = "BeaconUpgraded"
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

// EventDeployment represents the EventDeployment struct from Deployer.sol
type EventDeployment struct {
	Artifact        string // Contract artifact path (e.g., "src/Counter.sol:Counter")
	Label           string // Optional label for deployment identification
	Entropy         string // Entropy string used for salt generation
	Salt            common.Hash
	BytecodeHash    common.Hash // This is the actual bytecode hash from the event
	InitCodeHash    common.Hash // Hash of bytecode + constructor args
	ConstructorArgs []byte
	CreateStrategy  string
}

// ContractDeployedEvent represents a deployed contract
type ContractDeployedEvent struct {
	Deployer      common.Address
	Location      common.Address
	TransactionID common.Hash
	Deployment    EventDeployment
}

func (e ContractDeployedEvent) Type() EventType { return EventTypeContractDeployed }
func (e ContractDeployedEvent) String() string {
	return fmt.Sprintf("Deployed contract at %s by %s (strategy: %s, salt: %s)",
		e.Location.Hex(), e.Deployer.Hex()[:10]+"...", e.Deployment.CreateStrategy, e.Deployment.Salt.Hex()[:10]+"...")
}

// Transaction represents the base transaction struct from treb-sol
type Transaction struct {
	Label string
	To    common.Address
	Data  []byte
	Value *big.Int
}

// RichTransaction represents a transaction with metadata from treb-sol
type RichTransaction struct {
	Transaction           Transaction
	TransactionID         common.Hash
	SenderID              common.Hash
	Status                uint8  // 0: PENDING, 1: EXECUTED, 2: QUEUED
	SimulatedReturnData   []byte
	ExecutedReturnData    []byte
}

// SafeTransactionQueuedEvent represents a Safe transaction queued
type SafeTransactionQueuedEvent struct {
	SafeTxHash   common.Hash
	Safe         common.Address
	Proposer     common.Address
	Transactions []RichTransaction
}

func (e SafeTransactionQueuedEvent) Type() EventType { return EventTypeSafeTransaction }
func (e SafeTransactionQueuedEvent) String() string {
	labels := []string{}
	for _, tx := range e.Transactions {
		if tx.Transaction.Label != "" {
			labels = append(labels, tx.Transaction.Label)
		}
	}
	labelsStr := ""
	if len(labels) > 0 {
		labelsStr = fmt.Sprintf(" [%s]", strings.Join(labels, ", "))
	}
	return fmt.Sprintf("Safe transaction queued: safe=%s, proposer=%s, txHash=%s, transactions=%d%s",
		e.Safe.Hex()[:10]+"...", e.Proposer.Hex()[:10]+"...", e.SafeTxHash.Hex()[:10]+"...", len(e.Transactions), labelsStr)
}

// TransactionSimulatedEvent represents a simulated transaction
type TransactionSimulatedEvent struct {
	TransactionID common.Hash
	Sender        common.Address
	To            common.Address
	Value         *big.Int
	Data          []byte
	Label         string
	ReturnData    []byte
}

func (e TransactionSimulatedEvent) Type() EventType { return EventTypeTransactionSimulated }
func (e TransactionSimulatedEvent) String() string {
	return fmt.Sprintf("Transaction simulated: %s -> %s (label: %s, value: %s)",
		e.Sender.Hex()[:10]+"...", e.To.Hex()[:10]+"...", e.Label, e.Value.String())
}

// TransactionFailedEvent represents a failed transaction
type TransactionFailedEvent struct {
	TransactionID common.Hash
	Sender        common.Address
	To            common.Address
	Value         *big.Int
	Data          []byte
	Label         string
}

func (e TransactionFailedEvent) Type() EventType { return EventTypeTransactionFailed }
func (e TransactionFailedEvent) String() string {
	return fmt.Sprintf("Transaction failed: %s -> %s (label: %s)",
		e.Sender.Hex()[:10]+"...", e.To.Hex()[:10]+"...", e.Label)
}

// TransactionBroadcastEvent represents a broadcast transaction
type TransactionBroadcastEvent struct {
	TransactionID common.Hash
	Sender        common.Address
	To            common.Address
	Value         *big.Int
	Data          []byte
	Label         string
	ReturnData    []byte
}

func (e TransactionBroadcastEvent) Type() EventType { return EventTypeTransactionBroadcast }
func (e TransactionBroadcastEvent) String() string {
	return fmt.Sprintf("Transaction broadcast: %s -> %s (label: %s, value: %s)",
		e.Sender.Hex()[:10]+"...", e.To.Hex()[:10]+"...", e.Label, e.Value.String())
}

// UpgradedEvent represents a proxy implementation upgrade
type UpgradedEvent struct {
	ProxyAddress     common.Address
	Implementation   common.Address
	TransactionID    common.Hash // We'll try to link this from the transaction context
}

func (e UpgradedEvent) Type() EventType { return EventTypeUpgraded }
func (e UpgradedEvent) String() string {
	return fmt.Sprintf("Proxy upgraded: proxy=%s, implementation=%s", 
		e.ProxyAddress.Hex()[:10]+"...", e.Implementation.Hex()[:10]+"...")
}

// AdminChangedEvent represents a proxy admin change
type AdminChangedEvent struct {
	ProxyAddress   common.Address
	PreviousAdmin  common.Address
	NewAdmin       common.Address
	TransactionID  common.Hash
}

func (e AdminChangedEvent) Type() EventType { return EventTypeAdminChanged }
func (e AdminChangedEvent) String() string {
	return fmt.Sprintf("Admin changed: proxy=%s, old=%s, new=%s", 
		e.ProxyAddress.Hex()[:10]+"...", e.PreviousAdmin.Hex()[:10]+"...", e.NewAdmin.Hex()[:10]+"...")
}

// BeaconUpgradedEvent represents a beacon proxy upgrade
type BeaconUpgradedEvent struct {
	ProxyAddress  common.Address
	Beacon        common.Address
	TransactionID common.Hash
}

func (e BeaconUpgradedEvent) Type() EventType { return EventTypeBeaconUpgraded }
func (e BeaconUpgradedEvent) String() string {
	return fmt.Sprintf("Beacon upgraded: proxy=%s, beacon=%s", 
		e.ProxyAddress.Hex()[:10]+"...", e.Beacon.Hex()[:10]+"...")
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

	return &ContractDeployedEvent{
		Deployer:      deployer,
		Location:      location,
		TransactionID: transactionID,
		Deployment: EventDeployment{
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
func parseSafeTransactionQueuedEvent(log Log) (*SafeTransactionQueuedEvent, error) {
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
	var transactions []RichTransaction
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

		richTx := RichTransaction{
			Transaction: Transaction{
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

	return &SafeTransactionQueuedEvent{
		SafeTxHash:   safeTxHash,
		Safe:         safe,
		Proposer:     proposer,
		Transactions: transactions,
	}, nil
}

// parseTransactionSimulatedEvent parses a TransactionSimulated event from a log
func parseTransactionSimulatedEvent(log Log) (*TransactionSimulatedEvent, error) {
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

	return &TransactionSimulatedEvent{
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
func parseTransactionFailedEvent(log Log) (*TransactionFailedEvent, error) {
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

	return &TransactionFailedEvent{
		TransactionID: transactionID,
		Sender:        sender,
		To:            to,
		Value:         value,
		Data:          txData,
		Label:         label,
	}, nil
}

// parseTransactionBroadcastEvent parses a TransactionBroadcast event from a log
func parseTransactionBroadcastEvent(log Log) (*TransactionBroadcastEvent, error) {
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

	return &TransactionBroadcastEvent{
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
func parseUpgradedEvent(log Log) (*UpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid number of topics for Upgraded event")
	}

	// Topics: [eventSig, implementation (indexed)]
	implementation := common.HexToAddress(log.Topics[1].Hex())

	// The proxy address is the address that emitted the event
	proxyAddress := log.Address

	return &UpgradedEvent{
		ProxyAddress:   proxyAddress,
		Implementation: implementation,
		TransactionID:  common.Hash{}, // Will be filled in by context
	}, nil
}

// parseAdminChangedEvent parses an AdminChanged event from a log
func parseAdminChangedEvent(log Log) (*AdminChangedEvent, error) {
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

	return &AdminChangedEvent{
		ProxyAddress:  log.Address,
		PreviousAdmin: previousAdmin,
		NewAdmin:      newAdmin,
		TransactionID: common.Hash{}, // Will be filled in by context
	}, nil
}

// parseBeaconUpgradedEvent parses a BeaconUpgraded event from a log
func parseBeaconUpgradedEvent(log Log) (*BeaconUpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid number of topics for BeaconUpgraded event")
	}

	// Topics: [eventSig, beacon (indexed)]
	beacon := common.HexToAddress(log.Topics[1].Hex())

	return &BeaconUpgradedEvent{
		ProxyAddress:  log.Address,
		Beacon:        beacon,
		TransactionID: common.Hash{}, // Will be filled in by context
	}, nil
}

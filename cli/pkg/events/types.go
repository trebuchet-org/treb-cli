package events

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeDeployingContract       EventType = "DeployingContract"
	EventTypeContractDeployed        EventType = "ContractDeployed"
	EventTypeSafeTransactionQueued   EventType = "SafeTransactionQueued"
	EventTypeTransactionSimulated    EventType = "TransactionSimulated"
	EventTypeTransactionFailed       EventType = "TransactionFailed"
	EventTypeTransactionBroadcast    EventType = "TransactionBroadcast"
	EventTypeSenderConfigured        EventType = "SenderDeployerConfigured"
	EventTypeAdminChanged            EventType = "AdminChanged"
	EventTypeBeaconUpgraded          EventType = "BeaconUpgraded"
	EventTypeUpgraded                EventType = "Upgraded"
	EventTypeProxyDeployed           EventType = "ProxyDeployed"
	EventTypeTransactionBundleCreated EventType = "TransactionBundleCreated"
	EventTypeRegistryUpdated         EventType = "RegistryUpdated"
	EventTypeUnknown                 EventType = "Unknown"
)

// ParsedEvent is the interface for all parsed events
type ParsedEvent interface {
	Type() EventType
	String() string
}

// DeploymentData represents deployment data embedded in events
type DeploymentData struct {
	Salt             common.Hash
	Entropy          string
	Label            string
	InitCodeHash     common.Hash
	ConstructorArgs  []byte
	BytecodeHash     common.Hash
	Artifact         string
}

// EventDeployment represents the EventDeployment struct from Deployer.sol
// This matches the Solidity struct used in ContractDeployed events
type EventDeployment struct {
	Artifact        string      // Contract artifact path (e.g., "src/Counter.sol:Counter")
	Label           string      // Optional label for deployment identification
	Entropy         string      // Entropy string used for salt generation
	Salt            common.Hash
	BytecodeHash    common.Hash // This is the actual bytecode hash from the event
	InitCodeHash    common.Hash // Hash of bytecode + constructor args
	ConstructorArgs []byte
	CreateStrategy  string
}

// DeployingContractEvent represents a contract being deployed
type DeployingContractEvent struct {
	What         string
	Label        string
	InitCodeHash common.Hash
}

func (e *DeployingContractEvent) Type() EventType {
	return EventTypeDeployingContract
}

func (e *DeployingContractEvent) String() string {
	return fmt.Sprintf("Deploying %s (label: %s, initCodeHash: %s)", e.What, e.Label, e.InitCodeHash.Hex()[:10]+"...")
}

// ContractDeployedEvent represents a contract deployment
type ContractDeployedEvent struct {
	TransactionID common.Hash
	Deployer      common.Address
	Location      common.Address
	Deployment    EventDeployment
}

func (e *ContractDeployedEvent) Type() EventType {
	return EventTypeContractDeployed
}

func (e *ContractDeployedEvent) String() string {
	return fmt.Sprintf("Contract deployed at %s by %s (tx: %s, label: %s)",
		e.Location.Hex(),
		e.Deployer.Hex()[:10],
		e.TransactionID.Hex()[:10],
		e.Deployment.Label,
	)
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

// SafeTransactionData represents a Safe transaction
type SafeTransactionData struct {
	To        common.Address
	Value     *big.Int
	Data      []byte
	Operation uint8
}

// SafeTransactionInfo contains Safe transaction metadata
type SafeTransactionInfo struct {
	TransactionID common.Hash
	Transaction   SafeTransactionData
	Status        uint8
}

// SafeDeployment represents a deployment in a Safe transaction
type SafeDeployment struct {
	TransactionID common.Hash
	Deployment    DeploymentData
}

// SafeTransactionQueuedEvent represents a queued Safe transaction
type SafeTransactionQueuedEvent struct {
	SafeTxHash   common.Hash
	Safe         common.Address
	Proposer     common.Address
	Transactions []RichTransaction
}

func (e *SafeTransactionQueuedEvent) Type() EventType {
	return EventTypeSafeTransactionQueued
}

func (e *SafeTransactionQueuedEvent) String() string {
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

// SenderConfig represents sender configuration
type SenderConfig struct {
	SenderType uint8
	Label      string
}

// SenderDeployerConfiguredEvent represents sender configuration
type SenderDeployerConfiguredEvent struct {
	TransactionID common.Hash
	Sender        SenderConfig
	Deployer      common.Address
}

func (e *SenderDeployerConfiguredEvent) Type() EventType {
	return EventTypeSenderConfigured
}

func (e *SenderDeployerConfiguredEvent) String() string {
	senderType := "Unknown"
	switch e.Sender.SenderType {
	case 0:
		senderType = "PrivateKey"
	case 1:
		senderType = "Ledger"
	case 2:
		senderType = "Trezor"
	case 10:
		senderType = "Safe"
	}
	
	return fmt.Sprintf("Sender configured: %s (%s) -> %s",
		senderType,
		e.Sender.Label,
		e.Deployer.Hex()[:10],
	)
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

func (e *TransactionSimulatedEvent) Type() EventType {
	return EventTypeTransactionSimulated
}

func (e *TransactionSimulatedEvent) String() string {
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

func (e *TransactionFailedEvent) Type() EventType {
	return EventTypeTransactionFailed
}

func (e *TransactionFailedEvent) String() string {
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

func (e *TransactionBroadcastEvent) Type() EventType {
	return EventTypeTransactionBroadcast
}

func (e *TransactionBroadcastEvent) String() string {
	return fmt.Sprintf("Transaction broadcast: %s -> %s (label: %s, value: %s)",
		e.Sender.Hex()[:10]+"...", e.To.Hex()[:10]+"...", e.Label, e.Value.String())
}

// ProxyRelationshipType represents the type of proxy relationship
type ProxyRelationshipType string

const (
	ProxyTypeTransparent ProxyRelationshipType = "Transparent"
	ProxyTypeUUPS        ProxyRelationshipType = "UUPS"
	ProxyTypeBeacon      ProxyRelationshipType = "Beacon"
	ProxyTypeMinimal     ProxyRelationshipType = "Minimal"
)

// ProxyRelationship represents a proxy-implementation relationship
type ProxyRelationship struct {
	ProxyAddress          common.Address
	ImplementationAddress common.Address
	AdminAddress          *common.Address // Optional, for transparent proxies
	BeaconAddress         *common.Address // Optional, for beacon proxies
	ProxyType             ProxyRelationshipType
}

// AdminChangedEvent represents a proxy admin change
type AdminChangedEvent struct {
	ProxyAddress   common.Address
	PreviousAdmin  common.Address
	NewAdmin       common.Address
	TransactionID  common.Hash
}

func (e *AdminChangedEvent) Type() EventType {
	return EventTypeAdminChanged
}

func (e *AdminChangedEvent) String() string {
	return fmt.Sprintf("Admin changed: proxy=%s, old=%s, new=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.PreviousAdmin.Hex()[:10]+"...", e.NewAdmin.Hex()[:10]+"...")
}

// BeaconUpgradedEvent represents a beacon upgrade
type BeaconUpgradedEvent struct {
	ProxyAddress  common.Address
	Beacon        common.Address
	TransactionID common.Hash
}

func (e *BeaconUpgradedEvent) Type() EventType {
	return EventTypeBeaconUpgraded
}

func (e *BeaconUpgradedEvent) String() string {
	return fmt.Sprintf("Beacon upgraded: proxy=%s, beacon=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.Beacon.Hex()[:10]+"...")
}

// UpgradedEvent represents a proxy upgrade
type UpgradedEvent struct {
	ProxyAddress   common.Address
	Implementation common.Address
	TransactionID  common.Hash // We'll try to link this from the transaction context
}

func (e *UpgradedEvent) Type() EventType {
	return EventTypeUpgraded
}

func (e *UpgradedEvent) String() string {
	return fmt.Sprintf("Proxy upgraded: proxy=%s, implementation=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.Implementation.Hex()[:10]+"...")
}

// ProxyDeployedEvent represents a proxy deployment
type ProxyDeployedEvent struct {
	Proxy          common.Address
	Implementation common.Address
}

func (e *ProxyDeployedEvent) Type() EventType {
	return EventTypeProxyDeployed
}

func (e *ProxyDeployedEvent) String() string {
	return fmt.Sprintf("Proxy deployed at %s -> %s",
		e.Proxy.Hex()[:10],
		e.Implementation.Hex()[:10],
	)
}

// UnknownEvent represents an unknown event
type UnknownEvent struct {
	Address common.Address
	Topics  []common.Hash
	Data    string
}

func (e *UnknownEvent) Type() EventType {
	return EventTypeUnknown
}

func (e *UnknownEvent) String() string {
	topic := "0x0"
	if len(e.Topics) > 0 {
		topic = e.Topics[0].Hex()[:10]
	}
	return fmt.Sprintf("Unknown event from %s (topic: %s)",
		e.Address.Hex()[:10],
		topic,
	)
}

// Legacy event types for backward compatibility
type DeploymentEvent = ContractDeployedEvent

// Log represents a log entry from the script output
type Log struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    string         `json:"data"`
}
// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = bytes.Equal
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// ITrebEventsDeploymentDetails is an auto generated low-level Go binding around an user-defined struct.
type ITrebEventsDeploymentDetails struct {
	Artifact        string
	Label           string
	Entropy         string
	Salt            [32]byte
	BytecodeHash    [32]byte
	InitCodeHash    [32]byte
	ConstructorArgs []byte
	CreateStrategy  string
}

// SimulatedTransaction is an auto generated low-level Go binding around an user-defined struct.
type SimulatedTransaction struct {
	TransactionId [32]byte
	SenderId      [32]byte
	Sender        common.Address
	ReturnData    []byte
	Transaction   Transaction
}

// Transaction is an auto generated low-level Go binding around an user-defined struct.
type Transaction struct {
	To    common.Address
	Data  []byte
	Value *big.Int
}

// TrebMetaData contains all meta data concerning the Treb contract.
var TrebMetaData = bind.MetaData{
	ABI: "[{\"type\":\"event\",\"name\":\"ContractDeployed\",\"inputs\":[{\"name\":\"deployer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"location\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"transactionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"deployment\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structITrebEvents.DeploymentDetails\",\"components\":[{\"name\":\"artifact\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"entropy\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"bytecodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"constructorArgs\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"createStrategy\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DeploymentCollision\",\"inputs\":[{\"name\":\"existingContract\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"deploymentDetails\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structITrebEvents.DeploymentDetails\",\"components\":[{\"name\":\"artifact\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"entropy\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"bytecodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"constructorArgs\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"createStrategy\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SafeTransactionExecuted\",\"inputs\":[{\"name\":\"safeTxHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"safe\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"executor\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"transactionIds\",\"type\":\"bytes32[]\",\"indexed\":false,\"internalType\":\"bytes32[]\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SafeTransactionQueued\",\"inputs\":[{\"name\":\"safeTxHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"safe\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"transactionIds\",\"type\":\"bytes32[]\",\"indexed\":false,\"internalType\":\"bytes32[]\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TransactionSimulated\",\"inputs\":[{\"name\":\"simulatedTx\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structSimulatedTransaction\",\"components\":[{\"name\":\"transactionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"senderId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"returnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"transaction\",\"type\":\"tuple\",\"internalType\":\"structTransaction\",\"components\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"anonymous\":false}]",
	ID:  "Treb",
}

// Treb is an auto generated Go binding around an Ethereum contract.
type Treb struct {
	abi abi.ABI
}

// NewTreb creates a new instance of Treb.
func NewTreb() *Treb {
	parsed, err := TrebMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &Treb{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *Treb) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// TrebContractDeployed represents a ContractDeployed event raised by the Treb contract.
type TrebContractDeployed struct {
	Deployer      common.Address
	Location      common.Address
	TransactionId [32]byte
	Deployment    ITrebEventsDeploymentDetails
	Raw           *types.Log // Blockchain specific contextual infos
}

const TrebContractDeployedEventName = "ContractDeployed"

// ContractEventName returns the user-defined event name.
func (TrebContractDeployed) ContractEventName() string {
	return TrebContractDeployedEventName
}

// UnpackContractDeployedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event ContractDeployed(address indexed deployer, address indexed location, bytes32 indexed transactionId, (string,string,string,bytes32,bytes32,bytes32,bytes,string) deployment)
func (treb *Treb) UnpackContractDeployedEvent(log *types.Log) (*TrebContractDeployed, error) {
	event := "ContractDeployed"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebContractDeployed)
	if len(log.Data) > 0 {
		if err := treb.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range treb.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// TrebDeploymentCollision represents a DeploymentCollision event raised by the Treb contract.
type TrebDeploymentCollision struct {
	ExistingContract  common.Address
	DeploymentDetails ITrebEventsDeploymentDetails
	Raw               *types.Log // Blockchain specific contextual infos
}

const TrebDeploymentCollisionEventName = "DeploymentCollision"

// ContractEventName returns the user-defined event name.
func (TrebDeploymentCollision) ContractEventName() string {
	return TrebDeploymentCollisionEventName
}

// UnpackDeploymentCollisionEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event DeploymentCollision(address indexed existingContract, (string,string,string,bytes32,bytes32,bytes32,bytes,string) deploymentDetails)
func (treb *Treb) UnpackDeploymentCollisionEvent(log *types.Log) (*TrebDeploymentCollision, error) {
	event := "DeploymentCollision"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebDeploymentCollision)
	if len(log.Data) > 0 {
		if err := treb.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range treb.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// TrebSafeTransactionExecuted represents a SafeTransactionExecuted event raised by the Treb contract.
type TrebSafeTransactionExecuted struct {
	SafeTxHash     [32]byte
	Safe           common.Address
	Executor       common.Address
	TransactionIds [][32]byte
	Raw            *types.Log // Blockchain specific contextual infos
}

const TrebSafeTransactionExecutedEventName = "SafeTransactionExecuted"

// ContractEventName returns the user-defined event name.
func (TrebSafeTransactionExecuted) ContractEventName() string {
	return TrebSafeTransactionExecutedEventName
}

// UnpackSafeTransactionExecutedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event SafeTransactionExecuted(bytes32 indexed safeTxHash, address indexed safe, address indexed executor, bytes32[] transactionIds)
func (treb *Treb) UnpackSafeTransactionExecutedEvent(log *types.Log) (*TrebSafeTransactionExecuted, error) {
	event := "SafeTransactionExecuted"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebSafeTransactionExecuted)
	if len(log.Data) > 0 {
		if err := treb.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range treb.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// TrebSafeTransactionQueued represents a SafeTransactionQueued event raised by the Treb contract.
type TrebSafeTransactionQueued struct {
	SafeTxHash     [32]byte
	Safe           common.Address
	Proposer       common.Address
	TransactionIds [][32]byte
	Raw            *types.Log // Blockchain specific contextual infos
}

const TrebSafeTransactionQueuedEventName = "SafeTransactionQueued"

// ContractEventName returns the user-defined event name.
func (TrebSafeTransactionQueued) ContractEventName() string {
	return TrebSafeTransactionQueuedEventName
}

// UnpackSafeTransactionQueuedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event SafeTransactionQueued(bytes32 indexed safeTxHash, address indexed safe, address indexed proposer, bytes32[] transactionIds)
func (treb *Treb) UnpackSafeTransactionQueuedEvent(log *types.Log) (*TrebSafeTransactionQueued, error) {
	event := "SafeTransactionQueued"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebSafeTransactionQueued)
	if len(log.Data) > 0 {
		if err := treb.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range treb.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// TrebTransactionSimulated represents a TransactionSimulated event raised by the Treb contract.
type TrebTransactionSimulated struct {
	SimulatedTx SimulatedTransaction
	Raw         *types.Log // Blockchain specific contextual infos
}

const TrebTransactionSimulatedEventName = "TransactionSimulated"

// ContractEventName returns the user-defined event name.
func (TrebTransactionSimulated) ContractEventName() string {
	return TrebTransactionSimulatedEventName
}

// UnpackTransactionSimulatedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event TransactionSimulated((bytes32,bytes32,address,bytes,(address,bytes,uint256)) simulatedTx)
func (treb *Treb) UnpackTransactionSimulatedEvent(log *types.Log) (*TrebTransactionSimulated, error) {
	event := "TransactionSimulated"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebTransactionSimulated)
	if len(log.Data) > 0 {
		if err := treb.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range treb.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

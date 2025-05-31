// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package treb

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

// DeployerEventDeployment is an auto generated low-level Go binding around an user-defined struct.
type DeployerEventDeployment struct {
	Artifact        string
	Label           string
	Entropy         string
	Salt            [32]byte
	BytecodeHash    [32]byte
	InitCodeHash    [32]byte
	ConstructorArgs []byte
	CreateStrategy  string
}

// RichTransaction is an auto generated low-level Go binding around an user-defined struct.
type RichTransaction struct {
	Transaction         Transaction
	TransactionId       [32]byte
	SenderId            [32]byte
	Status              uint8
	SimulatedReturnData []byte
	ExecutedReturnData  []byte
}

// Transaction is an auto generated low-level Go binding around an user-defined struct.
type Transaction struct {
	Label string
	To    common.Address
	Data  []byte
	Value *big.Int
}

// TrebMetaData contains all meta data concerning the Treb contract.
var TrebMetaData = bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"IS_SCRIPT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"execute\",\"inputs\":[{\"name\":\"_senderId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_transactions\",\"type\":\"tuple[]\",\"internalType\":\"structTransaction[]\",\"components\":[{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"bundleTransactions\",\"type\":\"tuple[]\",\"internalType\":\"structRichTransaction[]\",\"components\":[{\"name\":\"transaction\",\"type\":\"tuple\",\"internalType\":\"structTransaction\",\"components\":[{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"transactionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"senderId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"enumTransactionStatus\"},{\"name\":\"simulatedReturnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"executedReturnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"execute\",\"inputs\":[{\"name\":\"_senderId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_transaction\",\"type\":\"tuple\",\"internalType\":\"structTransaction\",\"components\":[{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"bundleTransaction\",\"type\":\"tuple\",\"internalType\":\"structRichTransaction\",\"components\":[{\"name\":\"transaction\",\"type\":\"tuple\",\"internalType\":\"structTransaction\",\"components\":[{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"transactionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"senderId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"enumTransactionStatus\"},{\"name\":\"simulatedReturnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"executedReturnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"lookup\",\"inputs\":[{\"name\":\"_identifier\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"_env\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"_chainId\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lookup\",\"inputs\":[{\"name\":\"_identifier\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"_env\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lookup\",\"inputs\":[{\"name\":\"_identifier\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"BroadcastStarted\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ContractDeployed\",\"inputs\":[{\"name\":\"deployer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"location\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"transactionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"deployment\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structDeployer.EventDeployment\",\"components\":[{\"name\":\"artifact\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"entropy\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"bytecodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"constructorArgs\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"createStrategy\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DeployingContract\",\"inputs\":[{\"name\":\"what\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"label\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"initCodeHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SafeTransactionQueued\",\"inputs\":[{\"name\":\"safeTxHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"safe\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"transactions\",\"type\":\"tuple[]\",\"indexed\":false,\"internalType\":\"structRichTransaction[]\",\"components\":[{\"name\":\"transaction\",\"type\":\"tuple\",\"internalType\":\"structTransaction\",\"components\":[{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"transactionId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"senderId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"enumTransactionStatus\"},{\"name\":\"simulatedReturnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"executedReturnData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TransactionBroadcast\",\"inputs\":[{\"name\":\"transactionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"label\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"returnData\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TransactionFailed\",\"inputs\":[{\"name\":\"transactionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"label\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TransactionSimulated\",\"inputs\":[{\"name\":\"transactionId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"label\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"returnData\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"CustomQueueReceiverNotImplemented\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EmptyTransactionArray\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidTargetAddress\",\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"NoSenderInitConfigs\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SenderNotFound\",\"inputs\":[{\"name\":\"id\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
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

// PackISSCRIPT is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf8ccbf47.
//
// Solidity: function IS_SCRIPT() view returns(bool)
func (treb *Treb) PackISSCRIPT() []byte {
	enc, err := treb.abi.Pack("IS_SCRIPT")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackISSCRIPT is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf8ccbf47.
//
// Solidity: function IS_SCRIPT() view returns(bool)
func (treb *Treb) UnpackISSCRIPT(data []byte) (bool, error) {
	out, err := treb.abi.Unpack("IS_SCRIPT", data)
	if err != nil {
		return *new(bool), err
	}
	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	return out0, err
}

// PackExecute is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x0865dd0d.
//
// Solidity: function execute(bytes32 _senderId, (string,address,bytes,uint256)[] _transactions) returns(((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes)[] bundleTransactions)
func (treb *Treb) PackExecute(senderId [32]byte, transactions []Transaction) []byte {
	enc, err := treb.abi.Pack("execute", senderId, transactions)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackExecute is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x0865dd0d.
//
// Solidity: function execute(bytes32 _senderId, (string,address,bytes,uint256)[] _transactions) returns(((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes)[] bundleTransactions)
func (treb *Treb) UnpackExecute(data []byte) ([]RichTransaction, error) {
	out, err := treb.abi.Unpack("execute", data)
	if err != nil {
		return *new([]RichTransaction), err
	}
	out0 := *abi.ConvertType(out[0], new([]RichTransaction)).(*[]RichTransaction)
	return out0, err
}

// PackExecute0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x541c56dc.
//
// Solidity: function execute(bytes32 _senderId, (string,address,bytes,uint256) _transaction) returns(((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes) bundleTransaction)
func (treb *Treb) PackExecute0(senderId [32]byte, transaction Transaction) []byte {
	enc, err := treb.abi.Pack("execute0", senderId, transaction)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackExecute0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x541c56dc.
//
// Solidity: function execute(bytes32 _senderId, (string,address,bytes,uint256) _transaction) returns(((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes) bundleTransaction)
func (treb *Treb) UnpackExecute0(data []byte) (RichTransaction, error) {
	out, err := treb.abi.Unpack("execute0", data)
	if err != nil {
		return *new(RichTransaction), err
	}
	out0 := *abi.ConvertType(out[0], new(RichTransaction)).(*RichTransaction)
	return out0, err
}

// PackLookup is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x0bd08a21.
//
// Solidity: function lookup(string _identifier, string _env, string _chainId) view returns(address)
func (treb *Treb) PackLookup(identifier string, env string, chainId string) []byte {
	enc, err := treb.abi.Pack("lookup", identifier, env, chainId)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackLookup is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x0bd08a21.
//
// Solidity: function lookup(string _identifier, string _env, string _chainId) view returns(address)
func (treb *Treb) UnpackLookup(data []byte) (common.Address, error) {
	out, err := treb.abi.Unpack("lookup", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackLookup0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x461dcd17.
//
// Solidity: function lookup(string _identifier, string _env) view returns(address)
func (treb *Treb) PackLookup0(identifier string, env string) []byte {
	enc, err := treb.abi.Pack("lookup0", identifier, env)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackLookup0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x461dcd17.
//
// Solidity: function lookup(string _identifier, string _env) view returns(address)
func (treb *Treb) UnpackLookup0(data []byte) (common.Address, error) {
	out, err := treb.abi.Unpack("lookup0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackLookup1 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf67187ac.
//
// Solidity: function lookup(string _identifier) view returns(address)
func (treb *Treb) PackLookup1(identifier string) []byte {
	enc, err := treb.abi.Pack("lookup1", identifier)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackLookup1 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf67187ac.
//
// Solidity: function lookup(string _identifier) view returns(address)
func (treb *Treb) UnpackLookup1(data []byte) (common.Address, error) {
	out, err := treb.abi.Unpack("lookup1", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// TrebBroadcastStarted represents a BroadcastStarted event raised by the Treb contract.
type TrebBroadcastStarted struct {
	Raw *types.Log // Blockchain specific contextual infos
}

const TrebBroadcastStartedEventName = "BroadcastStarted"

// ContractEventName returns the user-defined event name.
func (TrebBroadcastStarted) ContractEventName() string {
	return TrebBroadcastStartedEventName
}

// UnpackBroadcastStartedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event BroadcastStarted()
func (treb *Treb) UnpackBroadcastStartedEvent(log *types.Log) (*TrebBroadcastStarted, error) {
	event := "BroadcastStarted"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebBroadcastStarted)
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

// TrebContractDeployed represents a ContractDeployed event raised by the Treb contract.
type TrebContractDeployed struct {
	Deployer      common.Address
	Location      common.Address
	TransactionId [32]byte
	Deployment    DeployerEventDeployment
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

// TrebDeployingContract represents a DeployingContract event raised by the Treb contract.
type TrebDeployingContract struct {
	What         string
	Label        string
	InitCodeHash [32]byte
	Raw          *types.Log // Blockchain specific contextual infos
}

const TrebDeployingContractEventName = "DeployingContract"

// ContractEventName returns the user-defined event name.
func (TrebDeployingContract) ContractEventName() string {
	return TrebDeployingContractEventName
}

// UnpackDeployingContractEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event DeployingContract(string what, string label, bytes32 initCodeHash)
func (treb *Treb) UnpackDeployingContractEvent(log *types.Log) (*TrebDeployingContract, error) {
	event := "DeployingContract"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebDeployingContract)
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
	SafeTxHash   [32]byte
	Safe         common.Address
	Proposer     common.Address
	Transactions []RichTransaction
	Raw          *types.Log // Blockchain specific contextual infos
}

const TrebSafeTransactionQueuedEventName = "SafeTransactionQueued"

// ContractEventName returns the user-defined event name.
func (TrebSafeTransactionQueued) ContractEventName() string {
	return TrebSafeTransactionQueuedEventName
}

// UnpackSafeTransactionQueuedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event SafeTransactionQueued(bytes32 indexed safeTxHash, address indexed safe, address indexed proposer, ((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes)[] transactions)
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

// TrebTransactionBroadcast represents a TransactionBroadcast event raised by the Treb contract.
type TrebTransactionBroadcast struct {
	TransactionId [32]byte
	Sender        common.Address
	To            common.Address
	Value         *big.Int
	Data          []byte
	Label         string
	ReturnData    []byte
	Raw           *types.Log // Blockchain specific contextual infos
}

const TrebTransactionBroadcastEventName = "TransactionBroadcast"

// ContractEventName returns the user-defined event name.
func (TrebTransactionBroadcast) ContractEventName() string {
	return TrebTransactionBroadcastEventName
}

// UnpackTransactionBroadcastEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event TransactionBroadcast(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label, bytes returnData)
func (treb *Treb) UnpackTransactionBroadcastEvent(log *types.Log) (*TrebTransactionBroadcast, error) {
	event := "TransactionBroadcast"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebTransactionBroadcast)
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

// TrebTransactionFailed represents a TransactionFailed event raised by the Treb contract.
type TrebTransactionFailed struct {
	TransactionId [32]byte
	Sender        common.Address
	To            common.Address
	Value         *big.Int
	Data          []byte
	Label         string
	Raw           *types.Log // Blockchain specific contextual infos
}

const TrebTransactionFailedEventName = "TransactionFailed"

// ContractEventName returns the user-defined event name.
func (TrebTransactionFailed) ContractEventName() string {
	return TrebTransactionFailedEventName
}

// UnpackTransactionFailedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event TransactionFailed(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label)
func (treb *Treb) UnpackTransactionFailedEvent(log *types.Log) (*TrebTransactionFailed, error) {
	event := "TransactionFailed"
	if log.Topics[0] != treb.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TrebTransactionFailed)
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
	TransactionId [32]byte
	Sender        common.Address
	To            common.Address
	Value         *big.Int
	Data          []byte
	Label         string
	ReturnData    []byte
	Raw           *types.Log // Blockchain specific contextual infos
}

const TrebTransactionSimulatedEventName = "TransactionSimulated"

// ContractEventName returns the user-defined event name.
func (TrebTransactionSimulated) ContractEventName() string {
	return TrebTransactionSimulatedEventName
}

// UnpackTransactionSimulatedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event TransactionSimulated(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label, bytes returnData)
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

// UnpackError attempts to decode the provided error data using user-defined
// error definitions.
func (treb *Treb) UnpackError(raw []byte) (any, error) {
	if bytes.Equal(raw[:4], treb.abi.Errors["CustomQueueReceiverNotImplemented"].ID.Bytes()[:4]) {
		return treb.UnpackCustomQueueReceiverNotImplementedError(raw[4:])
	}
	if bytes.Equal(raw[:4], treb.abi.Errors["EmptyTransactionArray"].ID.Bytes()[:4]) {
		return treb.UnpackEmptyTransactionArrayError(raw[4:])
	}
	if bytes.Equal(raw[:4], treb.abi.Errors["InvalidTargetAddress"].ID.Bytes()[:4]) {
		return treb.UnpackInvalidTargetAddressError(raw[4:])
	}
	if bytes.Equal(raw[:4], treb.abi.Errors["NoSenderInitConfigs"].ID.Bytes()[:4]) {
		return treb.UnpackNoSenderInitConfigsError(raw[4:])
	}
	if bytes.Equal(raw[:4], treb.abi.Errors["SenderNotFound"].ID.Bytes()[:4]) {
		return treb.UnpackSenderNotFoundError(raw[4:])
	}
	return nil, errors.New("Unknown error")
}

// TrebCustomQueueReceiverNotImplemented represents a CustomQueueReceiverNotImplemented error raised by the Treb contract.
type TrebCustomQueueReceiverNotImplemented struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error CustomQueueReceiverNotImplemented()
func TrebCustomQueueReceiverNotImplementedErrorID() common.Hash {
	return common.HexToHash("0x1b80173970aeb30cfd8e35afc2c79106c2874900bf5cdf32547ef08a971f2f60")
}

// UnpackCustomQueueReceiverNotImplementedError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error CustomQueueReceiverNotImplemented()
func (treb *Treb) UnpackCustomQueueReceiverNotImplementedError(raw []byte) (*TrebCustomQueueReceiverNotImplemented, error) {
	out := new(TrebCustomQueueReceiverNotImplemented)
	if err := treb.abi.UnpackIntoInterface(out, "CustomQueueReceiverNotImplemented", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// TrebEmptyTransactionArray represents a EmptyTransactionArray error raised by the Treb contract.
type TrebEmptyTransactionArray struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error EmptyTransactionArray()
func TrebEmptyTransactionArrayErrorID() common.Hash {
	return common.HexToHash("0x392022c4e9c207e547ee6e42db9adc65756a4a781289324a6c350573ccd27ba0")
}

// UnpackEmptyTransactionArrayError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error EmptyTransactionArray()
func (treb *Treb) UnpackEmptyTransactionArrayError(raw []byte) (*TrebEmptyTransactionArray, error) {
	out := new(TrebEmptyTransactionArray)
	if err := treb.abi.UnpackIntoInterface(out, "EmptyTransactionArray", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// TrebInvalidTargetAddress represents a InvalidTargetAddress error raised by the Treb contract.
type TrebInvalidTargetAddress struct {
	Index *big.Int
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error InvalidTargetAddress(uint256 index)
func TrebInvalidTargetAddressErrorID() common.Hash {
	return common.HexToHash("0xcdb7c76613539b6b7214f147513a62e03ef96ab602eacdf819cc3c64d6839a2b")
}

// UnpackInvalidTargetAddressError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error InvalidTargetAddress(uint256 index)
func (treb *Treb) UnpackInvalidTargetAddressError(raw []byte) (*TrebInvalidTargetAddress, error) {
	out := new(TrebInvalidTargetAddress)
	if err := treb.abi.UnpackIntoInterface(out, "InvalidTargetAddress", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// TrebNoSenderInitConfigs represents a NoSenderInitConfigs error raised by the Treb contract.
type TrebNoSenderInitConfigs struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error NoSenderInitConfigs()
func TrebNoSenderInitConfigsErrorID() common.Hash {
	return common.HexToHash("0x5de8e6797becd38c74645e2b3db24718635ed9558d33a24dc785559f8bc38781")
}

// UnpackNoSenderInitConfigsError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error NoSenderInitConfigs()
func (treb *Treb) UnpackNoSenderInitConfigsError(raw []byte) (*TrebNoSenderInitConfigs, error) {
	out := new(TrebNoSenderInitConfigs)
	if err := treb.abi.UnpackIntoInterface(out, "NoSenderInitConfigs", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// TrebSenderNotFound represents a SenderNotFound error raised by the Treb contract.
type TrebSenderNotFound struct {
	Id string
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error SenderNotFound(string id)
func TrebSenderNotFoundErrorID() common.Hash {
	return common.HexToHash("0x0ee3aceab08b2df29eaca46f644104e2473b8cc7da5e4a3dcc785a85595d4ad4")
}

// UnpackSenderNotFoundError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error SenderNotFound(string id)
func (treb *Treb) UnpackSenderNotFoundError(raw []byte) (*TrebSenderNotFound, error) {
	out := new(TrebSenderNotFound)
	if err := treb.abi.UnpackIntoInterface(out, "SenderNotFound", raw); err != nil {
		return nil, err
	}
	return out, nil
}

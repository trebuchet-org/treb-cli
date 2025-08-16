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

// ICreateXValues is an auto generated low-level Go binding around an user-defined struct.
type ICreateXValues struct {
	ConstructorAmount *big.Int
	InitCallAmount    *big.Int
}

// CreateXMetaData contains all meta data concerning the CreateX contract.
var CreateXMetaData = bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"computeCreate2Address\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"computedAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"computeCreate2Address\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCodeHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"deployer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"computedAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"computeCreate3Address\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"deployer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"computedAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"computeCreate3Address\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"computedAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"computeCreateAddress\",\"inputs\":[{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"computedAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"computeCreateAddress\",\"inputs\":[{\"name\":\"deployer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"computedAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deployCreate\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2AndInit\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"refundAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2AndInit\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2AndInit\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"refundAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2AndInit\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2Clone\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"implementation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate2Clone\",\"inputs\":[{\"name\":\"implementation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate3\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate3\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate3AndInit\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate3AndInit\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate3AndInit\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"refundAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreate3AndInit\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"refundAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreateAndInit\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreateAndInit\",\"inputs\":[{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"values\",\"type\":\"tuple\",\"internalType\":\"structICreateX.Values\",\"components\":[{\"name\":\"constructorAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initCallAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"refundAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deployCreateClone\",\"inputs\":[{\"name\":\"implementation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"payable\"},{\"type\":\"event\",\"name\":\"ContractCreation\",\"inputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ContractCreation\",\"inputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Create3ProxyContractCreation\",\"inputs\":[{\"name\":\"newContract\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"FailedContractCreation\",\"inputs\":[{\"name\":\"emitter\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"FailedContractInitialisation\",\"inputs\":[{\"name\":\"emitter\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"revertData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"type\":\"error\",\"name\":\"FailedEtherTransfer\",\"inputs\":[{\"name\":\"emitter\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"revertData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"type\":\"error\",\"name\":\"InvalidNonceValue\",\"inputs\":[{\"name\":\"emitter\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"InvalidSalt\",\"inputs\":[{\"name\":\"emitter\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
	ID:  "CreateX",
}

// CreateX is an auto generated Go binding around an Ethereum contract.
type CreateX struct {
	abi abi.ABI
}

// NewCreateX creates a new instance of CreateX.
func NewCreateX() *CreateX {
	parsed, err := CreateXMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &CreateX{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *CreateX) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackComputeCreate2Address is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x890c283b.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function computeCreate2Address(bytes32 salt, bytes32 initCodeHash) view returns(address computedAddress)
func (createX *CreateX) PackComputeCreate2Address(salt [32]byte, initCodeHash [32]byte) []byte {
	enc, err := createX.abi.Pack("computeCreate2Address", salt, initCodeHash)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackComputeCreate2Address is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x890c283b.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function computeCreate2Address(bytes32 salt, bytes32 initCodeHash) view returns(address computedAddress)
func (createX *CreateX) TryPackComputeCreate2Address(salt [32]byte, initCodeHash [32]byte) ([]byte, error) {
	return createX.abi.Pack("computeCreate2Address", salt, initCodeHash)
}

// UnpackComputeCreate2Address is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x890c283b.
//
// Solidity: function computeCreate2Address(bytes32 salt, bytes32 initCodeHash) view returns(address computedAddress)
func (createX *CreateX) UnpackComputeCreate2Address(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("computeCreate2Address", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackComputeCreate2Address0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd323826a.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function computeCreate2Address(bytes32 salt, bytes32 initCodeHash, address deployer) pure returns(address computedAddress)
func (createX *CreateX) PackComputeCreate2Address0(salt [32]byte, initCodeHash [32]byte, deployer common.Address) []byte {
	enc, err := createX.abi.Pack("computeCreate2Address0", salt, initCodeHash, deployer)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackComputeCreate2Address0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd323826a.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function computeCreate2Address(bytes32 salt, bytes32 initCodeHash, address deployer) pure returns(address computedAddress)
func (createX *CreateX) TryPackComputeCreate2Address0(salt [32]byte, initCodeHash [32]byte, deployer common.Address) ([]byte, error) {
	return createX.abi.Pack("computeCreate2Address0", salt, initCodeHash, deployer)
}

// UnpackComputeCreate2Address0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xd323826a.
//
// Solidity: function computeCreate2Address(bytes32 salt, bytes32 initCodeHash, address deployer) pure returns(address computedAddress)
func (createX *CreateX) UnpackComputeCreate2Address0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("computeCreate2Address0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackComputeCreate3Address is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x42d654fc.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function computeCreate3Address(bytes32 salt, address deployer) pure returns(address computedAddress)
func (createX *CreateX) PackComputeCreate3Address(salt [32]byte, deployer common.Address) []byte {
	enc, err := createX.abi.Pack("computeCreate3Address", salt, deployer)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackComputeCreate3Address is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x42d654fc.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function computeCreate3Address(bytes32 salt, address deployer) pure returns(address computedAddress)
func (createX *CreateX) TryPackComputeCreate3Address(salt [32]byte, deployer common.Address) ([]byte, error) {
	return createX.abi.Pack("computeCreate3Address", salt, deployer)
}

// UnpackComputeCreate3Address is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x42d654fc.
//
// Solidity: function computeCreate3Address(bytes32 salt, address deployer) pure returns(address computedAddress)
func (createX *CreateX) UnpackComputeCreate3Address(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("computeCreate3Address", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackComputeCreate3Address0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x6cec2536.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function computeCreate3Address(bytes32 salt) view returns(address computedAddress)
func (createX *CreateX) PackComputeCreate3Address0(salt [32]byte) []byte {
	enc, err := createX.abi.Pack("computeCreate3Address0", salt)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackComputeCreate3Address0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x6cec2536.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function computeCreate3Address(bytes32 salt) view returns(address computedAddress)
func (createX *CreateX) TryPackComputeCreate3Address0(salt [32]byte) ([]byte, error) {
	return createX.abi.Pack("computeCreate3Address0", salt)
}

// UnpackComputeCreate3Address0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x6cec2536.
//
// Solidity: function computeCreate3Address(bytes32 salt) view returns(address computedAddress)
func (createX *CreateX) UnpackComputeCreate3Address0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("computeCreate3Address0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackComputeCreateAddress is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x28ddd046.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function computeCreateAddress(uint256 nonce) view returns(address computedAddress)
func (createX *CreateX) PackComputeCreateAddress(nonce *big.Int) []byte {
	enc, err := createX.abi.Pack("computeCreateAddress", nonce)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackComputeCreateAddress is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x28ddd046.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function computeCreateAddress(uint256 nonce) view returns(address computedAddress)
func (createX *CreateX) TryPackComputeCreateAddress(nonce *big.Int) ([]byte, error) {
	return createX.abi.Pack("computeCreateAddress", nonce)
}

// UnpackComputeCreateAddress is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x28ddd046.
//
// Solidity: function computeCreateAddress(uint256 nonce) view returns(address computedAddress)
func (createX *CreateX) UnpackComputeCreateAddress(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("computeCreateAddress", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackComputeCreateAddress0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x74637a7a.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function computeCreateAddress(address deployer, uint256 nonce) view returns(address computedAddress)
func (createX *CreateX) PackComputeCreateAddress0(deployer common.Address, nonce *big.Int) []byte {
	enc, err := createX.abi.Pack("computeCreateAddress0", deployer, nonce)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackComputeCreateAddress0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x74637a7a.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function computeCreateAddress(address deployer, uint256 nonce) view returns(address computedAddress)
func (createX *CreateX) TryPackComputeCreateAddress0(deployer common.Address, nonce *big.Int) ([]byte, error) {
	return createX.abi.Pack("computeCreateAddress0", deployer, nonce)
}

// UnpackComputeCreateAddress0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x74637a7a.
//
// Solidity: function computeCreateAddress(address deployer, uint256 nonce) view returns(address computedAddress)
func (createX *CreateX) UnpackComputeCreateAddress0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("computeCreateAddress0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x27fe1822.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate(bytes initCode) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate(initCode []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate", initCode)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x27fe1822.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate(bytes initCode) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate(initCode []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate", initCode)
}

// UnpackDeployCreate is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x27fe1822.
//
// Solidity: function deployCreate(bytes initCode) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x26307668.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2(bytes32 salt, bytes initCode) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate2(salt [32]byte, initCode []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate2", salt, initCode)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x26307668.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2(bytes32 salt, bytes initCode) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate2(salt [32]byte, initCode []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate2", salt, initCode)
}

// UnpackDeployCreate2 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x26307668.
//
// Solidity: function deployCreate2(bytes32 salt, bytes initCode) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate2(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate20 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x26a32fc7.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2(bytes initCode) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate20(initCode []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate20", initCode)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate20 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x26a32fc7.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2(bytes initCode) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate20(initCode []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate20", initCode)
}

// UnpackDeployCreate20 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x26a32fc7.
//
// Solidity: function deployCreate2(bytes initCode) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate20(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate20", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2AndInit is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xa7db93f2.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate2AndInit(salt [32]byte, initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) []byte {
	enc, err := createX.abi.Pack("deployCreate2AndInit", salt, initCode, data, values, refundAddress)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2AndInit is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xa7db93f2.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate2AndInit(salt [32]byte, initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) ([]byte, error) {
	return createX.abi.Pack("deployCreate2AndInit", salt, initCode, data, values, refundAddress)
}

// UnpackDeployCreate2AndInit is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xa7db93f2.
//
// Solidity: function deployCreate2AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate2AndInit(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2AndInit", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2AndInit0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xc3fe107b.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2AndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate2AndInit0(initCode []byte, data []byte, values ICreateXValues) []byte {
	enc, err := createX.abi.Pack("deployCreate2AndInit0", initCode, data, values)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2AndInit0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xc3fe107b.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2AndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate2AndInit0(initCode []byte, data []byte, values ICreateXValues) ([]byte, error) {
	return createX.abi.Pack("deployCreate2AndInit0", initCode, data, values)
}

// UnpackDeployCreate2AndInit0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xc3fe107b.
//
// Solidity: function deployCreate2AndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate2AndInit0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2AndInit0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2AndInit1 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe437252a.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2AndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate2AndInit1(initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) []byte {
	enc, err := createX.abi.Pack("deployCreate2AndInit1", initCode, data, values, refundAddress)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2AndInit1 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe437252a.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2AndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate2AndInit1(initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) ([]byte, error) {
	return createX.abi.Pack("deployCreate2AndInit1", initCode, data, values, refundAddress)
}

// UnpackDeployCreate2AndInit1 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xe437252a.
//
// Solidity: function deployCreate2AndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate2AndInit1(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2AndInit1", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2AndInit2 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe96deee4.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate2AndInit2(salt [32]byte, initCode []byte, data []byte, values ICreateXValues) []byte {
	enc, err := createX.abi.Pack("deployCreate2AndInit2", salt, initCode, data, values)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2AndInit2 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe96deee4.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate2AndInit2(salt [32]byte, initCode []byte, data []byte, values ICreateXValues) ([]byte, error) {
	return createX.abi.Pack("deployCreate2AndInit2", salt, initCode, data, values)
}

// UnpackDeployCreate2AndInit2 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xe96deee4.
//
// Solidity: function deployCreate2AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate2AndInit2(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2AndInit2", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2Clone is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2852527a.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2Clone(bytes32 salt, address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) PackDeployCreate2Clone(salt [32]byte, implementation common.Address, data []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate2Clone", salt, implementation, data)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2Clone is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2852527a.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2Clone(bytes32 salt, address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) TryPackDeployCreate2Clone(salt [32]byte, implementation common.Address, data []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate2Clone", salt, implementation, data)
}

// UnpackDeployCreate2Clone is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2852527a.
//
// Solidity: function deployCreate2Clone(bytes32 salt, address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) UnpackDeployCreate2Clone(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2Clone", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate2Clone0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x81503da1.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate2Clone(address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) PackDeployCreate2Clone0(implementation common.Address, data []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate2Clone0", implementation, data)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate2Clone0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x81503da1.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate2Clone(address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) TryPackDeployCreate2Clone0(implementation common.Address, data []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate2Clone0", implementation, data)
}

// UnpackDeployCreate2Clone0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x81503da1.
//
// Solidity: function deployCreate2Clone(address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) UnpackDeployCreate2Clone0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate2Clone0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate3 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x7f565360.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate3(bytes initCode) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate3(initCode []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate3", initCode)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate3 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x7f565360.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate3(bytes initCode) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate3(initCode []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate3", initCode)
}

// UnpackDeployCreate3 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x7f565360.
//
// Solidity: function deployCreate3(bytes initCode) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate3(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate3", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate30 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x9c36a286.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate3(bytes32 salt, bytes initCode) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate30(salt [32]byte, initCode []byte) []byte {
	enc, err := createX.abi.Pack("deployCreate30", salt, initCode)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate30 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x9c36a286.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate3(bytes32 salt, bytes initCode) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate30(salt [32]byte, initCode []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreate30", salt, initCode)
}

// UnpackDeployCreate30 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x9c36a286.
//
// Solidity: function deployCreate3(bytes32 salt, bytes initCode) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate30(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate30", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate3AndInit is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x00d84acb.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate3AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate3AndInit(salt [32]byte, initCode []byte, data []byte, values ICreateXValues) []byte {
	enc, err := createX.abi.Pack("deployCreate3AndInit", salt, initCode, data, values)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate3AndInit is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x00d84acb.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate3AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate3AndInit(salt [32]byte, initCode []byte, data []byte, values ICreateXValues) ([]byte, error) {
	return createX.abi.Pack("deployCreate3AndInit", salt, initCode, data, values)
}

// UnpackDeployCreate3AndInit is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x00d84acb.
//
// Solidity: function deployCreate3AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate3AndInit(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate3AndInit", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate3AndInit0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2f990e3f.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate3AndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate3AndInit0(initCode []byte, data []byte, values ICreateXValues) []byte {
	enc, err := createX.abi.Pack("deployCreate3AndInit0", initCode, data, values)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate3AndInit0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2f990e3f.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate3AndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate3AndInit0(initCode []byte, data []byte, values ICreateXValues) ([]byte, error) {
	return createX.abi.Pack("deployCreate3AndInit0", initCode, data, values)
}

// UnpackDeployCreate3AndInit0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2f990e3f.
//
// Solidity: function deployCreate3AndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate3AndInit0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate3AndInit0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate3AndInit1 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xddda0acb.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate3AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate3AndInit1(salt [32]byte, initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) []byte {
	enc, err := createX.abi.Pack("deployCreate3AndInit1", salt, initCode, data, values, refundAddress)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate3AndInit1 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xddda0acb.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate3AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate3AndInit1(salt [32]byte, initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) ([]byte, error) {
	return createX.abi.Pack("deployCreate3AndInit1", salt, initCode, data, values, refundAddress)
}

// UnpackDeployCreate3AndInit1 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xddda0acb.
//
// Solidity: function deployCreate3AndInit(bytes32 salt, bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate3AndInit1(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate3AndInit1", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreate3AndInit2 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf5745aba.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreate3AndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) PackDeployCreate3AndInit2(initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) []byte {
	enc, err := createX.abi.Pack("deployCreate3AndInit2", initCode, data, values, refundAddress)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreate3AndInit2 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf5745aba.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreate3AndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreate3AndInit2(initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) ([]byte, error) {
	return createX.abi.Pack("deployCreate3AndInit2", initCode, data, values, refundAddress)
}

// UnpackDeployCreate3AndInit2 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf5745aba.
//
// Solidity: function deployCreate3AndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreate3AndInit2(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreate3AndInit2", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreateAndInit is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x31a7c8c8.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreateAndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) PackDeployCreateAndInit(initCode []byte, data []byte, values ICreateXValues) []byte {
	enc, err := createX.abi.Pack("deployCreateAndInit", initCode, data, values)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreateAndInit is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x31a7c8c8.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreateAndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreateAndInit(initCode []byte, data []byte, values ICreateXValues) ([]byte, error) {
	return createX.abi.Pack("deployCreateAndInit", initCode, data, values)
}

// UnpackDeployCreateAndInit is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x31a7c8c8.
//
// Solidity: function deployCreateAndInit(bytes initCode, bytes data, (uint256,uint256) values) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreateAndInit(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreateAndInit", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreateAndInit0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x98e81077.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreateAndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) PackDeployCreateAndInit0(initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) []byte {
	enc, err := createX.abi.Pack("deployCreateAndInit0", initCode, data, values, refundAddress)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreateAndInit0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x98e81077.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreateAndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) TryPackDeployCreateAndInit0(initCode []byte, data []byte, values ICreateXValues, refundAddress common.Address) ([]byte, error) {
	return createX.abi.Pack("deployCreateAndInit0", initCode, data, values, refundAddress)
}

// UnpackDeployCreateAndInit0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x98e81077.
//
// Solidity: function deployCreateAndInit(bytes initCode, bytes data, (uint256,uint256) values, address refundAddress) payable returns(address newContract)
func (createX *CreateX) UnpackDeployCreateAndInit0(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreateAndInit0", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// PackDeployCreateClone is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf9664498.  This method will panic if any
// invalid/nil inputs are passed.
//
// Solidity: function deployCreateClone(address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) PackDeployCreateClone(implementation common.Address, data []byte) []byte {
	enc, err := createX.abi.Pack("deployCreateClone", implementation, data)
	if err != nil {
		panic(err)
	}
	return enc
}

// TryPackDeployCreateClone is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf9664498.  This method will return an error
// if any inputs are invalid/nil.
//
// Solidity: function deployCreateClone(address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) TryPackDeployCreateClone(implementation common.Address, data []byte) ([]byte, error) {
	return createX.abi.Pack("deployCreateClone", implementation, data)
}

// UnpackDeployCreateClone is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf9664498.
//
// Solidity: function deployCreateClone(address implementation, bytes data) payable returns(address proxy)
func (createX *CreateX) UnpackDeployCreateClone(data []byte) (common.Address, error) {
	out, err := createX.abi.Unpack("deployCreateClone", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, nil
}

// CreateXContractCreation represents a ContractCreation event raised by the CreateX contract.
type CreateXContractCreation struct {
	NewContract common.Address
	Salt        [32]byte
	Raw         *types.Log // Blockchain specific contextual infos
}

const CreateXContractCreationEventName = "ContractCreation"

// ContractEventName returns the user-defined event name.
func (CreateXContractCreation) ContractEventName() string {
	return CreateXContractCreationEventName
}

// UnpackContractCreationEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event ContractCreation(address indexed newContract, bytes32 indexed salt)
func (createX *CreateX) UnpackContractCreationEvent(log *types.Log) (*CreateXContractCreation, error) {
	event := "ContractCreation"
	if log.Topics[0] != createX.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CreateXContractCreation)
	if len(log.Data) > 0 {
		if err := createX.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range createX.abi.Events[event].Inputs {
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

// CreateXContractCreation0 represents a ContractCreation0 event raised by the CreateX contract.
type CreateXContractCreation0 struct {
	NewContract common.Address
	Raw         *types.Log // Blockchain specific contextual infos
}

const CreateXContractCreation0EventName = "ContractCreation0"

// ContractEventName returns the user-defined event name.
func (CreateXContractCreation0) ContractEventName() string {
	return CreateXContractCreation0EventName
}

// UnpackContractCreation0Event is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event ContractCreation(address indexed newContract)
func (createX *CreateX) UnpackContractCreation0Event(log *types.Log) (*CreateXContractCreation0, error) {
	event := "ContractCreation0"
	if log.Topics[0] != createX.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CreateXContractCreation0)
	if len(log.Data) > 0 {
		if err := createX.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range createX.abi.Events[event].Inputs {
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

// CreateXCreate3ProxyContractCreation represents a Create3ProxyContractCreation event raised by the CreateX contract.
type CreateXCreate3ProxyContractCreation struct {
	NewContract common.Address
	Salt        [32]byte
	Raw         *types.Log // Blockchain specific contextual infos
}

const CreateXCreate3ProxyContractCreationEventName = "Create3ProxyContractCreation"

// ContractEventName returns the user-defined event name.
func (CreateXCreate3ProxyContractCreation) ContractEventName() string {
	return CreateXCreate3ProxyContractCreationEventName
}

// UnpackCreate3ProxyContractCreationEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Create3ProxyContractCreation(address indexed newContract, bytes32 indexed salt)
func (createX *CreateX) UnpackCreate3ProxyContractCreationEvent(log *types.Log) (*CreateXCreate3ProxyContractCreation, error) {
	event := "Create3ProxyContractCreation"
	if log.Topics[0] != createX.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CreateXCreate3ProxyContractCreation)
	if len(log.Data) > 0 {
		if err := createX.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range createX.abi.Events[event].Inputs {
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
func (createX *CreateX) UnpackError(raw []byte) (any, error) {
	if bytes.Equal(raw[:4], createX.abi.Errors["FailedContractCreation"].ID.Bytes()[:4]) {
		return createX.UnpackFailedContractCreationError(raw[4:])
	}
	if bytes.Equal(raw[:4], createX.abi.Errors["FailedContractInitialisation"].ID.Bytes()[:4]) {
		return createX.UnpackFailedContractInitialisationError(raw[4:])
	}
	if bytes.Equal(raw[:4], createX.abi.Errors["FailedEtherTransfer"].ID.Bytes()[:4]) {
		return createX.UnpackFailedEtherTransferError(raw[4:])
	}
	if bytes.Equal(raw[:4], createX.abi.Errors["InvalidNonceValue"].ID.Bytes()[:4]) {
		return createX.UnpackInvalidNonceValueError(raw[4:])
	}
	if bytes.Equal(raw[:4], createX.abi.Errors["InvalidSalt"].ID.Bytes()[:4]) {
		return createX.UnpackInvalidSaltError(raw[4:])
	}
	return nil, errors.New("Unknown error")
}

// CreateXFailedContractCreation represents a FailedContractCreation error raised by the CreateX contract.
type CreateXFailedContractCreation struct {
	Emitter common.Address
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error FailedContractCreation(address emitter)
func CreateXFailedContractCreationErrorID() common.Hash {
	return common.HexToHash("0xc05cee7adec1c7022c70b91bddcde5124ca9bd2894bcd20bdbeb98c4ccd6ad31")
}

// UnpackFailedContractCreationError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error FailedContractCreation(address emitter)
func (createX *CreateX) UnpackFailedContractCreationError(raw []byte) (*CreateXFailedContractCreation, error) {
	out := new(CreateXFailedContractCreation)
	if err := createX.abi.UnpackIntoInterface(out, "FailedContractCreation", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateXFailedContractInitialisation represents a FailedContractInitialisation error raised by the CreateX contract.
type CreateXFailedContractInitialisation struct {
	Emitter    common.Address
	RevertData []byte
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error FailedContractInitialisation(address emitter, bytes revertData)
func CreateXFailedContractInitialisationErrorID() common.Hash {
	return common.HexToHash("0xa57ca239dc21ebdb895858cd57c414f9c89f18ea5c815cb1e329c666d45236f0")
}

// UnpackFailedContractInitialisationError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error FailedContractInitialisation(address emitter, bytes revertData)
func (createX *CreateX) UnpackFailedContractInitialisationError(raw []byte) (*CreateXFailedContractInitialisation, error) {
	out := new(CreateXFailedContractInitialisation)
	if err := createX.abi.UnpackIntoInterface(out, "FailedContractInitialisation", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateXFailedEtherTransfer represents a FailedEtherTransfer error raised by the CreateX contract.
type CreateXFailedEtherTransfer struct {
	Emitter    common.Address
	RevertData []byte
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error FailedEtherTransfer(address emitter, bytes revertData)
func CreateXFailedEtherTransferErrorID() common.Hash {
	return common.HexToHash("0xc2b3f4452c5ac36c715121b95d78a40ac33806494b2975a8238b27da8a77e1e1")
}

// UnpackFailedEtherTransferError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error FailedEtherTransfer(address emitter, bytes revertData)
func (createX *CreateX) UnpackFailedEtherTransferError(raw []byte) (*CreateXFailedEtherTransfer, error) {
	out := new(CreateXFailedEtherTransfer)
	if err := createX.abi.UnpackIntoInterface(out, "FailedEtherTransfer", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateXInvalidNonceValue represents a InvalidNonceValue error raised by the CreateX contract.
type CreateXInvalidNonceValue struct {
	Emitter common.Address
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error InvalidNonceValue(address emitter)
func CreateXInvalidNonceValueErrorID() common.Hash {
	return common.HexToHash("0x3c55ab3b3cc44087e1906d945b58a9ee2cdac44c0773594f81c831027ddd8bc4")
}

// UnpackInvalidNonceValueError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error InvalidNonceValue(address emitter)
func (createX *CreateX) UnpackInvalidNonceValueError(raw []byte) (*CreateXInvalidNonceValue, error) {
	out := new(CreateXInvalidNonceValue)
	if err := createX.abi.UnpackIntoInterface(out, "InvalidNonceValue", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateXInvalidSalt represents a InvalidSalt error raised by the CreateX contract.
type CreateXInvalidSalt struct {
	Emitter common.Address
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error InvalidSalt(address emitter)
func CreateXInvalidSaltErrorID() common.Hash {
	return common.HexToHash("0x13b3a2a19cc002fe27dc4952e92fb58eb225aa1ce015e59c8ba9b607a2163fe9")
}

// UnpackInvalidSaltError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error InvalidSalt(address emitter)
func (createX *CreateX) UnpackInvalidSaltError(raw []byte) (*CreateXInvalidSalt, error) {
	out := new(CreateXInvalidSalt)
	if err := createX.abi.UnpackIntoInterface(out, "InvalidSalt", raw); err != nil {
		return nil, err
	}
	return out, nil
}

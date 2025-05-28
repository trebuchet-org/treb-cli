// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package proxy

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

// DeploymentConfig is an auto generated low-level Go binding around an user-defined struct.
type DeploymentConfig struct {
	Namespace      string
	Label          string
	DeploymentType uint8
	ExecutorConfig ExecutorConfig
}

// DeploymentResult is an auto generated low-level Go binding around an user-defined struct.
type DeploymentResult struct {
	Deployed        common.Address
	Predicted       common.Address
	Salt            [32]byte
	InitCode        []byte
	SafeTxHash      [32]byte
	ConstructorArgs []byte
	Status          string
	Strategy        string
	DeploymentType  string
}

// ExecutorConfig is an auto generated low-level Go binding around an user-defined struct.
type ExecutorConfig struct {
	Sender                 common.Address
	SenderType             uint8
	SenderPrivateKey       *big.Int
	SenderDerivationPath   string
	ProposerType           uint8
	Proposer               common.Address
	ProposerPrivateKey     *big.Int
	ProposerDerivationPath string
}

// ProxyDeploymentConfig is an auto generated low-level Go binding around an user-defined struct.
type ProxyDeploymentConfig struct {
	ImplementationAddress common.Address
	DeploymentConfig      DeploymentConfig
}

// ProxyDeploymentMetaData contains all meta data concerning the ProxyDeployment contract.
var ProxyDeploymentMetaData = bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"DEPLOYMENTS_FILE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"IS_SCRIPT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"artifactPath\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainId\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"computeCreate3Address\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"deployer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"create3\",\"inputs\":[{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"executor\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"executorConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"senderType\",\"type\":\"uint8\",\"internalType\":\"enumSenderType\"},{\"name\":\"senderPrivateKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"senderDerivationPath\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"proposerType\",\"type\":\"uint8\",\"internalType\":\"enumSenderType\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposerPrivateKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposerDerivationPath\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getDeployment\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getDeploymentByEnv\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"environment\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getFullyQualifiedId\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"hasDeployment\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"implementationAddress\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"implementationArtifactPath\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"namespace\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"predictAddress\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"run\",\"inputs\":[{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structDeploymentConfig\",\"components\":[{\"name\":\"namespace\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"deploymentType\",\"type\":\"uint8\",\"internalType\":\"enumDeploymentType\"},{\"name\":\"executorConfig\",\"type\":\"tuple\",\"internalType\":\"structExecutorConfig\",\"components\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"senderType\",\"type\":\"uint8\",\"internalType\":\"enumSenderType\"},{\"name\":\"senderPrivateKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"senderDerivationPath\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"proposerType\",\"type\":\"uint8\",\"internalType\":\"enumSenderType\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposerPrivateKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposerDerivationPath\",\"type\":\"string\",\"internalType\":\"string\"}]}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structDeploymentResult\",\"components\":[{\"name\":\"deployed\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"predicted\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"safeTxHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"constructorArgs\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"status\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"strategy\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"deploymentType\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"run\",\"inputs\":[{\"name\":\"_config\",\"type\":\"tuple\",\"internalType\":\"structProxyDeploymentConfig\",\"components\":[{\"name\":\"implementationAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"deploymentConfig\",\"type\":\"tuple\",\"internalType\":\"structDeploymentConfig\",\"components\":[{\"name\":\"namespace\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"deploymentType\",\"type\":\"uint8\",\"internalType\":\"enumDeploymentType\"},{\"name\":\"executorConfig\",\"type\":\"tuple\",\"internalType\":\"structExecutorConfig\",\"components\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"senderType\",\"type\":\"uint8\",\"internalType\":\"enumSenderType\"},{\"name\":\"senderPrivateKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"senderDerivationPath\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"proposerType\",\"type\":\"uint8\",\"internalType\":\"enumSenderType\"},{\"name\":\"proposer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proposerPrivateKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proposerDerivationPath\",\"type\":\"string\",\"internalType\":\"string\"}]}]}]}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structDeploymentResult\",\"components\":[{\"name\":\"deployed\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"predicted\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"initCode\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"safeTxHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"constructorArgs\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"status\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"strategy\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"deploymentType\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"strategy\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumDeployStrategy\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"SafeTransactionQueued\",\"inputs\":[{\"name\":\"safe\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"proposer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"safeTxHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"label\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"transactionCount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TransactionExecuted\",\"inputs\":[{\"name\":\"executor\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"target\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"label\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"status\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumExecutionStatus\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ApiKitUrlNotFound\",\"inputs\":[{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"CompilationArtifactsNotFound\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DeploymentAddressMismatch\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DeploymentAlreadyExists\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DeploymentFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DeploymentPendingSafe\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"TransactionFailed\",\"inputs\":[{\"name\":\"label\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"UnlinkedLibraries\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"UnsupportedDeployer\",\"inputs\":[{\"name\":\"deployerType\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
	ID:  "ProxyDeployment",
}

// ProxyDeployment is an auto generated Go binding around an Ethereum contract.
type ProxyDeployment struct {
	abi abi.ABI
}

// NewProxyDeployment creates a new instance of ProxyDeployment.
func NewProxyDeployment() *ProxyDeployment {
	parsed, err := ProxyDeploymentMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &ProxyDeployment{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *ProxyDeployment) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDEPLOYMENTSFILE is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x4e1c85cb.
//
// Solidity: function DEPLOYMENTS_FILE() view returns(string)
func (proxyDeployment *ProxyDeployment) PackDEPLOYMENTSFILE() []byte {
	enc, err := proxyDeployment.abi.Pack("DEPLOYMENTS_FILE")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDEPLOYMENTSFILE is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x4e1c85cb.
//
// Solidity: function DEPLOYMENTS_FILE() view returns(string)
func (proxyDeployment *ProxyDeployment) UnpackDEPLOYMENTSFILE(data []byte) (string, error) {
	out, err := proxyDeployment.abi.Unpack("DEPLOYMENTS_FILE", data)
	if err != nil {
		return *new(string), err
	}
	out0 := *abi.ConvertType(out[0], new(string)).(*string)
	return out0, err
}

// PackISSCRIPT is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf8ccbf47.
//
// Solidity: function IS_SCRIPT() view returns(bool)
func (proxyDeployment *ProxyDeployment) PackISSCRIPT() []byte {
	enc, err := proxyDeployment.abi.Pack("IS_SCRIPT")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackISSCRIPT is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf8ccbf47.
//
// Solidity: function IS_SCRIPT() view returns(bool)
func (proxyDeployment *ProxyDeployment) UnpackISSCRIPT(data []byte) (bool, error) {
	out, err := proxyDeployment.abi.Unpack("IS_SCRIPT", data)
	if err != nil {
		return *new(bool), err
	}
	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	return out0, err
}

// PackArtifactPath is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x8b2d2f7d.
//
// Solidity: function artifactPath() view returns(string)
func (proxyDeployment *ProxyDeployment) PackArtifactPath() []byte {
	enc, err := proxyDeployment.abi.Pack("artifactPath")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackArtifactPath is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x8b2d2f7d.
//
// Solidity: function artifactPath() view returns(string)
func (proxyDeployment *ProxyDeployment) UnpackArtifactPath(data []byte) (string, error) {
	out, err := proxyDeployment.abi.Unpack("artifactPath", data)
	if err != nil {
		return *new(string), err
	}
	out0 := *abi.ConvertType(out[0], new(string)).(*string)
	return out0, err
}

// PackChainId is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint256)
func (proxyDeployment *ProxyDeployment) PackChainId() []byte {
	enc, err := proxyDeployment.abi.Pack("chainId")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackChainId is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint256)
func (proxyDeployment *ProxyDeployment) UnpackChainId(data []byte) (*big.Int, error) {
	out, err := proxyDeployment.abi.Unpack("chainId", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackComputeCreate3Address is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x42d654fc.
//
// Solidity: function computeCreate3Address(bytes32 salt, address deployer) pure returns(address)
func (proxyDeployment *ProxyDeployment) PackComputeCreate3Address(salt [32]byte, deployer common.Address) []byte {
	enc, err := proxyDeployment.abi.Pack("computeCreate3Address", salt, deployer)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackComputeCreate3Address is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x42d654fc.
//
// Solidity: function computeCreate3Address(bytes32 salt, address deployer) pure returns(address)
func (proxyDeployment *ProxyDeployment) UnpackComputeCreate3Address(data []byte) (common.Address, error) {
	out, err := proxyDeployment.abi.Unpack("computeCreate3Address", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackCreate3 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2af25238.
//
// Solidity: function create3(bytes32 salt, bytes initCode) returns(address)
func (proxyDeployment *ProxyDeployment) PackCreate3(salt [32]byte, initCode []byte) []byte {
	enc, err := proxyDeployment.abi.Pack("create3", salt, initCode)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackCreate3 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2af25238.
//
// Solidity: function create3(bytes32 salt, bytes initCode) returns(address)
func (proxyDeployment *ProxyDeployment) UnpackCreate3(data []byte) (common.Address, error) {
	out, err := proxyDeployment.abi.Unpack("create3", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackExecutor is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xc34c08e5.
//
// Solidity: function executor() view returns(address)
func (proxyDeployment *ProxyDeployment) PackExecutor() []byte {
	enc, err := proxyDeployment.abi.Pack("executor")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackExecutor is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xc34c08e5.
//
// Solidity: function executor() view returns(address)
func (proxyDeployment *ProxyDeployment) UnpackExecutor(data []byte) (common.Address, error) {
	out, err := proxyDeployment.abi.Unpack("executor", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackExecutorConfig is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x80e2e998.
//
// Solidity: function executorConfig() view returns(address sender, uint8 senderType, uint256 senderPrivateKey, string senderDerivationPath, uint8 proposerType, address proposer, uint256 proposerPrivateKey, string proposerDerivationPath)
func (proxyDeployment *ProxyDeployment) PackExecutorConfig() []byte {
	enc, err := proxyDeployment.abi.Pack("executorConfig")
	if err != nil {
		panic(err)
	}
	return enc
}

// ExecutorConfigOutput serves as a container for the return parameters of contract
// method ExecutorConfig.
type ExecutorConfigOutput struct {
	Sender                 common.Address
	SenderType             uint8
	SenderPrivateKey       *big.Int
	SenderDerivationPath   string
	ProposerType           uint8
	Proposer               common.Address
	ProposerPrivateKey     *big.Int
	ProposerDerivationPath string
}

// UnpackExecutorConfig is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x80e2e998.
//
// Solidity: function executorConfig() view returns(address sender, uint8 senderType, uint256 senderPrivateKey, string senderDerivationPath, uint8 proposerType, address proposer, uint256 proposerPrivateKey, string proposerDerivationPath)
func (proxyDeployment *ProxyDeployment) UnpackExecutorConfig(data []byte) (ExecutorConfigOutput, error) {
	out, err := proxyDeployment.abi.Unpack("executorConfig", data)
	outstruct := new(ExecutorConfigOutput)
	if err != nil {
		return *outstruct, err
	}
	outstruct.Sender = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.SenderType = *abi.ConvertType(out[1], new(uint8)).(*uint8)
	outstruct.SenderPrivateKey = abi.ConvertType(out[2], new(big.Int)).(*big.Int)
	outstruct.SenderDerivationPath = *abi.ConvertType(out[3], new(string)).(*string)
	outstruct.ProposerType = *abi.ConvertType(out[4], new(uint8)).(*uint8)
	outstruct.Proposer = *abi.ConvertType(out[5], new(common.Address)).(*common.Address)
	outstruct.ProposerPrivateKey = abi.ConvertType(out[6], new(big.Int)).(*big.Int)
	outstruct.ProposerDerivationPath = *abi.ConvertType(out[7], new(string)).(*string)
	return *outstruct, err

}

// PackGetDeployment is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xa8091d97.
//
// Solidity: function getDeployment(string identifier) view returns(address)
func (proxyDeployment *ProxyDeployment) PackGetDeployment(identifier string) []byte {
	enc, err := proxyDeployment.abi.Pack("getDeployment", identifier)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetDeployment is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xa8091d97.
//
// Solidity: function getDeployment(string identifier) view returns(address)
func (proxyDeployment *ProxyDeployment) UnpackGetDeployment(data []byte) (common.Address, error) {
	out, err := proxyDeployment.abi.Unpack("getDeployment", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackGetDeploymentByEnv is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x3a2350d9.
//
// Solidity: function getDeploymentByEnv(string identifier, string environment) view returns(address)
func (proxyDeployment *ProxyDeployment) PackGetDeploymentByEnv(identifier string, environment string) []byte {
	enc, err := proxyDeployment.abi.Pack("getDeploymentByEnv", identifier, environment)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetDeploymentByEnv is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x3a2350d9.
//
// Solidity: function getDeploymentByEnv(string identifier, string environment) view returns(address)
func (proxyDeployment *ProxyDeployment) UnpackGetDeploymentByEnv(data []byte) (common.Address, error) {
	out, err := proxyDeployment.abi.Unpack("getDeploymentByEnv", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackGetFullyQualifiedId is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x00d7660e.
//
// Solidity: function getFullyQualifiedId(string identifier) view returns(string)
func (proxyDeployment *ProxyDeployment) PackGetFullyQualifiedId(identifier string) []byte {
	enc, err := proxyDeployment.abi.Pack("getFullyQualifiedId", identifier)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetFullyQualifiedId is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x00d7660e.
//
// Solidity: function getFullyQualifiedId(string identifier) view returns(string)
func (proxyDeployment *ProxyDeployment) UnpackGetFullyQualifiedId(data []byte) (string, error) {
	out, err := proxyDeployment.abi.Unpack("getFullyQualifiedId", data)
	if err != nil {
		return *new(string), err
	}
	out0 := *abi.ConvertType(out[0], new(string)).(*string)
	return out0, err
}

// PackHasDeployment is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x108eeb6b.
//
// Solidity: function hasDeployment(string identifier) view returns(bool)
func (proxyDeployment *ProxyDeployment) PackHasDeployment(identifier string) []byte {
	enc, err := proxyDeployment.abi.Pack("hasDeployment", identifier)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackHasDeployment is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x108eeb6b.
//
// Solidity: function hasDeployment(string identifier) view returns(bool)
func (proxyDeployment *ProxyDeployment) UnpackHasDeployment(data []byte) (bool, error) {
	out, err := proxyDeployment.abi.Unpack("hasDeployment", data)
	if err != nil {
		return *new(bool), err
	}
	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	return out0, err
}

// PackImplementationAddress is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xb97a2319.
//
// Solidity: function implementationAddress() view returns(address)
func (proxyDeployment *ProxyDeployment) PackImplementationAddress() []byte {
	enc, err := proxyDeployment.abi.Pack("implementationAddress")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackImplementationAddress is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xb97a2319.
//
// Solidity: function implementationAddress() view returns(address)
func (proxyDeployment *ProxyDeployment) UnpackImplementationAddress(data []byte) (common.Address, error) {
	out, err := proxyDeployment.abi.Unpack("implementationAddress", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackImplementationArtifactPath is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x00a5e3aa.
//
// Solidity: function implementationArtifactPath() view returns(string)
func (proxyDeployment *ProxyDeployment) PackImplementationArtifactPath() []byte {
	enc, err := proxyDeployment.abi.Pack("implementationArtifactPath")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackImplementationArtifactPath is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x00a5e3aa.
//
// Solidity: function implementationArtifactPath() view returns(string)
func (proxyDeployment *ProxyDeployment) UnpackImplementationArtifactPath(data []byte) (string, error) {
	out, err := proxyDeployment.abi.Unpack("implementationArtifactPath", data)
	if err != nil {
		return *new(string), err
	}
	out0 := *abi.ConvertType(out[0], new(string)).(*string)
	return out0, err
}

// PackNamespace is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x7c015a89.
//
// Solidity: function namespace() view returns(string)
func (proxyDeployment *ProxyDeployment) PackNamespace() []byte {
	enc, err := proxyDeployment.abi.Pack("namespace")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackNamespace is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x7c015a89.
//
// Solidity: function namespace() view returns(string)
func (proxyDeployment *ProxyDeployment) UnpackNamespace(data []byte) (string, error) {
	out, err := proxyDeployment.abi.Unpack("namespace", data)
	if err != nil {
		return *new(string), err
	}
	out0 := *abi.ConvertType(out[0], new(string)).(*string)
	return out0, err
}

// PackPredictAddress is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x0d1469ee.
//
// Solidity: function predictAddress() returns()
func (proxyDeployment *ProxyDeployment) PackPredictAddress() []byte {
	enc, err := proxyDeployment.abi.Pack("predictAddress")
	if err != nil {
		panic(err)
	}
	return enc
}

// PackRun is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe398ab32.
//
// Solidity: function run((string,string,uint8,(address,uint8,uint256,string,uint8,address,uint256,string)) _config) returns((address,address,bytes32,bytes,bytes32,bytes,string,string,string))
func (proxyDeployment *ProxyDeployment) PackRun(config DeploymentConfig) []byte {
	enc, err := proxyDeployment.abi.Pack("run", config)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackRun is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xe398ab32.
//
// Solidity: function run((string,string,uint8,(address,uint8,uint256,string,uint8,address,uint256,string)) _config) returns((address,address,bytes32,bytes,bytes32,bytes,string,string,string))
func (proxyDeployment *ProxyDeployment) UnpackRun(data []byte) (DeploymentResult, error) {
	out, err := proxyDeployment.abi.Unpack("run", data)
	if err != nil {
		return *new(DeploymentResult), err
	}
	out0 := *abi.ConvertType(out[0], new(DeploymentResult)).(*DeploymentResult)
	return out0, err
}

// PackRun0 is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xea8a5ee7.
//
// Solidity: function run((address,(string,string,uint8,(address,uint8,uint256,string,uint8,address,uint256,string))) _config) returns((address,address,bytes32,bytes,bytes32,bytes,string,string,string))
func (proxyDeployment *ProxyDeployment) PackRun0(config ProxyDeploymentConfig) []byte {
	enc, err := proxyDeployment.abi.Pack("run0", config)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackRun0 is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xea8a5ee7.
//
// Solidity: function run((address,(string,string,uint8,(address,uint8,uint256,string,uint8,address,uint256,string))) _config) returns((address,address,bytes32,bytes,bytes32,bytes,string,string,string))
func (proxyDeployment *ProxyDeployment) UnpackRun0(data []byte) (DeploymentResult, error) {
	out, err := proxyDeployment.abi.Unpack("run0", data)
	if err != nil {
		return *new(DeploymentResult), err
	}
	out0 := *abi.ConvertType(out[0], new(DeploymentResult)).(*DeploymentResult)
	return out0, err
}

// PackStrategy is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xa8c62e76.
//
// Solidity: function strategy() view returns(uint8)
func (proxyDeployment *ProxyDeployment) PackStrategy() []byte {
	enc, err := proxyDeployment.abi.Pack("strategy")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackStrategy is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xa8c62e76.
//
// Solidity: function strategy() view returns(uint8)
func (proxyDeployment *ProxyDeployment) UnpackStrategy(data []byte) (uint8, error) {
	out, err := proxyDeployment.abi.Unpack("strategy", data)
	if err != nil {
		return *new(uint8), err
	}
	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)
	return out0, err
}

// ProxyDeploymentSafeTransactionQueued represents a SafeTransactionQueued event raised by the ProxyDeployment contract.
type ProxyDeploymentSafeTransactionQueued struct {
	Safe             common.Address
	Proposer         common.Address
	SafeTxHash       [32]byte
	Label            string
	TransactionCount *big.Int
	Raw              *types.Log // Blockchain specific contextual infos
}

const ProxyDeploymentSafeTransactionQueuedEventName = "SafeTransactionQueued"

// ContractEventName returns the user-defined event name.
func (ProxyDeploymentSafeTransactionQueued) ContractEventName() string {
	return ProxyDeploymentSafeTransactionQueuedEventName
}

// UnpackSafeTransactionQueuedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event SafeTransactionQueued(address indexed safe, address indexed proposer, bytes32 safeTxHash, string label, uint256 transactionCount)
func (proxyDeployment *ProxyDeployment) UnpackSafeTransactionQueuedEvent(log *types.Log) (*ProxyDeploymentSafeTransactionQueued, error) {
	event := "SafeTransactionQueued"
	if log.Topics[0] != proxyDeployment.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(ProxyDeploymentSafeTransactionQueued)
	if len(log.Data) > 0 {
		if err := proxyDeployment.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range proxyDeployment.abi.Events[event].Inputs {
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

// ProxyDeploymentTransactionExecuted represents a TransactionExecuted event raised by the ProxyDeployment contract.
type ProxyDeploymentTransactionExecuted struct {
	Executor common.Address
	Target   common.Address
	Label    string
	Status   uint8
	Raw      *types.Log // Blockchain specific contextual infos
}

const ProxyDeploymentTransactionExecutedEventName = "TransactionExecuted"

// ContractEventName returns the user-defined event name.
func (ProxyDeploymentTransactionExecuted) ContractEventName() string {
	return ProxyDeploymentTransactionExecutedEventName
}

// UnpackTransactionExecutedEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event TransactionExecuted(address indexed executor, address indexed target, string label, uint8 status)
func (proxyDeployment *ProxyDeployment) UnpackTransactionExecutedEvent(log *types.Log) (*ProxyDeploymentTransactionExecuted, error) {
	event := "TransactionExecuted"
	if log.Topics[0] != proxyDeployment.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(ProxyDeploymentTransactionExecuted)
	if len(log.Data) > 0 {
		if err := proxyDeployment.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range proxyDeployment.abi.Events[event].Inputs {
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
func (proxyDeployment *ProxyDeployment) UnpackError(raw []byte) (any, error) {
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["ApiKitUrlNotFound"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackApiKitUrlNotFoundError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["CompilationArtifactsNotFound"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackCompilationArtifactsNotFoundError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["DeploymentAddressMismatch"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackDeploymentAddressMismatchError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["DeploymentAlreadyExists"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackDeploymentAlreadyExistsError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["DeploymentFailed"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackDeploymentFailedError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["DeploymentPendingSafe"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackDeploymentPendingSafeError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["TransactionFailed"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackTransactionFailedError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["UnlinkedLibraries"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackUnlinkedLibrariesError(raw[4:])
	}
	if bytes.Equal(raw[:4], proxyDeployment.abi.Errors["UnsupportedDeployer"].ID.Bytes()[:4]) {
		return proxyDeployment.UnpackUnsupportedDeployerError(raw[4:])
	}
	return nil, errors.New("Unknown error")
}

// ProxyDeploymentApiKitUrlNotFound represents a ApiKitUrlNotFound error raised by the ProxyDeployment contract.
type ProxyDeploymentApiKitUrlNotFound struct {
	ChainId *big.Int
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error ApiKitUrlNotFound(uint256 chainId)
func ProxyDeploymentApiKitUrlNotFoundErrorID() common.Hash {
	return common.HexToHash("0x2c1a39bcaf83c4eae3ca503f965eb4aea7158b94d60fcb26572866147916992d")
}

// UnpackApiKitUrlNotFoundError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error ApiKitUrlNotFound(uint256 chainId)
func (proxyDeployment *ProxyDeployment) UnpackApiKitUrlNotFoundError(raw []byte) (*ProxyDeploymentApiKitUrlNotFound, error) {
	out := new(ProxyDeploymentApiKitUrlNotFound)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "ApiKitUrlNotFound", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentCompilationArtifactsNotFound represents a CompilationArtifactsNotFound error raised by the ProxyDeployment contract.
type ProxyDeploymentCompilationArtifactsNotFound struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error CompilationArtifactsNotFound()
func ProxyDeploymentCompilationArtifactsNotFoundErrorID() common.Hash {
	return common.HexToHash("0x8aab59037f0938e1392598a292df9a2f7c426b46c961588d81a765ed8b2b7cd3")
}

// UnpackCompilationArtifactsNotFoundError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error CompilationArtifactsNotFound()
func (proxyDeployment *ProxyDeployment) UnpackCompilationArtifactsNotFoundError(raw []byte) (*ProxyDeploymentCompilationArtifactsNotFound, error) {
	out := new(ProxyDeploymentCompilationArtifactsNotFound)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "CompilationArtifactsNotFound", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentDeploymentAddressMismatch represents a DeploymentAddressMismatch error raised by the ProxyDeployment contract.
type ProxyDeploymentDeploymentAddressMismatch struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error DeploymentAddressMismatch()
func ProxyDeploymentDeploymentAddressMismatchErrorID() common.Hash {
	return common.HexToHash("0x62ec2069c0923c6970e6f5dd93dc3ef68a6d6799b630c9a16367eccdd8fe4b74")
}

// UnpackDeploymentAddressMismatchError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error DeploymentAddressMismatch()
func (proxyDeployment *ProxyDeployment) UnpackDeploymentAddressMismatchError(raw []byte) (*ProxyDeploymentDeploymentAddressMismatch, error) {
	out := new(ProxyDeploymentDeploymentAddressMismatch)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "DeploymentAddressMismatch", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentDeploymentAlreadyExists represents a DeploymentAlreadyExists error raised by the ProxyDeployment contract.
type ProxyDeploymentDeploymentAlreadyExists struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error DeploymentAlreadyExists()
func ProxyDeploymentDeploymentAlreadyExistsErrorID() common.Hash {
	return common.HexToHash("0x77c3669a6c75c08ffabee5d91ac04d982537324742d60c7d968a77216bef336f")
}

// UnpackDeploymentAlreadyExistsError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error DeploymentAlreadyExists()
func (proxyDeployment *ProxyDeployment) UnpackDeploymentAlreadyExistsError(raw []byte) (*ProxyDeploymentDeploymentAlreadyExists, error) {
	out := new(ProxyDeploymentDeploymentAlreadyExists)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "DeploymentAlreadyExists", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentDeploymentFailed represents a DeploymentFailed error raised by the ProxyDeployment contract.
type ProxyDeploymentDeploymentFailed struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error DeploymentFailed()
func ProxyDeploymentDeploymentFailedErrorID() common.Hash {
	return common.HexToHash("0x3011642595b52db2c854a6ba7204cc406419c7d6c628eeb9fe80adcc365544e4")
}

// UnpackDeploymentFailedError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error DeploymentFailed()
func (proxyDeployment *ProxyDeployment) UnpackDeploymentFailedError(raw []byte) (*ProxyDeploymentDeploymentFailed, error) {
	out := new(ProxyDeploymentDeploymentFailed)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "DeploymentFailed", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentDeploymentPendingSafe represents a DeploymentPendingSafe error raised by the ProxyDeployment contract.
type ProxyDeploymentDeploymentPendingSafe struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error DeploymentPendingSafe()
func ProxyDeploymentDeploymentPendingSafeErrorID() common.Hash {
	return common.HexToHash("0x57df2cb9d9128bef276807d2fd8ef5f051765b189c43d4ee1dcd4499ecc36cfb")
}

// UnpackDeploymentPendingSafeError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error DeploymentPendingSafe()
func (proxyDeployment *ProxyDeployment) UnpackDeploymentPendingSafeError(raw []byte) (*ProxyDeploymentDeploymentPendingSafe, error) {
	out := new(ProxyDeploymentDeploymentPendingSafe)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "DeploymentPendingSafe", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentTransactionFailed represents a TransactionFailed error raised by the ProxyDeployment contract.
type ProxyDeploymentTransactionFailed struct {
	Label string
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error TransactionFailed(string label)
func ProxyDeploymentTransactionFailedErrorID() common.Hash {
	return common.HexToHash("0xed653df34a59bd648b20a2866d9e5c26d58b46110e9aa1782a50fc977defd1b0")
}

// UnpackTransactionFailedError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error TransactionFailed(string label)
func (proxyDeployment *ProxyDeployment) UnpackTransactionFailedError(raw []byte) (*ProxyDeploymentTransactionFailed, error) {
	out := new(ProxyDeploymentTransactionFailed)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "TransactionFailed", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentUnlinkedLibraries represents a UnlinkedLibraries error raised by the ProxyDeployment contract.
type ProxyDeploymentUnlinkedLibraries struct {
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error UnlinkedLibraries()
func ProxyDeploymentUnlinkedLibrariesErrorID() common.Hash {
	return common.HexToHash("0x8b027d8a2af60cfc45741220ae5e7cfeacfde5a84814e7c3b4d9f1620c996fb4")
}

// UnpackUnlinkedLibrariesError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error UnlinkedLibraries()
func (proxyDeployment *ProxyDeployment) UnpackUnlinkedLibrariesError(raw []byte) (*ProxyDeploymentUnlinkedLibraries, error) {
	out := new(ProxyDeploymentUnlinkedLibraries)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "UnlinkedLibraries", raw); err != nil {
		return nil, err
	}
	return out, nil
}

// ProxyDeploymentUnsupportedDeployer represents a UnsupportedDeployer error raised by the ProxyDeployment contract.
type ProxyDeploymentUnsupportedDeployer struct {
	DeployerType string
}

// ErrorID returns the hash of canonical representation of the error's signature.
//
// Solidity: error UnsupportedDeployer(string deployerType)
func ProxyDeploymentUnsupportedDeployerErrorID() common.Hash {
	return common.HexToHash("0x6644ecf07f5658e06f9580cb6fb6f9e36c49f5b6b75d6ddc1b21a5a2e0a17f44")
}

// UnpackUnsupportedDeployerError is the Go binding used to decode the provided
// error data into the corresponding Go error struct.
//
// Solidity: error UnsupportedDeployer(string deployerType)
func (proxyDeployment *ProxyDeployment) UnpackUnsupportedDeployerError(raw []byte) (*ProxyDeploymentUnsupportedDeployer, error) {
	out := new(ProxyDeploymentUnsupportedDeployer)
	if err := proxyDeployment.abi.UnpackIntoInterface(out, "UnsupportedDeployer", raw); err != nil {
		return nil, err
	}
	return out, nil
}

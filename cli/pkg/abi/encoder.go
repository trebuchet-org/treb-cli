package abi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// SenderType enum values
const (
	SenderTypePrivateKey = 0
	SenderTypeSafe       = 1
	SenderTypeLedger     = 2
)

// ExecutorConfig represents the executor configuration struct
type ExecutorConfig struct {
	SenderType               uint8          `abi:"senderType"`
	Sender                   common.Address `abi:"sender"`
	PrivateKey               *big.Int       `abi:"privateKey"`
	LedgerDerivationPath     string         `abi:"ledgerDerivationPath"`
	ProposerType             uint8          `abi:"proposerType"`
	Proposer                 common.Address `abi:"proposer"`
	ProposerPrivateKey       *big.Int       `abi:"proposerPrivateKey"`
	ProposerDerivationPath   string         `abi:"proposerDerivationPath"`
}

// DeploymentConfig represents the deployment configuration struct
type DeploymentConfig struct {
	Namespace      string         `abi:"namespace"`
	Label          string         `abi:"label"`
	ExecutorConfig ExecutorConfig `abi:"executorConfig"`
}

// ProxyDeploymentConfig represents the proxy deployment configuration struct
type ProxyDeploymentConfig struct {
	ImplementationAddress common.Address   `abi:"implementationAddress"`
	DeploymentConfig      DeploymentConfig `abi:"deploymentConfig"`
}

// LibraryDeploymentConfig represents the library deployment configuration struct
type LibraryDeploymentConfig struct {
	ExecutorConfig       ExecutorConfig `abi:"executorConfig"`
	LibraryName          string         `abi:"libraryName"`
	LibraryArtifactPath  string         `abi:"libraryArtifactPath"`
}

// EncodeDeploymentConfig encodes the deployment config for the run() method
func EncodeDeploymentConfig(config *DeploymentConfig, runMethodABI string) (string, error) {
	return encodeConfigForMethod(config, runMethodABI, "run")
}

// EncodeProxyDeploymentConfig encodes the proxy deployment config for the run() method
func EncodeProxyDeploymentConfig(config *ProxyDeploymentConfig, runMethodABI string) (string, error) {
	return encodeConfigForMethod(config, runMethodABI, "run")
}

// EncodeLibraryDeploymentConfig encodes the library deployment config for the run() method
func EncodeLibraryDeploymentConfig(config *LibraryDeploymentConfig, runMethodABI string) (string, error) {
	return encodeConfigForMethod(config, runMethodABI, "run")
}

// encodeConfigForMethod encodes any config struct for a specific method
func encodeConfigForMethod(config interface{}, runMethodABI string, methodName string) (string, error) {
	// Parse the method ABI
	parsedABI, err := abi.JSON(strings.NewReader(runMethodABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Get the specified method
	method, exists := parsedABI.Methods[methodName]
	if !exists {
		return "", fmt.Errorf("%s method not found in ABI", methodName)
	}

	// Check if the method expects parameters
	if len(method.Inputs) == 0 {
		// Legacy method without parameters
		return "", nil
	}

	// Encode the config struct
	encodedData, err := method.Inputs.Pack(config)
	if err != nil {
		return "", fmt.Errorf("failed to encode config: %w", err)
	}

	// Return as hex string
	return "0x" + hex.EncodeToString(encodedData), nil
}

// CreateExecutorConfig creates an executor config from deployment parameters
func CreateExecutorConfig(
	sender common.Address,
	senderType string,
	privateKey *big.Int,
	ledgerPath string,
	proposer common.Address,
	proposerType string,
	proposerPrivateKey *big.Int,
	proposerLedgerPath string,
) ExecutorConfig {
	var senderTypeEnum uint8
	switch senderType {
	case "private_key":
		senderTypeEnum = SenderTypePrivateKey
	case "safe":
		senderTypeEnum = SenderTypeSafe
	case "ledger":
		senderTypeEnum = SenderTypeLedger
	}

	var proposerTypeEnum uint8
	switch proposerType {
	case "private_key":
		proposerTypeEnum = SenderTypePrivateKey
	case "ledger":
		proposerTypeEnum = SenderTypeLedger
	}

	return ExecutorConfig{
		SenderType:             senderTypeEnum,
		Sender:                 sender,
		PrivateKey:             privateKey,
		LedgerDerivationPath:   ledgerPath,
		ProposerType:           proposerTypeEnum,
		Proposer:               proposer,
		ProposerPrivateKey:     proposerPrivateKey,
		ProposerDerivationPath: proposerLedgerPath,
	}
}

// CreateDeploymentConfig creates a deployment config from the context
func CreateDeploymentConfig(
	namespace string,
	label string,
	executorConfig ExecutorConfig,
) *DeploymentConfig {
	return &DeploymentConfig{
		Namespace:      namespace,
		Label:          label,
		ExecutorConfig: executorConfig,
	}
}

// CreateProxyDeploymentConfig creates a proxy deployment config
func CreateProxyDeploymentConfig(
	implementationAddress common.Address,
	deploymentConfig DeploymentConfig,
) *ProxyDeploymentConfig {
	return &ProxyDeploymentConfig{
		ImplementationAddress: implementationAddress,
		DeploymentConfig:      deploymentConfig,
	}
}

// CreateLibraryDeploymentConfig creates a library deployment config
func CreateLibraryDeploymentConfig(
	executorConfig ExecutorConfig,
	libraryName string,
	libraryArtifactPath string,
) *LibraryDeploymentConfig {
	return &LibraryDeploymentConfig{
		ExecutorConfig:      executorConfig,
		LibraryName:         libraryName,
		LibraryArtifactPath: libraryArtifactPath,
	}
}
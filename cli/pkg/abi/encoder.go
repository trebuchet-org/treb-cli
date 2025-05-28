package abi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/deployment"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/library"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/proxy"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// SenderType enum values - maps to Solidity's SenderType enum
const (
	SenderTypePrivateKey = 0
	SenderTypeSafe       = 1
	SenderTypeLedger     = 2
)

// DeploymentType enum values - maps to Solidity's DeploymentType enum
const (
	DeploymentTypeSingleton = 0
	DeploymentTypeProxy     = 1
	DeploymentTypeLibrary   = 2
)

// Type aliases for easier access
type ExecutorConfig = deployment.ExecutorConfig
type DeploymentConfig = deployment.DeploymentConfig
type DeploymentResult = deployment.DeploymentResult
type ProxyDeploymentConfig = proxy.ProxyDeploymentConfig
type LibraryDeploymentConfig = library.LibraryDeploymentConfig

// EncodeDeploymentConfig encodes the deployment config for the run() method
func EncodeDeploymentConfig(config *DeploymentConfig) (string, error) {
	contract := deployment.NewDeployment()
	data := contract.PackRun(*config)
	return "0x" + hex.EncodeToString(data), nil
}

// EncodeProxyDeploymentConfig encodes the proxy deployment config for the run() method
func EncodeProxyDeploymentConfig(config *ProxyDeploymentConfig) (string, error) {
	contract := proxy.NewProxyDeployment()
	// ProxyDeployment has two run methods, we need to use the one with ProxyDeploymentConfig
	data := contract.PackRun0(*config)
	return "0x" + hex.EncodeToString(data), nil
}

// EncodeLibraryDeploymentConfig encodes the library deployment config for the run() method
func EncodeLibraryDeploymentConfig(config *LibraryDeploymentConfig) (string, error) {
	contract := library.NewLibraryDeployment()
	data := contract.PackRun(*config)
	return "0x" + hex.EncodeToString(data), nil
}

// MethodInfo contains both the encoded data and the method signature
type MethodInfo struct {
	EncodedData string
	Signature   string
	Calldata    string // Full calldata with function selector
}

// EncodeDeploymentConfigWithSignature encodes the deployment config and returns signature
func EncodeDeploymentRun(config *DeploymentConfig) []byte {
	contract := deployment.NewDeployment()
	return contract.PackRun(*config)
}

func EncodeProxyDeploymentRun(config *ProxyDeploymentConfig) []byte {
	contract := proxy.NewProxyDeployment()
	// ProxyDeployment has two run methods, use the one with ProxyDeploymentConfig
	return contract.PackRun0(*config)
}

func EncodeLibraryDeploymentRun(config *LibraryDeploymentConfig) []byte {
	contract := library.NewLibraryDeployment()
	return contract.PackRun(*config)
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

	// Ensure nil values are handled properly for big.Int fields
	if privateKey == nil {
		privateKey = big.NewInt(0)
	}
	if proposerPrivateKey == nil {
		proposerPrivateKey = big.NewInt(0)
	}

	return ExecutorConfig{
		Sender:                 sender,
		SenderType:             senderTypeEnum,
		SenderPrivateKey:       privateKey,
		SenderDerivationPath:   ledgerPath,
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
	deploymentType types.DeploymentType,
	executorConfig ExecutorConfig,
) *DeploymentConfig {
	return &DeploymentConfig{
		Namespace:      namespace,
		Label:          label,
		DeploymentType: DeploymentTypeToUint8(deploymentType),
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
		DeploymentConfig:      ConvertDeploymentConfigToProxy(deploymentConfig),
	}
}

// ConvertDeploymentConfigToProxy converts deployment.DeploymentConfig to proxy.DeploymentConfig
func ConvertDeploymentConfigToProxy(config DeploymentConfig) proxy.DeploymentConfig {
	return castStruct[proxy.DeploymentConfig](config)
}

// ConvertExecutorConfigToLibrary converts deployment.ExecutorConfig to library.ExecutorConfig
func ConvertExecutorConfigToLibrary(config ExecutorConfig) library.ExecutorConfig {
	return castStruct[library.ExecutorConfig](config)
}

// castStruct uses JSON marshaling to convert between structs with identical fields
func castStruct[T any](src interface{}) T {
	var dst T
	data, err := json.Marshal(src)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal struct: %v", err))
	}
	if err := json.Unmarshal(data, &dst); err != nil {
		panic(fmt.Sprintf("failed to unmarshal struct: %v", err))
	}
	return dst
}

// CreateLibraryDeploymentConfig creates a library deployment config
func CreateLibraryDeploymentConfig(
	executorConfig ExecutorConfig,
	libraryName string,
	libraryArtifactPath string,
) *LibraryDeploymentConfig {
	return &LibraryDeploymentConfig{
		ExecutorConfig:      ConvertExecutorConfigToLibrary(executorConfig),
		LibraryArtifactPath: libraryArtifactPath,
	}
}

// ================ Enum Converters ================

// DeploymentTypeToUint8 converts types.DeploymentType to uint8 for ABI encoding
func DeploymentTypeToUint8(dt types.DeploymentType) uint8 {
	switch dt {
	case types.SingletonDeployment:
		return DeploymentTypeSingleton
	case types.ProxyDeployment:
		return DeploymentTypeProxy
	case types.LibraryDeployment:
		return DeploymentTypeLibrary
	default:
		return DeploymentTypeSingleton // Default to singleton
	}
}

// Uint8ToDeploymentType converts uint8 from ABI to types.DeploymentType
func Uint8ToDeploymentType(val uint8) types.DeploymentType {
	switch val {
	case DeploymentTypeSingleton:
		return types.SingletonDeployment
	case DeploymentTypeProxy:
		return types.ProxyDeployment
	case DeploymentTypeLibrary:
		return types.LibraryDeployment
	default:
		return types.UnknownDeployment
	}
}

// StatusFromString converts string status from Solidity to types.Status
func StatusFromString(status string) types.Status {
	switch status {
	case "EXECUTED":
		return types.StatusExecuted
	case "PENDING_SAFE":
		return types.StatusQueued
	default:
		return types.StatusUnknown
	}
}

// DeployStrategyFromString converts string strategy from Solidity to types.DeployStrategy
func DeployStrategyFromString(strategy string) types.DeployStrategy {
	switch strategy {
	case "CREATE2":
		return types.Create2Strategy
	case "CREATE3":
		return types.Create3Strategy
	default:
		return types.UnknownStrategy
	}
}

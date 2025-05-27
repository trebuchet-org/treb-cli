package abi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// DeploymentConfig represents the deployment configuration struct
type DeploymentConfig struct {
	ProjectName     string         `abi:"projectName"`
	Namespace       string         `abi:"namespace"`
	Label           string         `abi:"label"`
	ChainId         *big.Int       `abi:"chainId"`
	NetworkName     string         `abi:"networkName"`
	Sender          common.Address `abi:"sender"`
	SenderType      string         `abi:"senderType"`
	RegistryAddress common.Address `abi:"registryAddress"`
	Broadcast       bool           `abi:"broadcast"`
	Verify          bool           `abi:"verify"`
}

// EncodeDeploymentConfig encodes the deployment config for the run() method
func EncodeDeploymentConfig(config *DeploymentConfig, runMethodABI string) (string, error) {
	// Parse the method ABI
	parsedABI, err := abi.JSON(strings.NewReader(runMethodABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Get the run method
	method, exists := parsedABI.Methods["run"]
	if !exists {
		return "", fmt.Errorf("run method not found in ABI")
	}

	// Check if the method expects a DeploymentConfig struct as first parameter
	if len(method.Inputs) == 0 {
		// Legacy run() method without parameters
		return "", nil
	}

	// Encode the config struct
	encodedData, err := method.Inputs.Pack(config)
	if err != nil {
		return "", fmt.Errorf("failed to encode deployment config: %w", err)
	}

	// Return as hex string
	return "0x" + hex.EncodeToString(encodedData), nil
}

// CreateDeploymentConfig creates a deployment config from the context
func CreateDeploymentConfig(
	projectName string,
	namespace string,
	label string,
	chainId *big.Int,
	networkName string,
	sender common.Address,
	senderType string,
	registryAddress common.Address,
	broadcast bool,
	verify bool,
) *DeploymentConfig {
	return &DeploymentConfig{
		ProjectName:     projectName,
		Namespace:       namespace,
		Label:           label,
		ChainId:         chainId,
		NetworkName:     networkName,
		Sender:          sender,
		SenderType:      senderType,
		RegistryAddress: registryAddress,
		Broadcast:       broadcast,
		Verify:          verify,
	}
}
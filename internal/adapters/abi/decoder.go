package abi

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/trebuchet-org/treb-cli/internal/adapters/abi/bindings"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// EventDecoder decodes blockchain events using ABI bindings
type EventDecoder struct {
	trebABI   *abi.ABI
	createxABI *abi.ABI
}

// NewEventDecoder creates a new event decoder
func NewEventDecoder() (*EventDecoder, error) {
	// Get ABI objects from bindings
	trebABIParsed, err := abi.JSON(strings.NewReader(bindings.TrebMetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Treb ABI: %w", err)
	}

	createxABIParsed, err := abi.JSON(strings.NewReader(bindings.CreateXMetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CreateX ABI: %w", err)
	}

	return &EventDecoder{
		trebABI:    &trebABIParsed,
		createxABI: &createxABIParsed,
	}, nil
}

// DecodeLog decodes a log entry into a domain event
func (d *EventDecoder) DecodeLog(log types.Log) (*domain.DeploymentEvent, error) {
	// Debug logging
	if os.Getenv("TREB_TEST_DEBUG") != "" && len(log.Topics) > 0 {
		fmt.Printf("DEBUG: DecodeLog - trying to decode event with signature: %s\n", log.Topics[0].Hex())
	}
	
	// Try to decode as Treb event first
	if event, err := d.decodeTrebEvent(log); err == nil {
		return event, nil
	}

	// Try to decode as CreateX event
	if event, err := d.decodeCreateXEvent(log); err == nil {
		return event, nil
	}

	return nil, fmt.Errorf("unknown event")
}

// decodeTrebEvent attempts to decode a log as a Treb event
func (d *EventDecoder) decodeTrebEvent(log types.Log) (*domain.DeploymentEvent, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("no topics in log")
	}

	eventSig := log.Topics[0]

	// Check for ContractDeployed event
	contractDeployedSig := d.trebABI.Events["ContractDeployed"].ID
	if os.Getenv("TREB_TEST_DEBUG") != "" {
		fmt.Printf("DEBUG: ContractDeployed signature: %s, checking against: %s\n", contractDeployedSig.Hex(), eventSig.Hex())
	}
	if eventSig == contractDeployedSig {
		// Use the same structure as v1 bindings
		type DeploymentDetails struct {
			Artifact        string
			Label           string
			Entropy         string
			Salt            [32]byte
			BytecodeHash    [32]byte
			InitCodeHash    [32]byte
			ConstructorArgs []byte
			CreateStrategy  string
		}
		
		// Indexed parameters are in Topics
		// Topics[0] = event signature
		// Topics[1] = deployer (indexed)
		// Topics[2] = location (indexed)
		// Topics[3] = transactionId (indexed)
		if len(log.Topics) < 4 {
			return nil, fmt.Errorf("ContractDeployed event missing topics")
		}
		
		deployer := common.BytesToAddress(log.Topics[1].Bytes())
		location := common.BytesToAddress(log.Topics[2].Bytes())
		transactionId := log.Topics[3]
		
		// Non-indexed parameters are in Data
		var deployment DeploymentDetails
		err := d.trebABI.UnpackIntoInterface(&deployment, "ContractDeployed", log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack ContractDeployed event: %w", err)
		}

		// Extract contract name from artifact
		// Format is usually "path/to/Contract.sol:ContractName"
		contractName := deployment.Artifact
		if idx := strings.LastIndex(contractName, ":"); idx != -1 {
			contractName = contractName[idx+1:]
		}
		
		// Convert transactionId hash to [32]byte
		var txID [32]byte
		copy(txID[:], transactionId.Bytes())

		result := &domain.DeploymentEvent{
			EventType:     domain.EventContractDeployed,
			Address:       location.Hex(),
			ContractName:  contractName,
			Label:         deployment.Label,
			Deployer:      deployer.Hex(),
			Salt:          deployment.Salt,
			TransactionID: txID,
		}
		
		if os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: Decoded ContractDeployed - Address: %s, Contract: %s, Deployer: %s\n", 
				result.Address, result.ContractName, result.Deployer)
		}
		
		return result, nil
	}

	// Check for ProxyDeployed event
	proxyDeployedSig := d.trebABI.Events["ProxyDeployed"].ID
	if eventSig == proxyDeployedSig {
		var event struct {
			Proxy           common.Address
			Implementation  common.Address
			Contract        string
			Namespace       string
			ChainId         *big.Int
			Deployer        common.Address
			DeployTxHash    common.Hash
			Label           string
		}

		err := d.trebABI.UnpackIntoInterface(&event, "ProxyDeployed", log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack ProxyDeployed event: %w", err)
		}

		// Unpack indexed parameters
		if len(log.Topics) > 1 {
			event.Proxy = common.BytesToAddress(log.Topics[1].Bytes())
		}
		if len(log.Topics) > 2 {
			event.Implementation = common.BytesToAddress(log.Topics[2].Bytes())
		}
		if len(log.Topics) > 3 {
			event.Namespace = log.Topics[3].Hex()
		}

		return &domain.DeploymentEvent{
			EventType:      domain.EventProxyDeployed,
			Address:        event.Proxy.Hex(),
			Implementation: event.Implementation.Hex(),
			ContractName:   event.Contract,
			Namespace:      event.Namespace,
			ChainID:        event.ChainId.Uint64(),
			Deployer:       event.Deployer.Hex(),
			TxHash:         event.DeployTxHash.Hex(),
			Label:          event.Label,
		}, nil
	}

	return nil, fmt.Errorf("unknown Treb event")
}

// decodeCreateXEvent attempts to decode a log as a CreateX event
func (d *EventDecoder) decodeCreateXEvent(log types.Log) (*domain.DeploymentEvent, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("no topics in log")
	}

	eventSig := log.Topics[0]

	// Check for ContractCreation event
	contractCreationSig := d.createxABI.Events["ContractCreation"].ID
	if eventSig == contractCreationSig {
		var event struct {
			NewContract common.Address
			Salt        [32]byte
		}

		err := d.createxABI.UnpackIntoInterface(&event, "ContractCreation", log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack ContractCreation event: %w", err)
		}

		// Unpack indexed parameters
		if len(log.Topics) > 1 {
			event.NewContract = common.BytesToAddress(log.Topics[1].Bytes())
		}

		return &domain.DeploymentEvent{
			EventType: domain.EventCreateXContractCreation,
			Address:   event.NewContract.Hex(),
			Salt:      event.Salt,
		}, nil
	}

	return nil, fmt.Errorf("unknown CreateX event")
}
package events

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/bindings"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
)

// EventParser parses events from forge script output
type EventParser struct {
	trebContract *bindings.Treb
}

// NewEventParser creates a new event parser
func NewEventParser() *EventParser {
	return &EventParser{
		trebContract: bindings.NewTreb(),
	}
}

// ParseEvents parses all events from script output
func (ep *EventParser) ParseEvents(output *forge.ScriptOutput) ([]interface{}, error) {
	if output == nil || output.RawLogs == nil {
		return nil, nil
	}

	var parsedEvents []interface{}

	for _, rawLog := range output.RawLogs {
		if len(rawLog.Topics) == 0 {
			continue
		}

		event, err := ep.ParseEvent(rawLog)
		if err != nil {
			// Skip unknown events silently
			if !strings.Contains(err.Error(), "unknown event signature") {
				// Log warning for actual parsing errors
				fmt.Printf("Warning: failed to parse event %s: %v\n", rawLog.Topics[0].Hex(), err)
			}
			continue
		}

		parsedEvents = append(parsedEvents, event)
	}

	return parsedEvents, nil
}

// ParseEvent parses a single event log
func (ep *EventParser) ParseEvent(rawLog forge.EventLog) (interface{}, error) {
	if len(rawLog.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	// Convert to types.Log for the generated unpacker
	typesLog, err := ep.convertToTypesLog(rawLog)
	if err != nil {
		return nil, fmt.Errorf("failed to convert log: %w", err)
	}

	eventSig := rawLog.Topics[0]

	// Try each known event type
	eventParsers := []struct {
		name   string
		parser func(*types.Log) (interface{}, error)
	}{
		{"ContractDeployed", func(log *types.Log) (interface{}, error) {
			return ep.trebContract.UnpackContractDeployedEvent(log)
		}},
		{"DeploymentCollision", func(log *types.Log) (interface{}, error) {
			return ep.trebContract.UnpackDeploymentCollisionEvent(log)
		}},
		{"SafeTransactionQueued", func(log *types.Log) (interface{}, error) {
			return ep.trebContract.UnpackSafeTransactionQueuedEvent(log)
		}},
		{"SafeTransactionExecuted", func(log *types.Log) (interface{}, error) {
			return ep.trebContract.UnpackSafeTransactionExecutedEvent(log)
		}},
		{"TransactionSimulated", func(log *types.Log) (interface{}, error) {
			return ep.trebContract.UnpackTransactionSimulatedEvent(log)
		}},
	}

	// Try each parser
	for _, parser := range eventParsers {
		eventID, err := ep.trebContract.GetEventID(parser.name)
		if err != nil {
			continue
		}
		if eventSig == eventID {
			return parser.parser(typesLog)
		}
	}

	// Try proxy events (not in ABI)
	proxyEvent, err := ep.parseProxyEvent(rawLog)
	if err == nil {
		return proxyEvent, nil
	}

	return nil, fmt.Errorf("unknown event signature: %s", eventSig.Hex())
}

// convertToTypesLog converts EventLog to types.Log
func (ep *EventParser) convertToTypesLog(rawLog forge.EventLog) (*types.Log, error) {
	// Decode hex data
	data, err := hex.DecodeString(strings.TrimPrefix(rawLog.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	return &types.Log{
		Address: rawLog.Address,
		Topics:  rawLog.Topics,
		Data:    data,
	}, nil
}

// parseProxyEvent attempts to parse proxy-related events
func (ep *EventParser) parseProxyEvent(rawLog forge.EventLog) (interface{}, error) {
	if len(rawLog.Topics) == 0 {
		return nil, fmt.Errorf("no topics")
	}

	eventSig := rawLog.Topics[0]

	// Known proxy event signatures
	var (
		upgradedTopic       = crypto.Keccak256Hash([]byte("Upgraded(address)"))
		adminChangedTopic   = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))
		beaconUpgradedTopic = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
	)

	switch eventSig {
	case upgradedTopic:
		return ep.parseUpgradedEvent(rawLog)
	case adminChangedTopic:
		return ep.parseAdminChangedEvent(rawLog)
	case beaconUpgradedTopic:
		return ep.parseBeaconUpgradedEvent(rawLog)
	}

	return nil, fmt.Errorf("not a proxy event")
}

// parseUpgradedEvent parses an Upgraded event
func (ep *EventParser) parseUpgradedEvent(log forge.EventLog) (*UpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid Upgraded event: not enough topics")
	}

	return &UpgradedEvent{
		ProxyAddress:          log.Address,
		ImplementationAddress: common.HexToAddress(log.Topics[1].Hex()),
	}, nil
}

// parseAdminChangedEvent parses an AdminChanged event
func (ep *EventParser) parseAdminChangedEvent(log forge.EventLog) (*AdminChangedEvent, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("invalid AdminChanged event: not enough topics")
	}

	return &AdminChangedEvent{
		ProxyAddress:  log.Address,
		PreviousAdmin: common.HexToAddress(log.Topics[1].Hex()),
		NewAdmin:      common.HexToAddress(log.Topics[2].Hex()),
	}, nil
}

// parseBeaconUpgradedEvent parses a BeaconUpgraded event
func (ep *EventParser) parseBeaconUpgradedEvent(log forge.EventLog) (*BeaconUpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid BeaconUpgraded event: not enough topics")
	}

	return &BeaconUpgradedEvent{
		ProxyAddress: log.Address,
		Beacon:       common.HexToAddress(log.Topics[1].Hex()),
	}, nil
}

// ExtractDeploymentEvents filters deployment events from all events
func ExtractDeploymentEvents(allEvents []interface{}) []*bindings.TrebContractDeployed {
	var deploymentEvents []*bindings.TrebContractDeployed

	for _, event := range allEvents {
		if deployEvent, ok := event.(*bindings.TrebContractDeployed); ok {
			deploymentEvents = append(deploymentEvents, deployEvent)
		}
	}

	return deploymentEvents
}

// ExtractCollisionEvents filters deployment collision events from all events
func ExtractCollisionEvents(allEvents []interface{}) []*bindings.TrebDeploymentCollision {
	var collisionEvents []*bindings.TrebDeploymentCollision

	for _, event := range allEvents {
		if collisionEvent, ok := event.(*bindings.TrebDeploymentCollision); ok {
			collisionEvents = append(collisionEvents, collisionEvent)
		}
	}

	return collisionEvents
}

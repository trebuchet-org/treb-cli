package abi

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/bindings"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
)

type Event = domain.ParsedEvent

// ParseEvents parses all events from script output
func (p *EventParser) ParseEvents(output *forge.ScriptOutput) ([]Event, error) {
	if output == nil || output.RawLogs == nil {
		return nil, nil
	}

	var parsedEvents []domain.ParsedEvent

	for _, rawLog := range output.RawLogs {
		if len(rawLog.Topics) == 0 {
			continue
		}

		event, err := p.ParseEvent(&rawLog)
		if err != nil {
			// Skip unknown events silently
			if !strings.Contains(err.Error(), "unknown event signature") {
				// Log warning for actual parsing errors
				fmt.Printf("Warning: failed to parse event %s: %v\n", rawLog.Topics[0].Hex(), err)
			}
			continue
		}

		p.log.Debug("Parsed event", "event", event)
		parsedEvents = append(parsedEvents, event)
	}

	return parsedEvents, nil
}

// ParseEvent parses a single event log
func (p *EventParser) ParseEvent(rawLog *forge.EventLog) (Event, error) {
	if len(rawLog.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	// Convert to types.Log for the generated unpacker
	typesLog, err := p.convertToTypesLog(*rawLog)
	if err != nil {
		return nil, fmt.Errorf("failed to convert log: %w", err)
	}

	must := func(hash common.Hash, err error) common.Hash {
		if err != nil {
			panic(err)
		}
		return hash
	}

	eventSig := rawLog.Topics[0]

	// Try each known event type
	eventEventParsers := []struct {
		eventSig common.Hash
		parser   func(*types.Log) (Event, error)
	}{
		{must(p.trebContract.GetEventID("ContractDeployed")), func(log *types.Log) (Event, error) {
			return p.trebContract.UnpackContractDeployedEvent(log)
		}},
		{must(p.trebContract.GetEventID("DeploymentCollision")), func(log *types.Log) (Event, error) {
			return p.trebContract.UnpackDeploymentCollisionEvent(log)
		}},
		{must(p.trebContract.GetEventID("SafeTransactionQueued")), func(log *types.Log) (Event, error) {
			return p.trebContract.UnpackSafeTransactionQueuedEvent(log)
		}},
		{must(p.trebContract.GetEventID("SafeTransactionExecuted")), func(log *types.Log) (Event, error) {
			return p.trebContract.UnpackSafeTransactionExecutedEvent(log)
		}},
		{must(p.trebContract.GetEventID("TransactionSimulated")), func(log *types.Log) (Event, error) {
			return p.trebContract.UnpackTransactionSimulatedEvent(log)
		}},
	}

	// Try each parser
	for _, parser := range eventEventParsers {
		if eventSig == parser.eventSig {
			return parser.parser(typesLog)
		}
	}

	// Try proxy events (not in ABI)
	proxyEvent, err := p.parseProxyEvent(*rawLog)
	if err == nil {
		return proxyEvent, nil
	}

	return nil, fmt.Errorf("unknown event signature: %s", eventSig.Hex())
}

// convertToTypesLog converts EventLog to types.Log
func (p *EventParser) convertToTypesLog(rawLog forge.EventLog) (*types.Log, error) {
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
func (p *EventParser) parseProxyEvent(rawLog forge.EventLog) (Event, error) {
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
		return p.parseUpgradedEvent(rawLog)
	case adminChangedTopic:
		return p.parseAdminChangedEvent(rawLog)
	case beaconUpgradedTopic:
		return p.parseBeaconUpgradedEvent(rawLog)
	}

	return nil, fmt.Errorf("not a proxy event")
}

// parseUpgradedEvent parses an Upgraded event
func (p *EventParser) parseUpgradedEvent(log forge.EventLog) (*domain.UpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid Upgraded event: not enough topics")
	}

	return &domain.UpgradedEvent{
		ProxyAddress:          log.Address,
		ImplementationAddress: common.HexToAddress(log.Topics[1].Hex()),
	}, nil
}

// parseAdminChangedEvent parses an AdminChanged event
func (p *EventParser) parseAdminChangedEvent(log forge.EventLog) (*domain.AdminChangedEvent, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("invalid AdminChanged event: not enough topics")
	}

	return &domain.AdminChangedEvent{
		ProxyAddress:  log.Address,
		PreviousAdmin: common.HexToAddress(log.Topics[1].Hex()),
		NewAdmin:      common.HexToAddress(log.Topics[2].Hex()),
	}, nil
}

// parseBeaconUpgradedEvent parses a BeaconUpgraded event
func (p *EventParser) parseBeaconUpgradedEvent(log forge.EventLog) (*domain.BeaconUpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid BeaconUpgraded event: not enough topics")
	}

	return &domain.BeaconUpgradedEvent{
		ProxyAddress: log.Address,
		Beacon:       common.HexToAddress(log.Topics[1].Hex()),
	}, nil
}

// ExtractDeploymentEvents filters deployment events from all events
func ExtractDeploymentEvents(allEvents []any) []*bindings.TrebContractDeployed {
	var deploymentEvents []*bindings.TrebContractDeployed

	for _, event := range allEvents {
		if deployEvent, ok := event.(*bindings.TrebContractDeployed); ok {
			deploymentEvents = append(deploymentEvents, deployEvent)
		}
	}

	return deploymentEvents
}

// ExtractCollisionEvents filters deployment collision events from all events
func ExtractCollisionEvents(allEvents []any) []*bindings.TrebDeploymentCollision {
	var collisionEvents []*bindings.TrebDeploymentCollision

	for _, event := range allEvents {
		if collisionEvent, ok := event.(*bindings.TrebDeploymentCollision); ok {
			collisionEvents = append(collisionEvents, collisionEvent)
		}
	}

	return collisionEvents
}

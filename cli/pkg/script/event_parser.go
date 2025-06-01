package script

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/treb"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// EventParser uses generated ABI bindings to parse events
type EventParser struct {
	trebContract *treb.Treb
}

// NewEventParser creates a new event parser using generated ABI bindings
func NewEventParser() *EventParser {
	return &EventParser{
		trebContract: treb.NewTreb(),
	}
}

// ParseEvent parses a single event using the generated ABI bindings
// Returns interface{} to allow both generated types and proxy events
func (p *EventParser) ParseEvent(rawLog RawLog) (interface{}, error) {
	if len(rawLog.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	// Convert to types.Log format for the generated unpacker
	typesLog, err := ConvertRawLogToTypesLog(rawLog)
	if err != nil {
		return nil, fmt.Errorf("failed to convert log: %w", err)
	}

	eventSig := rawLog.Topics[0]

	// Check against known event signatures using the generated bindings
	if deployingContractID, err := p.trebContract.GetEventID("DeployingContract"); err == nil && eventSig == deployingContractID {
		event, err := p.trebContract.UnpackDeployingContractEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack DeployingContract event: %w", err)
		}
		return event, nil
	}

	if contractDeployedID, err := p.trebContract.GetEventID("ContractDeployed"); err == nil && eventSig == contractDeployedID {
		event, err := p.trebContract.UnpackContractDeployedEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack ContractDeployed event: %w", err)
		}
		return event, nil
	}

	if safeTransactionQueuedID, err := p.trebContract.GetEventID("SafeTransactionQueued"); err == nil && eventSig == safeTransactionQueuedID {
		event, err := p.trebContract.UnpackSafeTransactionQueuedEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack SafeTransactionQueued event: %w", err)
		}
		return event, nil
	}

	if transactionSimulatedID, err := p.trebContract.GetEventID("TransactionSimulated"); err == nil && eventSig == transactionSimulatedID {
		event, err := p.trebContract.UnpackTransactionSimulatedEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack TransactionSimulated event: %w", err)
		}
		return event, nil
	}

	if transactionFailedID, err := p.trebContract.GetEventID("TransactionFailed"); err == nil && eventSig == transactionFailedID {
		event, err := p.trebContract.UnpackTransactionFailedEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack TransactionFailed event: %w", err)
		}
		return event, nil
	}

	if transactionBroadcastID, err := p.trebContract.GetEventID("TransactionBroadcast"); err == nil && eventSig == transactionBroadcastID {
		event, err := p.trebContract.UnpackTransactionBroadcastEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack TransactionBroadcast event: %w", err)
		}
		return event, nil
	}

	if broadcastStartedID, err := p.trebContract.GetEventID("BroadcastStarted"); err == nil && eventSig == broadcastStartedID {
		event, err := p.trebContract.UnpackBroadcastStartedEvent(typesLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack BroadcastStarted event: %w", err)
		}
		return event, nil
	}

	// Fall back to manual parsing for proxy events (not in our ABI)
	eventsLog := events.Log{
		Address: rawLog.Address,
		Topics:  rawLog.Topics,
		Data:    rawLog.Data,
	}

	switch eventSig {
	case UpgradedTopic:
		return parseUpgradedEvent(eventsLog)
	case AdminChangedTopic:
		return parseAdminChangedEvent(eventsLog)
	case BeaconUpgradedTopic:
		return parseBeaconUpgradedEvent(eventsLog)
	}

	return nil, fmt.Errorf("unknown event signature: %s", eventSig.Hex())
}

// NOTE: Conversion methods removed - we now return generated types directly
// Consumers should use type switches to handle the different generated event types

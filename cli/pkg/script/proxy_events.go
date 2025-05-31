package script

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// Event signatures for proxy events (keccak256 of event signature)
var (
	// ERC1967 Proxy Events
	// Upgraded(address indexed implementation)
	UpgradedTopic = crypto.Keccak256Hash([]byte("Upgraded(address)"))

	// AdminChanged(address previousAdmin, address newAdmin)
	AdminChangedTopic = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))

	// BeaconUpgraded(address indexed beacon)
	BeaconUpgradedTopic = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
)

// parseUpgradedEvent parses an Upgraded event from a log
func parseUpgradedEvent(log events.Log) (*events.UpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid number of topics for Upgraded event")
	}

	// Topics: [eventSig, implementation (indexed)]
	implementation := common.HexToAddress(log.Topics[1].Hex())

	// The proxy address is the address that emitted the event
	proxyAddress := log.Address

	return &events.UpgradedEvent{
		ProxyAddress:          proxyAddress,
		ImplementationAddress: implementation,
		TransactionID:         common.Hash{}, // Will be filled in by context
	}, nil
}

// parseAdminChangedEvent parses an AdminChanged event from a log
func parseAdminChangedEvent(log events.Log) (*events.AdminChangedEvent, error) {
	if len(log.Topics) < 1 {
		return nil, fmt.Errorf("invalid number of topics for AdminChanged event")
	}

	// Decode non-indexed parameters from data
	// Parameters: address previousAdmin, address newAdmin
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	// Create ABI for decoding
	addressType, _ := abi.NewType("address", "", nil)
	args := abi.Arguments{
		{Type: addressType, Name: "previousAdmin"},
		{Type: addressType, Name: "newAdmin"},
	}

	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	if len(values) != 2 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}

	previousAdmin, ok := values[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("failed to cast previousAdmin")
	}

	newAdmin, ok := values[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf("failed to cast newAdmin")
	}

	return &events.AdminChangedEvent{
		ProxyAddress:  log.Address,
		PreviousAdmin: previousAdmin,
		NewAdmin:      newAdmin,
		TransactionID: common.Hash{}, // Will be filled in by context
	}, nil
}

// parseBeaconUpgradedEvent parses a BeaconUpgraded event from a log
func parseBeaconUpgradedEvent(log events.Log) (*events.BeaconUpgradedEvent, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("invalid number of topics for BeaconUpgraded event")
	}

	// Topics: [eventSig, beacon (indexed)]
	beacon := common.HexToAddress(log.Topics[1].Hex())

	return &events.BeaconUpgradedEvent{
		ProxyAddress:  log.Address,
		Beacon:        beacon,
		TransactionID: common.Hash{}, // Will be filled in by context
	}, nil
}
package domain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type EventType string

const (
	// Proxy events only - Treb events now come from generated bindings
	EventTypeAdminChanged   EventType = "AdminChanged"
	EventTypeBeaconUpgraded EventType = "BeaconUpgraded"
	EventTypeUpgraded       EventType = "Upgraded"
	EventTypeUnknown        EventType = "Unknown"
)

// ParsedEvent is the interface for all parsed events
type ParsedEvent interface {
	ContractEventName() string
	String() string
}

// AdminChangedEvent represents a proxy admin change
type AdminChangedEvent struct {
	ProxyAddress  common.Address
	PreviousAdmin common.Address
	NewAdmin      common.Address
	TransactionID common.Hash
}

func (AdminChangedEvent) ContractEventName() string {
	return string(EventTypeAdminChanged)
}

func (e *AdminChangedEvent) String() string {
	return fmt.Sprintf("%s: proxy=%s, old=%s, new=%s",
		e.ContractEventName(),
		e.ProxyAddress.Hex()[:10]+"...",
		e.PreviousAdmin.Hex()[:10]+"...",
		e.NewAdmin.Hex()[:10]+"...",
	)
}

// BeaconUpgradedEvent represents a beacon upgrade
type BeaconUpgradedEvent struct {
	ProxyAddress  common.Address
	Beacon        common.Address
	TransactionID common.Hash
}

func (BeaconUpgradedEvent) ContractEventName() string {
	return string(EventTypeBeaconUpgraded)
}

func (e *BeaconUpgradedEvent) String() string {
	return fmt.Sprintf("%s: proxy=%s, beacon=%s",
		e.ContractEventName(),
		e.ProxyAddress.Hex()[:10]+"...",
		e.Beacon.Hex()[:10]+"...",
	)
}

// UpgradedEvent represents a proxy implementation upgrade
type UpgradedEvent struct {
	ProxyAddress          common.Address
	ImplementationAddress common.Address
	TransactionID         common.Hash
}

func (UpgradedEvent) ContractEventName() string {
	return string(EventTypeUpgraded)
}

func (e *UpgradedEvent) String() string {
	return fmt.Sprintf("%s: proxy=%s, impl=%s",
		e.ContractEventName(),
		e.ProxyAddress.Hex()[:10]+"...",
		e.ImplementationAddress.Hex()[:10]+"...",
	)
}

// UnknownEvent represents an unknown event type
type UnknownEvent struct {
	Address       common.Address
	Topics        []common.Hash
	Data          string
	TransactionID common.Hash
}

func (UnknownEvent) ContractEventName() string {
	return string(EventTypeUnknown)
}

func (e *UnknownEvent) String() string {
	return fmt.Sprintf("%s: %s", e.ContractEventName(), e.Address.Hex()[:10]+"...")
}

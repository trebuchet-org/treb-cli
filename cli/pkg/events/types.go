package events

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// EventType represents the type of event
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
	Type() EventType
	String() string
}

// Log represents a raw event log for parsing
type Log struct {
	Address common.Address   `json:"address"`
	Topics  []common.Hash    `json:"topics"`
	Data    string           `json:"data"`
}

// AdminChangedEvent represents a proxy admin change
type AdminChangedEvent struct {
	ProxyAddress   common.Address
	PreviousAdmin  common.Address
	NewAdmin       common.Address
	TransactionID  common.Hash
}

func (e *AdminChangedEvent) Type() EventType {
	return EventTypeAdminChanged
}

func (e *AdminChangedEvent) String() string {
	return fmt.Sprintf("Admin changed: proxy=%s, old=%s, new=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.PreviousAdmin.Hex()[:10]+"...", e.NewAdmin.Hex()[:10]+"...")
}

// BeaconUpgradedEvent represents a beacon upgrade
type BeaconUpgradedEvent struct {
	ProxyAddress  common.Address
	Beacon        common.Address
	TransactionID common.Hash
}

func (e *BeaconUpgradedEvent) Type() EventType {
	return EventTypeBeaconUpgraded
}

func (e *BeaconUpgradedEvent) String() string {
	return fmt.Sprintf("Beacon upgraded: proxy=%s, beacon=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.Beacon.Hex()[:10]+"...")
}

// UpgradedEvent represents a proxy implementation upgrade
type UpgradedEvent struct {
	ProxyAddress       common.Address
	ImplementationAddress common.Address
	TransactionID      common.Hash
}

func (e *UpgradedEvent) Type() EventType {
	return EventTypeUpgraded
}

func (e *UpgradedEvent) String() string {
	return fmt.Sprintf("Implementation upgraded: proxy=%s, impl=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.ImplementationAddress.Hex()[:10]+"...")
}

// UnknownEvent represents an unknown event type
type UnknownEvent struct {
	Address       common.Address
	Topics        []common.Hash
	Data          string
	TransactionID common.Hash
}

func (e *UnknownEvent) Type() EventType {
	return EventTypeUnknown
}

func (e *UnknownEvent) String() string {
	return fmt.Sprintf("Unknown event: %s", e.Address.Hex()[:10]+"...")
}

// ProxyRelationshipType represents the type of proxy relationship
type ProxyRelationshipType string

const (
	ProxyTypeMinimal    ProxyRelationshipType = "MINIMAL"
	ProxyTypeUUPS       ProxyRelationshipType = "UUPS"
	ProxyTypeTransparent ProxyRelationshipType = "TRANSPARENT"
	ProxyTypeBeacon     ProxyRelationshipType = "BEACON"
)

// ProxyRelationship represents a proxy-implementation relationship
type ProxyRelationship struct {
	ProxyAddress         common.Address
	ImplementationAddress common.Address
	AdminAddress         *common.Address
	BeaconAddress        *common.Address
	ProxyType            ProxyRelationshipType
}

// ProxyDeployedEvent represents a proxy deployment event
type ProxyDeployedEvent struct {
	ProxyAddress          common.Address
	ImplementationAddress common.Address
	AdminAddress          *common.Address
	BeaconAddress         *common.Address
	TransactionID         common.Hash
	ProxyType             ProxyRelationshipType
}

func (e *ProxyDeployedEvent) Type() EventType {
	return "ProxyDeployed"
}

func (e *ProxyDeployedEvent) String() string {
	return fmt.Sprintf("Proxy deployed: proxy=%s, impl=%s",
		e.ProxyAddress.Hex()[:10]+"...", e.ImplementationAddress.Hex()[:10]+"...")
}
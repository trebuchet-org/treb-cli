package events

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
)

// ProxyTracker tracks proxy relationships from events
type ProxyTracker struct {
	relationships map[common.Address]*ProxyRelationship
}

// NewProxyTracker creates a new proxy tracker
func NewProxyTracker() *ProxyTracker {
	return &ProxyTracker{
		relationships: make(map[common.Address]*ProxyRelationship),
	}
}

// ProcessEvents processes events to identify proxy relationships
func (pt *ProxyTracker) ProcessEvents(events []ParsedEvent) {
	// First pass: Create relationships from proxy deployment events
	for _, event := range events {
		switch e := event.(type) {
		case *ProxyDeployedEvent:
			pt.relationships[e.ProxyAddress] = &ProxyRelationship{
				ProxyAddress:          e.ProxyAddress,
				ImplementationAddress: e.ImplementationAddress,
				ProxyType:             ProxyTypeMinimal,
			}
		case *UpgradedEvent:
			if rel, exists := pt.relationships[e.ProxyAddress]; exists {
				rel.ImplementationAddress = e.ImplementationAddress
			} else {
				pt.relationships[e.ProxyAddress] = &ProxyRelationship{
					ProxyAddress:          e.ProxyAddress,
					ImplementationAddress: e.ImplementationAddress,
					ProxyType:             ProxyTypeUUPS,
				}
			}
		}
	}

	// Second pass: Enhance with admin and beacon information
	for _, event := range events {
		switch e := event.(type) {
		case *AdminChangedEvent:
			if rel, exists := pt.relationships[e.ProxyAddress]; exists {
				rel.AdminAddress = &e.NewAdmin
				if rel.ProxyType == ProxyTypeMinimal {
					rel.ProxyType = ProxyTypeTransparent
				}
			}
		case *BeaconUpgradedEvent:
			if rel, exists := pt.relationships[e.ProxyAddress]; exists {
				rel.BeaconAddress = &e.Beacon
				rel.ProxyType = ProxyTypeBeacon
			}
		}
	}

	// Third pass: Check for standalone proxy events that might create new relationships
	for _, event := range events {
		switch oe := event.(type) {
		case *UpgradedEvent:
			// If we see an Upgraded event but don't have the proxy relationship yet, create one
			if _, exists := pt.relationships[oe.ProxyAddress]; !exists {
				pt.relationships[oe.ProxyAddress] = &ProxyRelationship{
					ProxyAddress:          oe.ProxyAddress,
					ImplementationAddress: oe.ImplementationAddress,
					ProxyType:             ProxyTypeUUPS,
				}
			}
		case *AdminChangedEvent:
			// If we see an AdminChanged event but don't have the proxy relationship yet, create one
			if _, exists := pt.relationships[oe.ProxyAddress]; !exists {
				pt.relationships[oe.ProxyAddress] = &ProxyRelationship{
					ProxyAddress:          oe.ProxyAddress,
					ImplementationAddress: common.Address{}, // Will be set by Upgraded event if present
					AdminAddress:          &oe.NewAdmin,
					ProxyType:             ProxyTypeTransparent,
				}
			}
		}
	}
}

// GetRelationshipForProxy returns the proxy relationship for a given address
func (pt *ProxyTracker) GetRelationshipForProxy(proxyAddress common.Address) (*ProxyRelationship, bool) {
	rel, exists := pt.relationships[proxyAddress]
	return rel, exists
}

// GetProxiesForImplementation returns all proxies pointing to an implementation
func (pt *ProxyTracker) GetProxiesForImplementation(implAddress common.Address) []*ProxyRelationship {
	var proxies []*ProxyRelationship
	for _, rel := range pt.relationships {
		if rel.ImplementationAddress == implAddress {
			proxies = append(proxies, rel)
		}
	}
	return proxies
}

// PrintProxyRelationships prints all detected proxy relationships
func (pt *ProxyTracker) PrintProxyRelationships() {
	if len(pt.relationships) == 0 {
		return
	}

	fmt.Println("\n🔗 Proxy Relationships:")
	for _, rel := range pt.relationships {
		proxyType := color.New(color.FgCyan).Sprint(rel.ProxyType)
		fmt.Printf("  %s %s (%s)\n", 
			getProxyIcon(rel.ProxyType),
			rel.ProxyAddress.Hex()[:10],
			proxyType,
		)
		fmt.Printf("    → Implementation: %s\n", rel.ImplementationAddress.Hex())
		
		if rel.AdminAddress != nil && *rel.AdminAddress != (common.Address{}) {
			fmt.Printf("    → Admin: %s\n", rel.AdminAddress.Hex())
		}
		
		if rel.BeaconAddress != nil && *rel.BeaconAddress != (common.Address{}) {
			fmt.Printf("    → Beacon: %s\n", rel.BeaconAddress.Hex())
		}
	}
}

// getProxyIcon returns an icon for the proxy type
func getProxyIcon(proxyType ProxyRelationshipType) string {
	switch proxyType {
	case ProxyTypeTransparent:
		return "🔍"
	case ProxyTypeUUPS:
		return "⬆️"
	case ProxyTypeBeacon:
		return "🏮"
	default:
		return "🔄"
	}
}

// GetAllRelationships returns all proxy relationships
func (pt *ProxyTracker) GetAllRelationships() map[common.Address]*ProxyRelationship {
	return pt.relationships
}
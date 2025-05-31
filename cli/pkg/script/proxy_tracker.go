package script

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/treb"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ProxyRelationship re-exports the type from events package for backward compatibility
type ProxyRelationship = events.ProxyRelationship

// ProxyTracker tracks proxy relationships from events
type ProxyTracker struct {
	relationships map[common.Address]*events.ProxyRelationship
	deployments   map[common.Address]*treb.TrebContractDeployed
}

// NewProxyTracker creates a new proxy tracker
func NewProxyTracker() *ProxyTracker {
	return &ProxyTracker{
		relationships: make(map[common.Address]*events.ProxyRelationship),
		deployments:   make(map[common.Address]*treb.TrebContractDeployed),
	}
}

// ProcessEvents processes all events to build proxy relationships
func (pt *ProxyTracker) ProcessEvents(parsedEvents []interface{}) {
	// First pass: collect all deployments
	for _, event := range parsedEvents {
		switch e := event.(type) {
		case *treb.TrebContractDeployed:
			pt.deployments[e.Location] = e
		}
	}

	// Second pass: process proxy events
	for _, event := range parsedEvents {
		switch e := event.(type) {
		case *events.UpgradedEvent:
			pt.processUpgradedEvent(e)
		case *events.AdminChangedEvent:
			pt.processAdminChangedEvent(e)
		case *events.BeaconUpgradedEvent:
			pt.processBeaconUpgradedEvent(e)
		}
	}
}

// processUpgradedEvent handles proxy implementation upgrades
func (pt *ProxyTracker) processUpgradedEvent(event *events.UpgradedEvent) {
	rel, exists := pt.relationships[event.ProxyAddress]
	if !exists {
		rel = &events.ProxyRelationship{
			ProxyAddress:  event.ProxyAddress,
			ProxyType:     events.ProxyTypeUUPS,
		}
		pt.relationships[event.ProxyAddress] = rel
	}
	
	// Update implementation
	rel.ImplementationAddress = event.ImplementationAddress
}

// processAdminChangedEvent handles admin changes
func (pt *ProxyTracker) processAdminChangedEvent(event *events.AdminChangedEvent) {
	rel, exists := pt.relationships[event.ProxyAddress]
	if !exists {
		rel = &events.ProxyRelationship{
			ProxyAddress:  event.ProxyAddress,
			ProxyType:     events.ProxyTypeTransparent,
		}
		pt.relationships[event.ProxyAddress] = rel
	}
	
	// Update admin
	rel.AdminAddress = &event.NewAdmin
	
	// Transparent proxies have admin changes
	if rel.ProxyType == events.ProxyTypeUUPS {
		rel.ProxyType = events.ProxyTypeTransparent
	}
}

// processBeaconUpgradedEvent handles beacon upgrades
func (pt *ProxyTracker) processBeaconUpgradedEvent(event *events.BeaconUpgradedEvent) {
	rel, exists := pt.relationships[event.ProxyAddress]
	if !exists {
		rel = &events.ProxyRelationship{
			ProxyAddress:  event.ProxyAddress,
			ProxyType:     events.ProxyTypeBeacon,
		}
		pt.relationships[event.ProxyAddress] = rel
	}
	
	// Update beacon
	rel.BeaconAddress = &event.Beacon
	rel.ProxyType = events.ProxyTypeBeacon
}

// GetProxyRelationships returns all detected proxy relationships
func (pt *ProxyTracker) GetProxyRelationships() map[common.Address]*events.ProxyRelationship {
	return pt.relationships
}

// GetRelationshipForProxy returns the relationship for a specific proxy address
func (pt *ProxyTracker) GetRelationshipForProxy(proxyAddr common.Address) (*events.ProxyRelationship, bool) {
	rel, exists := pt.relationships[proxyAddr]
	return rel, exists
}

// UpdateDeploymentTypes updates deployment entries with proxy relationship info
func (pt *ProxyTracker) UpdateDeploymentTypes(deployments []*types.DeploymentEntry) {
	for _, deployment := range deployments {
		if rel, exists := pt.relationships[deployment.Address]; exists {
			// Update deployment type based on proxy type
			deployment.Type = types.ProxyDeployment
			
			// Add proxy metadata
			if deployment.Metadata.Extra == nil {
				deployment.Metadata.Extra = make(map[string]interface{})
			}
			
			deployment.Metadata.Extra["proxyType"] = string(rel.ProxyType)
			deployment.Metadata.Extra["implementation"] = rel.ImplementationAddress.Hex()
			
			if rel.AdminAddress != nil {
				deployment.Metadata.Extra["admin"] = rel.AdminAddress.Hex()
			}
			
			if rel.BeaconAddress != nil {
				deployment.Metadata.Extra["beacon"] = rel.BeaconAddress.Hex()
			}
		}
	}
}

// PrintProxyRelationships prints a summary of detected proxy relationships
func (pt *ProxyTracker) PrintProxyRelationships() {
	if len(pt.relationships) == 0 {
		return
	}
	
	fmt.Printf("\n🔗 %sProxy Relationships Detected:%s\n", ColorBold, ColorReset)
	for proxyAddr, rel := range pt.relationships {
		fmt.Printf("\n  Proxy: %s%s%s\n", ColorBlue, proxyAddr.Hex(), ColorReset)
		fmt.Printf("    Type: %s\n", rel.ProxyType)
		fmt.Printf("    Implementation: %s%s%s\n", ColorGreen, rel.ImplementationAddress.Hex(), ColorReset)
		
		if rel.AdminAddress != nil {
			fmt.Printf("    Admin: %s%s%s\n", ColorGray, rel.AdminAddress.Hex(), ColorReset)
		}
		
		if rel.BeaconAddress != nil {
			fmt.Printf("    Beacon: %s%s%s\n", ColorPurple, rel.BeaconAddress.Hex(), ColorReset)
		}
		
		// Try to identify the proxy contract
		if deployment, exists := pt.deployments[proxyAddr]; exists {
			fmt.Printf("    Deployed as: %s\n", deployment.Deployment.Artifact)
		}
		
		// Try to identify the implementation contract
		if deployment, exists := pt.deployments[rel.ImplementationAddress]; exists {
			fmt.Printf("    Implementation is: %s\n", deployment.Deployment.Artifact)
		}
	}
}
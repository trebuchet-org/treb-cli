package script

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/treb"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// ExecutionPhase represents the current phase of script execution
type ExecutionPhase int

const (
	PhaseSimulation ExecutionPhase = iota
	PhaseBroadcast
)

func (p ExecutionPhase) String() string {
	switch p {
	case PhaseSimulation:
		return "Simulation"
	case PhaseBroadcast:
		return "Broadcast"
	default:
		return "Unknown"
	}
}

// EnhancedEventDisplay provides improved event display with phase tracking and transaction decoding
type EnhancedEventDisplay struct {
	currentPhase      ExecutionPhase
	transactionDecoder *abi.TransactionDecoder
	indexer           *contracts.Indexer
	deployedContracts map[common.Address]string // Track contracts deployed in this execution
	verbose           bool                      // Show extra detailed information
	correlatedTxs     map[[32]byte]bool         // Track transaction IDs that are correlated with higher-level events
	knownAddresses    map[common.Address]string // Track known addresses (deployers, safes, etc.)
}

// NewEnhancedEventDisplay creates a new enhanced event display
func NewEnhancedEventDisplay(indexer *contracts.Indexer) *EnhancedEventDisplay {
	display := &EnhancedEventDisplay{
		currentPhase:      PhaseSimulation,
		transactionDecoder: abi.NewTransactionDecoder(),
		indexer:           indexer,
		deployedContracts: make(map[common.Address]string),
		verbose:           false,
		correlatedTxs:     make(map[[32]byte]bool),
		knownAddresses:    make(map[common.Address]string),
	}
	
	// Initialize with well-known addresses
	display.initializeWellKnownAddresses()
	
	return display
}

// SetSenderConfigs registers sender addresses from the sender configurations
func (d *EnhancedEventDisplay) SetSenderConfigs(senderConfigs *SenderConfigs) {
	if senderConfigs == nil {
		return
	}
	
	for _, config := range senderConfigs.Configs {
		// Register the address with a friendly name
		if config.Account != (common.Address{}) {
			d.knownAddresses[config.Account] = config.Name
		}
	}
}

// SetVerbose enables or disables verbose output
func (d *EnhancedEventDisplay) SetVerbose(verbose bool) {
	d.verbose = verbose
}

// initializeWellKnownAddresses populates the known addresses map with common addresses
func (d *EnhancedEventDisplay) initializeWellKnownAddresses() {
	// CreateX factory
	d.knownAddresses[common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")] = "CreateX"
	
	// Common Safe addresses
	d.knownAddresses[common.HexToAddress("0x40A2aCCbd92BCA938b02010E17A5b8929b49130D")] = "MultiSend"
	d.knownAddresses[common.HexToAddress("0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67")] = "SafeProxyFactory"
}

// reconcileAddress returns a friendly name for an address if known, otherwise returns a shortened address
func (d *EnhancedEventDisplay) reconcileAddress(addr common.Address) string {
	// Check if it's a known address
	if name, exists := d.knownAddresses[addr]; exists {
		return name
	}
	
	// Check if it's a deployed contract
	if artifact, exists := d.deployedContracts[addr]; exists {
		return artifact
	}
	
	// Return shortened address
	return addr.Hex()[:10] + "..."
}

// ProcessEvents processes all events and displays them with enhanced formatting
func (d *EnhancedEventDisplay) ProcessEvents(allEvents []interface{}) {
	if len(allEvents) == 0 {
		PrintWarningMessage("No events detected")
		return
	}

	// First pass: register deployed contracts for ABI resolution
	d.registerDeployedContracts(allEvents)
	
	// Second pass: register proxy relationships
	d.registerProxyRelationships(allEvents)

	// Third pass: correlate transactions with higher-level events
	d.correlateTransactions(allEvents)

	// Fourth pass: display events with phase tracking
	d.displayEventsWithPhases(allEvents)

	// Summary
	d.printExecutionSummary()
}

// registerDeployedContracts processes ContractDeployed events to register ABIs for transaction decoding
func (d *EnhancedEventDisplay) registerDeployedContracts(allEvents []interface{}) {
	for _, event := range allEvents {
		if deployment, ok := event.(*treb.TrebContractDeployed); ok {
			artifact := deployment.Deployment.Artifact
			address := deployment.Location
			
			// Store just the contract name for cleaner display
		contractName := d.extractContractName(artifact)
		d.deployedContracts[address] = contractName
			
			// Try to load ABI from indexer
			if d.indexer != nil {
				contractInfo := d.indexer.GetContractByArtifact(artifact)
				
				if contractInfo != nil {
					// Try to load ABI from the artifact path
					var abiJSON string
					if contractInfo.ArtifactPath != "" {
						abiJSON = d.loadABIFromPath(contractInfo.ArtifactPath)
					}
					
					if abiJSON != "" {
						if err := d.transactionDecoder.RegisterContract(address, artifact, abiJSON); err != nil {
							// Log the error for debugging
							if d.verbose {
								fmt.Printf("Warning: Failed to register ABI for %s: %v\n", artifact, err)
							}
						}
					} else if d.verbose {
						// Log that ABI wasn't found
						fmt.Printf("Warning: No ABI found for %s (artifact path: %s)\n", artifact, contractInfo.ArtifactPath)
					}
				} else if d.verbose {
					fmt.Printf("Warning: No contract info found for %s in indexer\n", artifact)
				}
			}
		}
	}
}

// extractContractName extracts just the contract name from an artifact path
// e.g., "src/UpgradeableCounter.sol:UpgradeableCounter" -> "UpgradeableCounter"
func (d *EnhancedEventDisplay) extractContractName(artifact string) string {
	// First check if it has a colon separator (Foundry format)
	if idx := strings.LastIndex(artifact, ":"); idx != -1 {
		return artifact[idx+1:]
	}
	
	// Otherwise, check for path separator and .sol extension
	if idx := strings.LastIndex(artifact, "/"); idx != -1 {
		name := artifact[idx+1:]
		// Remove .sol extension if present
		if strings.HasSuffix(name, ".sol") {
			name = name[:len(name)-4]
		}
		return name
	}
	
	// If no separators, return as-is
	return artifact
}

// loadABIFromPath loads ABI JSON from a specific artifact path
func (d *EnhancedEventDisplay) loadABIFromPath(path string) string {
	if data, err := os.ReadFile(path); err == nil {
		// Parse the Foundry artifact JSON
		var artifact struct {
			ABI json.RawMessage `json:"abi"`
		}
		if err := json.Unmarshal(data, &artifact); err == nil {
			if d.verbose {
				fmt.Printf("Loaded ABI from artifact path: %s\n", path)
			}
			return string(artifact.ABI)
		}
	}
	return ""
}

// registerProxyRelationships processes proxy events to register proxy->implementation mappings
func (d *EnhancedEventDisplay) registerProxyRelationships(allEvents []interface{}) {
	for _, event := range allEvents {
		switch e := event.(type) {
		case *events.UpgradedEvent:
			// Register proxy -> implementation relationship
			d.transactionDecoder.RegisterProxyRelationship(e.ProxyAddress, e.ImplementationAddress)
			
			// Also try to copy implementation's ABI to proxy if we have it
			if implArtifact, exists := d.deployedContracts[e.ImplementationAddress]; exists {
				// Extract just the contract name from the artifact path
				implName := d.extractContractName(implArtifact)
				proxyName := fmt.Sprintf("Proxy[%s]", implName)
				d.deployedContracts[e.ProxyAddress] = proxyName
				
				// Try to load and register ABI for the proxy using implementation's ABI
				if d.indexer != nil {
					contractInfo := d.indexer.GetContractByArtifact(implArtifact)
					if contractInfo != nil && contractInfo.ArtifactPath != "" {
						if abiJSON := d.loadABIFromPath(contractInfo.ArtifactPath); abiJSON != "" {
							d.transactionDecoder.RegisterContract(e.ProxyAddress, proxyName, abiJSON)
						}
					}
				}
			}
			
		case *events.ProxyDeployedEvent:
			// Register proxy -> implementation relationship from deployment
			d.transactionDecoder.RegisterProxyRelationship(e.ProxyAddress, e.ImplementationAddress)
			
			// Also try to copy implementation's ABI to proxy if we have it
			if implArtifact, exists := d.deployedContracts[e.ImplementationAddress]; exists {
				// Extract just the contract name from the artifact path
				implName := d.extractContractName(implArtifact)
				proxyName := fmt.Sprintf("Proxy[%s]", implName)
				d.deployedContracts[e.ProxyAddress] = proxyName
				
				// Try to load and register ABI for the proxy using implementation's ABI
				if d.indexer != nil {
					contractInfo := d.indexer.GetContractByArtifact(implArtifact)
					if contractInfo != nil && contractInfo.ArtifactPath != "" {
						if abiJSON := d.loadABIFromPath(contractInfo.ArtifactPath); abiJSON != "" {
							d.transactionDecoder.RegisterContract(e.ProxyAddress, proxyName, abiJSON)
						}
					}
				}
			}
		}
	}
}

// correlateTransactions identifies transaction events that are correlated with higher-level events
func (d *EnhancedEventDisplay) correlateTransactions(allEvents []interface{}) {
	// First pass: Look for ContractDeployed events and mark their transaction IDs as correlated
	for _, event := range allEvents {
		switch e := event.(type) {
		case *treb.TrebContractDeployed:
			// Mark this transaction ID as correlated - we'll show the ContractDeployed event instead
			d.correlatedTxs[e.TransactionId] = true
		
		case *treb.TrebSafeTransactionQueued:
			// Safe transactions are higher-level events that may correlate with multiple transactions
			// For now, we don't mark these as correlated since the individual transactions
			// might still be interesting to see
		}
	}
	
	// Second pass: Look for proxy-related patterns
	// When we see proxy events (Upgraded, AdminChanged, etc.) immediately following a transaction,
	// we can correlate them
	for i, event := range allEvents {
		// Check if this is a proxy-related event
		var isProxyEvent bool
		var proxyAddress common.Address
		
		switch e := event.(type) {
		case *events.UpgradedEvent:
			isProxyEvent = true
			proxyAddress = e.ProxyAddress
		case *events.AdminChangedEvent:
			isProxyEvent = true
			proxyAddress = e.ProxyAddress
		case *events.BeaconUpgradedEvent:
			isProxyEvent = true
			proxyAddress = e.ProxyAddress
		case *events.ProxyDeployedEvent:
			isProxyEvent = true
			proxyAddress = e.ProxyAddress
		}
		
		if isProxyEvent && i > 0 {
			// Look at the previous event - if it's a transaction to the proxy address,
			// it's likely the transaction that triggered this proxy event
			switch prevEvent := allEvents[i-1].(type) {
			case *treb.TrebTransactionSimulated:
				if prevEvent.To == proxyAddress {
					d.correlatedTxs[prevEvent.TransactionId] = true
				}
			case *treb.TrebTransactionBroadcast:
				if prevEvent.To == proxyAddress {
					d.correlatedTxs[prevEvent.TransactionId] = true
				}
			}
		}
	}
}

// displayEventsWithPhases displays events with phase awareness and enhanced formatting
func (d *EnhancedEventDisplay) displayEventsWithPhases(allEvents []interface{}) {
	// Print initial phase header
	d.printPhaseHeader()

	for _, event := range allEvents {
		// Check for phase transition
		if broadcastStarted, ok := event.(*treb.TrebBroadcastStarted); ok {
			d.currentPhase = PhaseBroadcast
			d.printPhaseTransition(broadcastStarted)
			continue
		}

		// Display event with enhanced formatting
		d.displayEnhancedEvent(event)
	}
}

// printPhaseHeader prints the current phase header
func (d *EnhancedEventDisplay) printPhaseHeader() {
	phaseColor := ColorBlue
	phaseIcon := "🔍"
	
	if d.currentPhase == PhaseBroadcast {
		phaseColor = ColorGreen
		phaseIcon = "📤"
	}
	
	fmt.Printf("\n%s%s %s Phase%s\n", ColorBold, phaseIcon, d.currentPhase.String(), ColorReset)
	fmt.Printf("%s%s%s\n", phaseColor, strings.Repeat("─", 40), ColorReset)
}

// printPhaseTransition prints a phase transition message
func (d *EnhancedEventDisplay) printPhaseTransition(broadcastStarted *treb.TrebBroadcastStarted) {
	fmt.Printf("\n%s🚀 Broadcast Phase Started%s\n", ColorBold+ColorGreen, ColorReset)
	fmt.Printf("%s%s%s\n", ColorGreen, strings.Repeat("─", 40), ColorReset)
}

// displayEnhancedEvent displays a single event with enhanced formatting
func (d *EnhancedEventDisplay) displayEnhancedEvent(event interface{}) {
	switch e := event.(type) {
	case *treb.TrebDeployingContract:
		d.displayDeployingContract(e)
	case *treb.TrebContractDeployed:
		d.displayContractDeployed(e)
	case *treb.TrebTransactionSimulated:
		// Skip if this transaction is correlated with a higher-level event
		if d.correlatedTxs[e.TransactionId] {
			return
		}
		d.displayTransactionSimulated(e)
	case *treb.TrebTransactionBroadcast:
		// Skip if this transaction is correlated with a higher-level event
		if d.correlatedTxs[e.TransactionId] {
			return
		}
		d.displayTransactionBroadcast(e)
	case *treb.TrebTransactionFailed:
		// Always show failed transactions, even if correlated
		d.displayTransactionFailed(e)
	case *treb.TrebSafeTransactionQueued:
		d.displaySafeTransactionQueued(e)
	case *events.UpgradedEvent:
		d.displayUpgradedEvent(e)
	case *events.AdminChangedEvent:
		d.displayAdminChangedEvent(e)
	case *events.BeaconUpgradedEvent:
		d.displayBeaconUpgradedEvent(e)
	case *events.ProxyDeployedEvent:
		d.displayProxyDeployedEvent(e)
	default:
		fmt.Printf("  %T\n", event)
	}
}

// displayDeployingContract displays a DeployingContract event
func (d *EnhancedEventDisplay) displayDeployingContract(event *treb.TrebDeployingContract) {
	fmt.Printf("  %sDeploying%s %s%s%s", 
		ColorYellow, ColorReset, 
		ColorCyan, event.What, ColorReset)
	
	if event.Label != "" {
		fmt.Printf(" (label: %s)", event.Label)
	}
	
	fmt.Printf(" | InitCode: %s%s%s\n", 
		ColorGray, fmt.Sprintf("%x", event.InitCodeHash[:4]), ColorReset)
}

// displayContractDeployed displays a ContractDeployed event
func (d *EnhancedEventDisplay) displayContractDeployed(event *treb.TrebContractDeployed) {
	// Also track the deployer address
	if event.Deployer != (common.Address{}) {
		// If we don't already know this address, mark it as a deployer
		if _, exists := d.knownAddresses[event.Deployer]; !exists {
			d.knownAddresses[event.Deployer] = "deployer"
		}
	}
	
	// Extract just the contract name from the artifact path
	contractName := d.extractContractName(event.Deployment.Artifact)
	
	fmt.Printf("  %sDeployed%s %s%s%s at %s%s%s", 
		ColorGreen, ColorReset,
		ColorCyan, contractName, ColorReset,
		ColorGreen, event.Location.Hex(), ColorReset)
	
	if event.Deployment.Label != "" {
		fmt.Printf(" (label: %s)", event.Deployment.Label)
	}
	
	fmt.Printf("\n     Strategy: %s | Salt: %s%s%s | Deployer: %s\n",
		event.Deployment.CreateStrategy,
		ColorGray, fmt.Sprintf("%x", event.Deployment.Salt[:4]), ColorReset,
		d.reconcileAddress(event.Deployer))
}

// displayTransactionSimulated displays a TransactionSimulated event with decoded transaction data
func (d *EnhancedEventDisplay) displayTransactionSimulated(event *treb.TrebTransactionSimulated) {
	// Decode transaction for human-readable display
	decoded := d.transactionDecoder.DecodeTransaction(event.To, event.Data, event.Value, event.ReturnData)
	
	// Use compact format with address reconciliation
	fmt.Printf("  %sSimulated%s %s\n", 
		ColorBlue, ColorReset, decoded.FormatCompactWithReconciler(event.Sender, d.reconcileAddress))
	
	// Only show label if it's not the internal harness:execute label
	if event.Label != "" && event.Label != "harness:execute" {
		fmt.Printf("     Label: %s\n", event.Label)
	}
	
	// In verbose mode, show additional details
	if d.verbose {
		fmt.Printf("     TxID: %s%x%s\n", ColorGray, event.TransactionId[:], ColorReset)
		if len(event.Data) > 0 {
			dataStr := fmt.Sprintf("%x", event.Data)
			if len(dataStr) > 100 {
				dataStr = dataStr[:100] + "..."
			}
			fmt.Printf("     CallData: %s0x%s%s\n", ColorGray, dataStr, ColorReset)
		}
		if len(event.ReturnData) > 0 {
			returnStr := fmt.Sprintf("%x", event.ReturnData)
			if len(returnStr) > 100 {
				returnStr = returnStr[:100] + "..."
			}
			fmt.Printf("     ReturnData: %s0x%s%s\n", ColorGray, returnStr, ColorReset)
		}
	}
}

// displayTransactionBroadcast displays a TransactionBroadcast event with decoded transaction data
func (d *EnhancedEventDisplay) displayTransactionBroadcast(event *treb.TrebTransactionBroadcast) {
	// Decode transaction for human-readable display
	decoded := d.transactionDecoder.DecodeTransaction(event.To, event.Data, event.Value, event.ReturnData)
	
	// Use compact format with address reconciliation
	fmt.Printf("  %sBroadcast%s %s\n", 
		ColorGreen, ColorReset, decoded.FormatCompactWithReconciler(event.Sender, d.reconcileAddress))
	
	// Only show label if it's not the internal harness:execute label
	if event.Label != "" && event.Label != "harness:execute" {
		fmt.Printf("     Label: %s\n", event.Label)
	}
	
	// In verbose mode, show additional details
	if d.verbose {
		fmt.Printf("     TxID: %s%x%s\n", ColorGray, event.TransactionId[:], ColorReset)
		if len(event.Data) > 0 {
			dataStr := fmt.Sprintf("%x", event.Data)
			if len(dataStr) > 100 {
				dataStr = dataStr[:100] + "..."
			}
			fmt.Printf("     CallData: %s0x%s%s\n", ColorGray, dataStr, ColorReset)
		}
		if len(event.ReturnData) > 0 {
			returnStr := fmt.Sprintf("%x", event.ReturnData)
			if len(returnStr) > 100 {
				returnStr = returnStr[:100] + "..."
			}
			fmt.Printf("     ReturnData: %s0x%s%s\n", ColorGray, returnStr, ColorReset)
		}
	}
}

// displayTransactionFailed displays a TransactionFailed event
func (d *EnhancedEventDisplay) displayTransactionFailed(event *treb.TrebTransactionFailed) {
	// Decode transaction for human-readable display (no return data for failed tx)
	decoded := d.transactionDecoder.DecodeTransaction(event.To, event.Data, event.Value, nil)
	
	// Use compact format with address reconciliation
	fmt.Printf("  %sFailed%s %s\n", 
		ColorRed, ColorReset, decoded.FormatCompactWithReconciler(event.Sender, d.reconcileAddress))
	
	// Only show label if it's not the internal harness:execute label
	if event.Label != "" && event.Label != "harness:execute" {
		fmt.Printf("     Label: %s\n", event.Label)
	}
}

// displaySafeTransactionQueued displays a SafeTransactionQueued event
func (d *EnhancedEventDisplay) displaySafeTransactionQueued(event *treb.TrebSafeTransactionQueued) {
	fmt.Printf("  %sSafe Transaction Queued%s | Safe: %s%s%s | Proposer: %s%s%s\n", 
		ColorPurple, ColorReset,
		ColorBlue, event.Safe.Hex()[:10]+"...", ColorReset,
		ColorGray, event.Proposer.Hex()[:10]+"...", ColorReset)
	
	fmt.Printf("     SafeTxHash: %s%s%s | Transactions: %d\n",
		ColorGray, fmt.Sprintf("%x", event.SafeTxHash[:4]), ColorReset,
		len(event.Transactions))
}

// displayUpgradedEvent displays an Upgraded proxy event
func (d *EnhancedEventDisplay) displayUpgradedEvent(event *events.UpgradedEvent) {
	fmt.Printf("  %sProxy Upgraded%s | Proxy: %s%s%s → Impl: %s%s%s\n", 
		ColorYellow, ColorReset,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorGreen, event.ImplementationAddress.Hex()[:10]+"...", ColorReset)
}

// displayAdminChangedEvent displays an AdminChanged proxy event
func (d *EnhancedEventDisplay) displayAdminChangedEvent(event *events.AdminChangedEvent) {
	fmt.Printf("  %sAdmin Changed%s | Proxy: %s%s%s | New Admin: %s%s%s\n", 
		ColorYellow, ColorReset,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorPurple, event.NewAdmin.Hex()[:10]+"...", ColorReset)
}

// displayBeaconUpgradedEvent displays a BeaconUpgraded proxy event
func (d *EnhancedEventDisplay) displayBeaconUpgradedEvent(event *events.BeaconUpgradedEvent) {
	fmt.Printf("  %sBeacon Upgraded%s | Proxy: %s%s%s | Beacon: %s%s%s\n", 
		ColorYellow, ColorReset,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorPurple, event.Beacon.Hex()[:10]+"...", ColorReset)
}

// displayProxyDeployedEvent displays a ProxyDeployed event
func (d *EnhancedEventDisplay) displayProxyDeployedEvent(event *events.ProxyDeployedEvent) {
	fmt.Printf("  %sProxy Deployed%s | Type: %s | Proxy: %s%s%s → Impl: %s%s%s\n", 
		ColorGreen, ColorReset,
		event.ProxyType,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorCyan, event.ImplementationAddress.Hex()[:10]+"...", ColorReset)
	
	if event.AdminAddress != nil {
		fmt.Printf("     Admin: %s%s%s\n", ColorPurple, event.AdminAddress.Hex()[:10]+"...", ColorReset)
	}
	if event.BeaconAddress != nil {
		fmt.Printf("     Beacon: %s%s%s\n", ColorPurple, event.BeaconAddress.Hex()[:10]+"...", ColorReset)
	}
}

// printExecutionSummary prints a summary of the execution
func (d *EnhancedEventDisplay) printExecutionSummary() {
	if len(d.deployedContracts) > 0 {
		fmt.Printf("\n%s📦 Deployment Summary:%s\n", ColorBold, ColorReset)
		for address, artifact := range d.deployedContracts {
			fmt.Printf("  • %s%s%s at %s%s%s\n", 
				ColorCyan, artifact, ColorReset,
				ColorGreen, address.Hex(), ColorReset)
		}
	}
}
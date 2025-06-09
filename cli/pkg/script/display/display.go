package display

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
)

// Display handles the display of script execution results
type Display struct {
	transactionDecoder  *abi.TransactionDecoder
	transactionDisplay  *TransactionDisplay
	indexer            *contracts.Indexer
	deployedContracts  map[common.Address]string // Track contracts deployed in this execution
	verbose            bool                      // Show extra detailed information
	knownAddresses     map[common.Address]string // Track known addresses (deployers, safes, etc.)
}

// NewDisplay creates a new display handler
func NewDisplay(indexer *contracts.Indexer) *Display {
	display := &Display{
		transactionDecoder: abi.NewTransactionDecoder(),
		indexer:            indexer,
		deployedContracts:  make(map[common.Address]string),
		verbose:            false,
		knownAddresses:     make(map[common.Address]string),
	}

	// Initialize with well-known addresses
	display.initializeWellKnownAddresses()
	
	// Initialize transaction display
	display.transactionDisplay = NewTransactionDisplay(display)

	return display
}

// SetVerbose enables or disables verbose output
func (d *Display) SetVerbose(verbose bool) {
	d.verbose = verbose
}

// SetSenderConfigs registers sender addresses from the sender configurations
func (d *Display) SetSenderConfigs(senderConfigs *config.SenderConfigs) {
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

// SetRegistryResolver configures the transaction decoder to use registry-based ABI resolution
func (d *Display) SetRegistryResolver(registryManager *registry.Manager, chainID uint64) {
	if registryManager != nil && d.indexer != nil {
		// Wrap the indexer to satisfy the interface
		indexerAdapter := &indexerAdapter{indexer: d.indexer}
		resolver := abi.NewRegistryABIResolver(registryManager, indexerAdapter, chainID)
		// Enable debug if verbose mode is on
		if r, ok := resolver.(*abi.RegistryABIResolver); ok && d.verbose {
			r.EnableDebug(true)
		}
		d.transactionDecoder.SetABIResolver(resolver)
		d.transactionDisplay.SetABIResolver(resolver)
	}
}

// DisplayExecution displays the complete script execution
func (d *Display) DisplayExecution(execution *parser.ScriptExecution) {
	if execution == nil {
		PrintWarningMessage("No execution data")
		return
	}

	// Display logs first
	d.DisplayLogs(execution.Logs)

	// Register deployed contracts and proxy relationships
	d.registerDeployments(execution)

	// Display transactions
	d.displayTransactions(execution.Transactions)

	// Display execution summary
	d.printExecutionSummary()
}

// DisplayLogs displays console.log output from the script
func (d *Display) DisplayLogs(logs []string) {
	if len(logs) == 0 {
		return
	}

	fmt.Printf("\n%s📝 Script Logs:%s\n", ColorBold, ColorReset)
	fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 40), ColorReset)

	for _, log := range logs {
		fmt.Printf("  %s\n", log)
	}
}

// registerDeployments registers deployed contracts and proxy relationships
func (d *Display) registerDeployments(execution *parser.ScriptExecution) {
	// Register deployed contracts
	for _, dep := range execution.Deployments {
		contractName := extractContractName(dep.Deployment.Artifact)
		d.deployedContracts[dep.Address] = contractName

		// Track deployer
		if dep.Deployer != (common.Address{}) {
			if _, exists := d.knownAddresses[dep.Deployer]; !exists {
				d.knownAddresses[dep.Deployer] = "deployer"
			}
		}

		// Try to load ABI
		if d.indexer != nil {
			contractInfo := d.indexer.GetContractByArtifact(dep.Deployment.Artifact)
			if contractInfo != nil && contractInfo.ArtifactPath != "" {
				if abiJSON := d.loadABIFromPath(contractInfo.ArtifactPath); abiJSON != "" {
					if err := d.transactionDecoder.RegisterContract(dep.Address, dep.Deployment.Artifact, abiJSON); err != nil {
						if d.verbose {
							fmt.Printf("Warning: Failed to register ABI for %s: %v\n", dep.Deployment.Artifact, err)
						}
					}
				}
			}
		}
	}

	// Register proxy relationships
	for proxy, info := range execution.ProxyRelationships {
		d.transactionDecoder.RegisterProxyRelationship(proxy, info.ImplementationAddress)

		// Update display names
		if implName, exists := d.deployedContracts[info.ImplementationAddress]; exists {
			proxyName := d.deployedContracts[proxy]
			d.deployedContracts[proxy] = fmt.Sprintf("%s[%s]", proxyName, implName)
		}
	}
}

// displayTransactions displays the unified transaction list
func (d *Display) displayTransactions(transactions []*parser.Transaction) {
	if len(transactions) == 0 {
		return
	}

	fmt.Printf("%s🔄 Transactions:%s\n", ColorBold, ColorReset)
	fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 50), ColorReset)

	for _, tx := range transactions {
		// Use enhanced transaction display
		d.transactionDisplay.DisplayTransactionWithEvents(tx)
	}
}

// displayOtherEvents displays events that aren't part of transactions
func (d *Display) displayOtherEvents(allEvents []interface{}) {
	hasOtherEvents := false

	for _, event := range allEvents {
		shouldDisplay := false

		switch e := event.(type) {
		case *events.UpgradedEvent:
			if !hasOtherEvents {
				fmt.Printf("\n%s🔧 Other Events:%s\n", ColorBold, ColorReset)
				fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 40), ColorReset)
				hasOtherEvents = true
			}
			d.displayUpgradedEvent(e)
			shouldDisplay = true
		case *events.AdminChangedEvent:
			if !hasOtherEvents {
				fmt.Printf("\n%s🔧 Other Events:%s\n", ColorBold, ColorReset)
				fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 40), ColorReset)
				hasOtherEvents = true
			}
			d.displayAdminChangedEvent(e)
			shouldDisplay = true
		case *events.BeaconUpgradedEvent:
			if !hasOtherEvents {
				fmt.Printf("\n%s🔧 Other Events:%s\n", ColorBold, ColorReset)
				fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 40), ColorReset)
				hasOtherEvents = true
			}
			d.displayBeaconUpgradedEvent(e)
			shouldDisplay = true
		case *events.ProxyDeployedEvent:
			if !hasOtherEvents {
				fmt.Printf("\n%s🔧 Other Events:%s\n", ColorBold, ColorReset)
				fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 40), ColorReset)
				hasOtherEvents = true
			}
			d.displayProxyDeployedEvent(e)
			shouldDisplay = true
		}

		if shouldDisplay && d.verbose {
			// Show raw event type in verbose mode
			fmt.Printf("     Type: %T\n", event)
		}
	}
}

// Display methods for various event types
func (d *Display) displayUpgradedEvent(event *events.UpgradedEvent) {
	fmt.Printf("  %sProxy Upgraded%s | Proxy: %s%s%s → Impl: %s%s%s\n",
		ColorYellow, ColorReset,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorGreen, event.ImplementationAddress.Hex()[:10]+"...", ColorReset)
}

func (d *Display) displayAdminChangedEvent(event *events.AdminChangedEvent) {
	fmt.Printf("  %sAdmin Changed%s | Proxy: %s%s%s | New Admin: %s%s%s\n",
		ColorYellow, ColorReset,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorPurple, event.NewAdmin.Hex()[:10]+"...", ColorReset)
}

func (d *Display) displayBeaconUpgradedEvent(event *events.BeaconUpgradedEvent) {
	fmt.Printf("  %sBeacon Upgraded%s | Proxy: %s%s%s | Beacon: %s%s%s\n",
		ColorYellow, ColorReset,
		ColorBlue, event.ProxyAddress.Hex()[:10]+"...", ColorReset,
		ColorPurple, event.Beacon.Hex()[:10]+"...", ColorReset)
}

func (d *Display) displayProxyDeployedEvent(event *events.ProxyDeployedEvent) {
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
func (d *Display) printExecutionSummary() {
	if len(d.deployedContracts) > 0 {
		fmt.Printf("\n%s📦 Deployment Summary:%s\n", ColorBold, ColorReset)
		fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 50), ColorReset)
		
		for address, artifact := range d.deployedContracts {
			fmt.Printf("%s%s%s at %s%s%s\n",
				ColorCyan, artifact, ColorReset,
				ColorGreen, address.Hex(), ColorReset)
		}
		
		fmt.Println() // Add newline after deployment summary
	}
}

// initializeWellKnownAddresses populates the known addresses map with common addresses
func (d *Display) initializeWellKnownAddresses() {
	// CreateX factory
	d.knownAddresses[common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")] = "CreateX"

	// Common Safe addresses
	d.knownAddresses[common.HexToAddress("0x40A2aCCbd92BCA938b02010E17A5b8929b49130D")] = "MultiSend"
	d.knownAddresses[common.HexToAddress("0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67")] = "SafeProxyFactory"
}

// reconcileAddress returns a friendly name for an address if known
func (d *Display) reconcileAddress(addr common.Address) string {
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

// loadABIFromPath loads ABI JSON from a specific artifact path
func (d *Display) loadABIFromPath(path string) string {
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

// extractContractName extracts just the contract name from an artifact path
func extractContractName(artifact string) string {
	// First check if it has a colon separator (Foundry format)
	if idx := strings.LastIndex(artifact, ":"); idx != -1 {
		return artifact[idx+1:]
	}

	// Otherwise, check for path separator and .sol extension
	if idx := strings.LastIndex(artifact, "/"); idx != -1 {
		name := artifact[idx+1:]
		// Remove .sol extension if present
		name = strings.TrimSuffix(name, ".sol")
		return name
	}

	// If no separators, return as-is
	return artifact
}

// formatTransactionSummary formats a transaction summary
func (d *Display) formatTransactionSummary(tx *parser.Transaction) string {
	to := d.reconcileAddress(tx.Transaction.To)
	sender := d.reconcileAddress(tx.Sender)

	// Try to decode the transaction
	var methodStr string
	decoded := d.transactionDecoder.DecodeTransaction(tx.Transaction.To, tx.Transaction.Data, tx.Transaction.Value, tx.ReturnData)
	if decoded != nil && decoded.Method != "" {
		methodStr = decoded.Method + "(...)"
		if d.verbose && len(decoded.Inputs) > 0 {
			// Show first few args in verbose mode
			methodStr = decoded.Method + "("
			for i, arg := range decoded.Inputs {
				if i > 2 {
					methodStr += "..."
					break
				}
				if i > 0 {
					methodStr += ", "
				}
				methodStr += fmt.Sprintf("%v", arg.Value)
			}
			methodStr += ")"
		}
	} else {
		// Fallback to function selector
		if len(tx.Transaction.Data) >= 4 {
			methodStr = fmt.Sprintf("0x%x", tx.Transaction.Data[:4])
		} else {
			methodStr = "transfer"
		}
	}

	return fmt.Sprintf("%s → %s::%s", sender, to, methodStr)
}

// formatTransactionDetails formats transaction details
func (d *Display) formatTransactionDetails(tx *parser.Transaction) string {
	var details []string

	if tx.TxHash != nil {
		details = append(details, fmt.Sprintf("     Tx: %s", tx.TxHash.Hex()))
	}

	if tx.SafeTxHash != nil {
		details = append(details, fmt.Sprintf("     Safe Tx: %s", tx.SafeTxHash.Hex()))
	}

	if tx.BlockNumber != nil {
		details = append(details, fmt.Sprintf("     Block: %d", *tx.BlockNumber))
	}

	if tx.GasUsed != nil {
		details = append(details, fmt.Sprintf("     Gas: %d", *tx.GasUsed))
	}

	return strings.Join(details, "\n")
}

// getRegistryTransactionID generates a registry transaction ID
func (d *Display) getRegistryTransactionID(tx *parser.Transaction) string {
	if tx.TxHash != nil {
		return fmt.Sprintf("tx-%s", tx.TxHash.Hex())
	} else if tx.SafeTxHash != nil && tx.SafeBatchIdx != nil {
		return fmt.Sprintf("safe-%s-%d", tx.SafeTxHash.Hex(), *tx.SafeBatchIdx)
	} else {
		return fmt.Sprintf("tx-internal-%x", tx.TransactionId)
	}
}

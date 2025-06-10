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
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
)

// Display handles the display of script execution results
type Display struct {
	transactionDecoder *abi.TransactionDecoder
	transactionDisplay *TransactionDisplay
	indexer            *contracts.Indexer
	deployedContracts  map[common.Address]string // Track contracts deployed in this execution
	verbose            bool                      // Show extra detailed information
	knownAddresses     map[common.Address]string // Track known addresses (deployers, safes, etc.)
	execution          *parser.ScriptExecution
}

// NewDisplay creates a new display handler
func NewDisplay(indexer *contracts.Indexer, execution *parser.ScriptExecution) *Display {
	display := &Display{
		transactionDecoder: abi.NewTransactionDecoder(),
		indexer:            indexer,
		deployedContracts:  make(map[common.Address]string),
		verbose:            false,
		knownAddresses:     make(map[common.Address]string),
		execution:          execution,
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
func (d *Display) DisplayExecution() {
	// Display logs first
	d.DisplayLogs(d.execution.Logs)

	// Register deployed contracts and proxy relationships
	d.registerDeployments(d.execution)

	// Display transactions
	d.displayTransactions(d.execution.Transactions)

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

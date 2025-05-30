package script

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	abiPkg "github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// TransactionInfo groups related transaction events
type TransactionInfo struct {
	TransactionID string
	Simulated     *TransactionSimulatedEvent
	Broadcast     *TransactionBroadcastEvent
	Deployments   []*ContractDeployedEvent
	Failed        *TransactionFailedEvent
	ProxyEvents   []ParsedEvent // Upgraded, AdminChanged, BeaconUpgraded events
}

// GetEventIcon returns an icon for the event type
func GetEventIcon(eventType EventType) string {
	switch eventType {
	case EventTypeDeployingContract:
		return "🔨"
	case EventTypeContractDeployed:
		return "✅"
	case EventTypeSafeTransaction:
		return "🔐"
	case EventTypeTransactionBroadcast:
		return "📤"
	case EventTypeTransactionSimulated:
		return "🔍"
	case EventTypeTransactionFailed:
		return "❌"
	case EventTypeUpgraded:
		return "⬆️"
	case EventTypeAdminChanged:
		return "👤"
	case EventTypeBeaconUpgraded:
		return "🔆"
	default:
		return "📝"
	}
}

// PrintEventDetails prints detailed information about an event for verification
func PrintEventDetails(event ParsedEvent, indexer *contracts.Indexer) {
	fmt.Printf("      Type: %s\n", event.Type())
	
	switch e := event.(type) {
	case *ContractDeployedEvent:
		printContractDeployedDetails(e, indexer)
		
	case *DeployingContractEvent:
		printDeployingContractDetails(e)
		
	case *SafeTransactionQueuedEvent:
		printSafeTransactionDetails(e)
		
	case *TransactionBroadcastEvent:
		printTransactionBroadcastDetails(e)
	
	case *TransactionSimulatedEvent:
		printTransactionSimulatedDetails(e)
	
	case *TransactionFailedEvent:
		printTransactionFailedDetails(e)
		
	case *UpgradedEvent:
		printUpgradedDetails(e)
		
	case *AdminChangedEvent:
		printAdminChangedDetails(e)
		
	case *BeaconUpgradedEvent:
		printBeaconUpgradedDetails(e)
		
	default:
		fmt.Printf("      Unknown event type\n")
	}
	fmt.Println()
}

// printContractDeployedDetails prints details for a ContractDeployedEvent
func printContractDeployedDetails(e *ContractDeployedEvent, indexer *contracts.Indexer) {
	fmt.Printf("      Deployer: %s%s%s\n", ColorGray, e.Deployer.Hex(), ColorReset)
	fmt.Printf("      Location: %s%s%s\n", ColorGreen, e.Location.Hex(), ColorReset)
	fmt.Printf("      TransactionID: %s%s%s\n", ColorGray, e.TransactionID.Hex(), ColorReset)
	fmt.Printf("      Salt: %s%s%s\n", ColorGray, e.Deployment.Salt.Hex(), ColorReset)
	fmt.Printf("      BytecodeHash: %s%s%s\n", ColorGray, e.Deployment.BytecodeHash.Hex(), ColorReset)
	fmt.Printf("      CreateStrategy: %s\n", e.Deployment.CreateStrategy)
	
	// Display constructor args with decoding attempt
	if len(e.Deployment.ConstructorArgs) > 0 {
		fmt.Printf("      ConstructorArgs: 0x%x (%d bytes)\n", e.Deployment.ConstructorArgs, len(e.Deployment.ConstructorArgs))
		
		// Try to decode if we know the contract
		if indexer != nil {
			if contractInfo := indexer.GetContractByBytecodeHash(e.Deployment.BytecodeHash.Hex()); contractInfo != nil {
				parser := abiPkg.NewParser(".")
				decodedArgs, err := parser.DecodeConstructorArgs(contractInfo.Name, e.Deployment.ConstructorArgs)
				if err == nil && decodedArgs != "" {
					fmt.Printf("      Decoded Args: %s\n", decodedArgs)
				}
			}
		}
	} else {
		fmt.Printf("      ConstructorArgs: none\n")
	}
	
	// Try to identify the contract by bytecode hash
	if indexer != nil {
		if contractInfo := indexer.GetContractByBytecodeHash(e.Deployment.BytecodeHash.Hex()); contractInfo != nil {
			fmt.Printf("      Contract: %s%s:%s%s\n", ColorCyan, contractInfo.Path, contractInfo.Name, ColorReset)
			if contractInfo.IsLibrary {
				fmt.Printf("      Type: %slibrary%s\n", ColorPurple, ColorReset)
			} else if strings.Contains(strings.ToLower(contractInfo.Name), "proxy") {
				fmt.Printf("      Type: %sproxy%s\n", ColorBlue, ColorReset)
			} else {
				fmt.Printf("      Type: regular\n")
			}
		} else {
			fmt.Printf("      Contract: %s<no match>%s\n", ColorRed, ColorReset)
		}
	}
}

// printDeployingContractDetails prints details for a DeployingContractEvent
func printDeployingContractDetails(e *DeployingContractEvent) {
	fmt.Printf("      What: %s%s%s\n", ColorCyan, e.What, ColorReset)
	fmt.Printf("      Label: %s\n", e.Label)
	fmt.Printf("      InitCodeHash: %s%s%s\n", ColorGray, e.InitCodeHash.Hex(), ColorReset)
}

// printSafeTransactionDetails prints details for a SafeTransactionQueuedEvent
func printSafeTransactionDetails(e *SafeTransactionQueuedEvent) {
	fmt.Printf("      Safe: %s%s%s\n", ColorBlue, e.Safe.Hex(), ColorReset)
	fmt.Printf("      Proposer: %s%s%s\n", ColorGray, e.Proposer.Hex(), ColorReset)
	fmt.Printf("      SafeTxHash: %s%s%s\n", ColorGray, e.SafeTxHash.Hex(), ColorReset)
	fmt.Printf("      Transactions: %d\n", len(e.Transactions))
	
	// Print details of each transaction
	for i, tx := range e.Transactions {
		fmt.Printf("      Transaction %d:\n", i+1)
		fmt.Printf("        TransactionID: %s%s%s\n", ColorGray, tx.TransactionID.Hex()[:10]+"...", ColorReset)
		fmt.Printf("        Label: %s\n", tx.Transaction.Label)
		fmt.Printf("        To: %s%s%s\n", ColorGray, tx.Transaction.To.Hex(), ColorReset)
		if tx.Transaction.Value != nil && tx.Transaction.Value.Sign() > 0 {
			fmt.Printf("        Value: %s\n", tx.Transaction.Value.String())
		}
		statusText := "PENDING"
		statusColor := ColorYellow
		switch tx.Status {
		case 1:
			statusText = "EXECUTED"
			statusColor = ColorGreen
		case 2:
			statusText = "QUEUED"
			statusColor = ColorBlue
		}
		fmt.Printf("        Status: %s%s%s\n", statusColor, statusText, ColorReset)
	}
}

// printTransactionBroadcastDetails prints details for a TransactionBroadcastEvent
func printTransactionBroadcastDetails(e *TransactionBroadcastEvent) {
	fmt.Printf("      TransactionID: %s%s%s\n", ColorGray, e.TransactionID.Hex(), ColorReset)
	fmt.Printf("      Sender: %s%s%s\n", ColorGray, e.Sender.Hex(), ColorReset)
	fmt.Printf("      To: %s%s%s\n", ColorGray, e.To.Hex(), ColorReset)
	fmt.Printf("      Label: %s\n", e.Label)
	if e.Value != nil && e.Value.Sign() > 0 {
		fmt.Printf("      Value: %s\n", e.Value.String())
	}
}

// printTransactionSimulatedDetails prints details for a TransactionSimulatedEvent
func printTransactionSimulatedDetails(e *TransactionSimulatedEvent) {
	fmt.Printf("      TransactionID: %s%s%s\n", ColorGray, e.TransactionID.Hex(), ColorReset)
	fmt.Printf("      Sender: %s%s%s\n", ColorGray, e.Sender.Hex(), ColorReset)
	fmt.Printf("      To: %s%s%s\n", ColorGray, e.To.Hex(), ColorReset)
	fmt.Printf("      Label: %s\n", e.Label)
	if e.Value != nil && e.Value.Sign() > 0 {
		fmt.Printf("      Value: %s\n", e.Value.String())
	}
}

// printTransactionFailedDetails prints details for a TransactionFailedEvent
func printTransactionFailedDetails(e *TransactionFailedEvent) {
	fmt.Printf("      TransactionID: %s%s%s\n", ColorRed, e.TransactionID.Hex(), ColorReset)
	fmt.Printf("      Sender: %s%s%s\n", ColorGray, e.Sender.Hex(), ColorReset)
	fmt.Printf("      To: %s%s%s\n", ColorGray, e.To.Hex(), ColorReset)
	fmt.Printf("      Label: %s\n", e.Label)
}

// printUpgradedDetails prints details for an UpgradedEvent
func printUpgradedDetails(e *UpgradedEvent) {
	fmt.Printf("      ProxyAddress: %s%s%s\n", ColorBlue, e.ProxyAddress.Hex(), ColorReset)
	fmt.Printf("      Implementation: %s%s%s\n", ColorGreen, e.Implementation.Hex(), ColorReset)
	if e.TransactionID != (common.Hash{}) {
		fmt.Printf("      TransactionID: %s%s%s\n", ColorGray, e.TransactionID.Hex(), ColorReset)
	}
}

// printAdminChangedDetails prints details for an AdminChangedEvent
func printAdminChangedDetails(e *AdminChangedEvent) {
	fmt.Printf("      ProxyAddress: %s%s%s\n", ColorBlue, e.ProxyAddress.Hex(), ColorReset)
	fmt.Printf("      PreviousAdmin: %s%s%s\n", ColorGray, e.PreviousAdmin.Hex(), ColorReset)
	fmt.Printf("      NewAdmin: %s%s%s\n", ColorGreen, e.NewAdmin.Hex(), ColorReset)
	if e.TransactionID != (common.Hash{}) {
		fmt.Printf("      TransactionID: %s%s%s\n", ColorGray, e.TransactionID.Hex(), ColorReset)
	}
}

// printBeaconUpgradedDetails prints details for a BeaconUpgradedEvent
func printBeaconUpgradedDetails(e *BeaconUpgradedEvent) {
	fmt.Printf("      ProxyAddress: %s%s%s\n", ColorBlue, e.ProxyAddress.Hex(), ColorReset)
	fmt.Printf("      Beacon: %s%s%s\n", ColorGreen, e.Beacon.Hex(), ColorReset)
	if e.TransactionID != (common.Hash{}) {
		fmt.Printf("      TransactionID: %s%s%s\n", ColorGray, e.TransactionID.Hex(), ColorReset)
	}
}

// ReportTransactions reports transaction events and groups related operations
func ReportTransactions(events []ParsedEvent, indexer *contracts.Indexer, trebConfig *config.TrebConfig, broadcastPath string, chainID uint64) {
	// Load broadcast transactions if path is provided
	var broadcastTxs []broadcast.TransactionInfo
	if broadcastPath != "" {
		// Extract script name from broadcast path and use script-specific loading
		scriptName := extractScriptNameFromBroadcastPath(broadcastPath)
		if scriptName != "" {
			parser := broadcast.NewParser(".")
			broadcastFile, err := parser.ParseLatestBroadcast(scriptName, chainID)
			if err == nil {
				broadcastTxs = convertBroadcastFileToTransactionInfos(broadcastFile)
			}
		}
	}
	
	reportTransactionsWithTxInfo(events, indexer, trebConfig, broadcastTxs)
}

// reportTransactionsWithTxInfo is the internal implementation that shows transaction info when available
func reportTransactionsWithTxInfo(events []ParsedEvent, indexer *contracts.Indexer, trebConfig *config.TrebConfig, broadcastTxs []broadcast.TransactionInfo) {
	// Group events by transaction ID
	txMap := make(map[string]*TransactionInfo)
	
	for _, event := range events {
		switch e := event.(type) {
		case *TransactionSimulatedEvent:
			txID := e.TransactionID.Hex()
			if tx, exists := txMap[txID]; exists {
				tx.Simulated = e
			} else {
				txMap[txID] = &TransactionInfo{
					TransactionID: txID,
					Simulated: e,
				}
			}
		case *TransactionBroadcastEvent:
			txID := e.TransactionID.Hex()
			if tx, exists := txMap[txID]; exists {
				tx.Broadcast = e
			} else {
				txMap[txID] = &TransactionInfo{
					TransactionID: txID,
					Broadcast: e,
				}
			}
		case *TransactionFailedEvent:
			txID := e.TransactionID.Hex()
			if tx, exists := txMap[txID]; exists {
				tx.Failed = e
			} else {
				txMap[txID] = &TransactionInfo{
					TransactionID: txID,
					Failed: e,
				}
			}
		case *ContractDeployedEvent:
			txID := e.TransactionID.Hex()
			if tx, exists := txMap[txID]; exists {
				tx.Deployments = append(tx.Deployments, e)
			} else {
				txMap[txID] = &TransactionInfo{
					TransactionID: txID,
					Deployments: []*ContractDeployedEvent{e},
				}
			}
		case *SafeTransactionQueuedEvent:
			// Add each RichTransaction to the transaction map
			for _, richTx := range e.Transactions {
				txID := richTx.TransactionID.Hex()
				if tx, exists := txMap[txID]; exists {
					// Mark this as a Safe transaction
					if tx.Broadcast == nil {
						// Create a pseudo-broadcast event to show it's queued
						tx.Broadcast = &TransactionBroadcastEvent{
							TransactionID: richTx.TransactionID,
							Label:         fmt.Sprintf("Safe: %s", richTx.Transaction.Label),
						}
					}
				}
			}
		case *UpgradedEvent:
			// For proxy events, we need to find the associated transaction
			// This is a bit tricky since the events don't have transaction IDs
			// We'll associate them with the most recent transaction for now
			// TODO: Improve this by using transaction context
			if len(txMap) > 0 {
				// Find the last transaction (simplified approach)
				for txID, tx := range txMap {
					if tx.ProxyEvents == nil {
						tx.ProxyEvents = []ParsedEvent{}
					}
					tx.ProxyEvents = append(tx.ProxyEvents, e)
					e.TransactionID = common.HexToHash(txID)
					break
				}
			}
		case *AdminChangedEvent:
			if len(txMap) > 0 {
				for txID, tx := range txMap {
					if tx.ProxyEvents == nil {
						tx.ProxyEvents = []ParsedEvent{}
					}
					tx.ProxyEvents = append(tx.ProxyEvents, e)
					e.TransactionID = common.HexToHash(txID)
					break
				}
			}
		case *BeaconUpgradedEvent:
			if len(txMap) > 0 {
				for txID, tx := range txMap {
					if tx.ProxyEvents == nil {
						tx.ProxyEvents = []ParsedEvent{}
					}
					tx.ProxyEvents = append(tx.ProxyEvents, e)
					e.TransactionID = common.HexToHash(txID)
					break
				}
			}
		}
	}
	
	
	// Report transactions
	if len(txMap) > 0 {
		fmt.Printf("\n📤 %s%d transaction(s) captured:%s\n", ColorBold, len(txMap), ColorReset)
		for txID, tx := range txMap {
			if txID == "0x0000000000000000000000000000000000000000000000000000000000000000" {
				// Skip zero transaction ID
				continue
			}
			
			fmt.Printf("\n  Transaction: %s%s%s\n", ColorGray, txID[:10]+"...", ColorReset)
			
			// Show transaction details based on what events we have
			var sender string
			var label string
			
			if tx.Simulated != nil {
				sender = tx.Simulated.Sender.Hex()
				label = tx.Simulated.Label
			} else if tx.Broadcast != nil {
				sender = tx.Broadcast.Sender.Hex()
				label = tx.Broadcast.Label
			} else if tx.Failed != nil {
				sender = tx.Failed.Sender.Hex()
				label = tx.Failed.Label
			}
			
			// Try to resolve sender name from config
			senderName := ""
			if trebConfig != nil && sender != "" {
				name, err := trebConfig.GetSenderNameByAddress(sender)
				if err == nil {
					senderName = name
				}
			}
			
			if senderName != "" {
				fmt.Printf("    Sender: %s%s%s (%s%s%s)\n", ColorCyan, senderName, ColorReset, ColorGray, sender, ColorReset)
			} else if sender != "" {
				fmt.Printf("    Sender: %s%s%s\n", ColorGray, sender, ColorReset)
			}
			
			if label != "" {
				fmt.Printf("    Label: %s\n", label)
			}
			
			// Show status
			statusColor := ColorYellow
			statusText := "PENDING"
			
			if tx.Failed != nil {
				statusColor = ColorRed
				statusText = "FAILED"
			} else if tx.Broadcast != nil {
				statusColor = ColorGreen
				statusText = "EXECUTED"
			} else if tx.Simulated != nil {
				statusColor = ColorBlue
				statusText = "SIMULATED"
			}
			
			fmt.Printf("    Status: %s%s%s", statusColor, statusText, ColorReset)
			
			// Try to find matching broadcast transaction
			if len(broadcastTxs) > 0 && sender != "" {
				for _, btx := range broadcastTxs {
					if strings.EqualFold(btx.From, sender) {
						fmt.Printf(" (tx: %s%s%s", ColorGray, btx.Hash[:10]+"...", ColorReset)
						if btx.BlockNumber > 0 {
							fmt.Printf(" @ block %d", btx.BlockNumber)
						}
						fmt.Printf(")")
						break
					}
				}
			}
			fmt.Println()
			
			if len(tx.Deployments) > 0 {
				fmt.Printf("    Deployments:\n")
				for i, deployment := range tx.Deployments {
					// Decode the deployment transaction
					txDesc := decodeDeploymentTransaction(deployment, indexer)
					fmt.Printf("      %d. %s\n", i+1, txDesc)
				}
			}
			
			if len(tx.ProxyEvents) > 0 {
				fmt.Printf("    Proxy Operations:\n")
				for i, proxyEvent := range tx.ProxyEvents {
					fmt.Printf("      %d. %s %s\n", i+1, GetEventIcon(proxyEvent.Type()), proxyEvent.String())
				}
			}
		}
	}
}

// decodeDeploymentTransaction creates a human-friendly description of a deployment transaction
func decodeDeploymentTransaction(deployment *ContractDeployedEvent, indexer *contracts.Indexer) string {
	var contractName string
	if indexer != nil {
		if contractInfo := indexer.GetContractByBytecodeHash(deployment.Deployment.BytecodeHash.Hex()); contractInfo != nil {
			contractName = contractInfo.Name
		} else {
			contractName = "Unknown"
		}
	} else {
		contractName = "Unknown"
	}
	
	// Try to decode constructor args if present
	var constructorDesc string
	if len(deployment.Deployment.ConstructorArgs) > 0 {
		// Try to decode constructor arguments using the ABI parser
		parser := abiPkg.NewParser(".")
		decodedArgs, err := parser.DecodeConstructorArgs(contractName, deployment.Deployment.ConstructorArgs)
		if err == nil && decodedArgs != "" {
			constructorDesc = fmt.Sprintf("(%s)", decodedArgs)
		} else {
			constructorDesc = fmt.Sprintf("(%d bytes args)", len(deployment.Deployment.ConstructorArgs))
		}
	} else {
		constructorDesc = "()"
	}
	
	// Format as: create3(new Counter()) or create3(new SampleToken(224 bytes args))
	strategy := strings.ToLower(deployment.Deployment.CreateStrategy)
	contractNameFormatted := fmt.Sprintf("%s%s%s", ColorCyan, contractName, ColorReset)
	addressFormatted := fmt.Sprintf("%s%s%s", ColorGreen, deployment.Location.Hex()[:10]+"...", ColorReset)
	
	return fmt.Sprintf("%s(new %s%s) → %s", strategy, contractNameFormatted, constructorDesc, addressFormatted)
}



// ColorizeContractName returns a colorized contract name based on its type
func ColorizeContractName(name string) string {
	lowerName := strings.ToLower(name)
	
	if strings.Contains(lowerName, "proxy") {
		return fmt.Sprintf("%s%s%s", ColorBlue, name, ColorReset)
	} else if strings.Contains(lowerName, "library") || strings.Contains(lowerName, "lib") {
		return fmt.Sprintf("%s%s%s", ColorPurple, name, ColorReset)
	} else if strings.Contains(lowerName, "token") {
		return fmt.Sprintf("%s%s%s", ColorYellow, name, ColorReset)
	}
	
	return fmt.Sprintf("%s%s%s", ColorCyan, name, ColorReset)
}

// ColorizeAddress returns a colorized address string
func ColorizeAddress(address string) string {
	return fmt.Sprintf("%s%s%s", ColorGray, address, ColorReset)
}

// ColorizeHash returns a colorized hash string (shortened)
func ColorizeHash(hash string) string {
	if len(hash) > 10 {
		hash = hash[:10] + "..."
	}
	return fmt.Sprintf("%s%s%s", ColorGray, hash, ColorReset)
}

// FormatDeploymentSummary formats a deployment summary with colors
func FormatDeploymentSummary(deployment *ContractDeployedEvent, contractName string) string {
	name := ColorizeContractName(contractName)
	address := fmt.Sprintf("%s%s%s", ColorGreen, deployment.Location.Hex(), ColorReset)
	salt := ColorizeHash(deployment.Deployment.Salt.Hex())
	
	return fmt.Sprintf("%s at %s (salt: %s)", name, address, salt)
}

// PrintDeploymentBanner prints a colored banner for deployment operations
func PrintDeploymentBanner(title string, network string, profile string) {
	fmt.Printf("\n%s%s%s%s\n", ColorBold, ColorCyan, title, ColorReset)
	fmt.Printf("Network: %s%s%s\n", ColorBlue, network, ColorReset)
	fmt.Printf("Profile: %s%s%s\n", ColorBlue, profile, ColorReset)
	fmt.Println()
}

// PrintSuccessMessage prints a success message with green color
func PrintSuccessMessage(message string) {
	fmt.Printf("%s✅ %s%s\n", ColorGreen, message, ColorReset)
}

// PrintWarningMessage prints a warning message with yellow color
func PrintWarningMessage(message string) {
	fmt.Printf("%s⚠️  %s%s\n", ColorYellow, message, ColorReset)
}

// PrintErrorMessage prints an error message with red color
func PrintErrorMessage(message string) {
	fmt.Printf("%s❌ %s%s\n", ColorRed, message, ColorReset)
}

// PrintInfoMessage prints an info message with blue color
func PrintInfoMessage(message string) {
	fmt.Printf("%sℹ️  %s%s\n", ColorBlue, message, ColorReset)
}
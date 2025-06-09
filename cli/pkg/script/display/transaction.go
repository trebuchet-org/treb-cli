package display

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// TransactionDisplay handles enhanced transaction display with event parsing
type TransactionDisplay struct {
	display      *Display
	abiResolver  abi.ABIResolver
	eventParser  *events.EventParser
	parsedEvents map[string]interface{} // Cache for parsed events by topic0
}

// NewTransactionDisplay creates a new transaction display handler
func NewTransactionDisplay(display *Display) *TransactionDisplay {
	return &TransactionDisplay{
		display:      display,
		eventParser:  events.NewEventParser(),
		parsedEvents: make(map[string]interface{}),
	}
}

// SetABIResolver sets the ABI resolver for event decoding
func (td *TransactionDisplay) SetABIResolver(resolver abi.ABIResolver) {
	td.abiResolver = resolver
}

// DisplayTransactionWithEvents displays a transaction with all its events
func (td *TransactionDisplay) DisplayTransactionWithEvents(tx *parser.Transaction) {
	// Display basic transaction info
	td.displayTransactionHeader(tx)
	
	// If we have trace data, display the full trace tree
	if tx.TraceData != nil && len(tx.TraceData.Arena) > 0 {
		td.displayTraceTree(tx.TraceData)
	} else if td.display.verbose {
		// TODO: In verbose mode without trace data, we could try to extract events 
		// from the main ScriptOutput by correlating transaction IDs with events.
		// This would help when TraceOutputs are not available.
		fmt.Printf("│  %s(No trace data available for detailed trace analysis)%s\n", ColorGray, ColorReset)
	}
	
	// Display transaction footer with gas, block, etc.
	td.displayTransactionFooter(tx)
}

// displayTransactionHeader displays the transaction header
func (td *TransactionDisplay) displayTransactionHeader(tx *parser.Transaction) {
	// Determine status color and text
	var statusColor *color.Color
	var statusText string
	
	switch tx.Status {
	case types.TransactionStatusSimulated:
		statusColor = color.New(color.FgWhite, color.Faint)
		statusText = "simulated"
	case types.TransactionStatusQueued:
		statusColor = color.New(color.FgYellow)
		statusText = "queued   "
	case types.TransactionStatusExecuted:
		statusColor = color.New(color.FgGreen)
		statusText = "executed "
	case types.TransactionStatusFailed:
		statusColor = color.New(color.FgRed)
		statusText = "failed   "
	default:
		statusColor = color.New(color.FgWhite)
		statusText = "unknown  "
	}
	
	// Format transaction summary
	to := td.display.reconcileAddress(tx.Transaction.To)
	sender := td.display.reconcileAddress(tx.Sender)
	
	// Try to decode the transaction
	var methodStr string
	decoded := td.display.transactionDecoder.DecodeTransaction(
		tx.Transaction.To, 
		tx.Transaction.Data, 
		tx.Transaction.Value, 
		tx.ReturnData,
	)
	
	if decoded != nil && decoded.Method != "" {
		methodStr = decoded.Method
		
		// Format arguments
		if len(decoded.Inputs) > 0 {
			var argStrs []string
			for _, arg := range decoded.Inputs {
				valueStr := td.formatArgValue(arg.Value)
				if arg.Name != "" {
					argStrs = append(argStrs, fmt.Sprintf("%s: %s", arg.Name, valueStr))
				} else {
					argStrs = append(argStrs, valueStr)
				}
			}
			
			argsJoined := strings.Join(argStrs, ", ")
			
			// Truncate args if too long in non-verbose mode
			maxLen := 60
			if len(argsJoined) > maxLen && !td.display.verbose {
				argsJoined = argsJoined[:maxLen-3] + "..."
			}
			
			methodStr += fmt.Sprintf("(%s)", argsJoined)
		} else {
			methodStr += "()"
		}
	} else {
		// Fallback to function selector
		if len(tx.Transaction.Data) >= 4 {
			methodStr = fmt.Sprintf("0x%x", tx.Transaction.Data[:4])
		} else {
			methodStr = "transfer()"
		}
	}
	
	// Print header (no indent before status)
	fmt.Printf("\n%s %s → %s::%s\n", 
		statusColor.Sprint(statusText),
		sender, to, methodStr)
}

// displayTraceTree displays the full trace tree with calls and events
func (td *TransactionDisplay) displayTraceTree(trace *forge.TraceOutput) {
	if len(trace.Arena) == 0 {
		return
	}
	
	// Start with the root node
	td.displayTraceNode(&trace.Arena[0], trace.Arena, []bool{}, true)
}

// displayTraceNode recursively displays a trace node and its children
func (td *TransactionDisplay) displayTraceNode(node *forge.TraceNode, arena []forge.TraceNode, isLast []bool, isRoot bool) {
	// Process items in order
	for i, orderItem := range node.Ordering {
		isLastItem := i == len(node.Ordering)-1
		
		if orderItem.Log != nil {
			// Display log event
			if *orderItem.Log < len(node.Logs) {
				td.displayLogInTree(node.Logs[*orderItem.Log], node.Trace.Address, isLast, isLastItem)
			}
		} else if orderItem.Call != nil {
			// Display subcall
			childIdx := *orderItem.Call
			if childIdx < len(node.Children) {
				childNodeIdx := node.Children[childIdx]
				if childNodeIdx < len(arena) {
					childNode := &arena[childNodeIdx]
					td.displayCallInTree(childNode, arena, isLast, isLastItem)
				}
			}
		}
	}
}

// displayCallInTree displays a call node in the tree
func (td *TransactionDisplay) displayCallInTree(node *forge.TraceNode, arena []forge.TraceNode, parentIsLast []bool, isLast bool) {
	// Build tree prefix
	prefix := td.buildTreePrefix(parentIsLast, isLast)
	
	// Format the call
	callStr := td.formatCall(node)
	
	// Display the call
	fmt.Printf("%s%s\n", prefix, callStr)
	
	// Recursively display children
	newIsLast := append(append([]bool{}, parentIsLast...), isLast)
	td.displayTraceNode(node, arena, newIsLast, false)
}

// displayLogInTree displays a log event in the tree
func (td *TransactionDisplay) displayLogInTree(log forge.LogEntry, address common.Address, parentIsLast []bool, isLast bool) {
	// Build tree prefix
	prefix := td.buildTreePrefix(parentIsLast, isLast)
	
	// Format the event
	eventStr := td.formatLogEvent(log, address)
	
	// Display the event
	fmt.Printf("%s%s\n", prefix, eventStr)
}

// buildTreePrefix builds the tree prefix for proper indentation
func (td *TransactionDisplay) buildTreePrefix(parentIsLast []bool, isLast bool) string {
	var prefix string
	
	// Build parent prefixes
	for _, last := range parentIsLast {
		if last {
			prefix += "   "
		} else {
			prefix += "│  "
		}
	}
	
	// Add current item prefix
	if isLast {
		prefix += "└─ "
	} else {
		prefix += "├─ "
	}
	
	return prefix
}

// formatCall formats a call for display
func (td *TransactionDisplay) formatCall(node *forge.TraceNode) string {
	// Get the target address
	to := td.display.reconcileAddress(node.Trace.Address)
	
	// Try to decode the call
	var callStr string
	if node.Trace.Decoded != nil && node.Trace.Decoded.CallData != nil {
		// Use decoded signature
		callStr = node.Trace.Decoded.CallData.Signature
		
		// Add args if available and not too long
		if len(node.Trace.Decoded.CallData.Args) > 0 && !strings.Contains(callStr, "(") {
			args := strings.Join(node.Trace.Decoded.CallData.Args, ", ")
			if len(args) > 40 {
				args = args[:37] + "..."
			}
			callStr += fmt.Sprintf("(%s)", args)
		}
	} else {
		// Fallback to selector or call type
		if len(node.Trace.Data) >= 10 { // 0x + 8 chars for selector
			callStr = node.Trace.Data[:10] + "..."
		} else if node.Trace.Kind == "CREATE" || node.Trace.Kind == "CREATE2" {
			callStr = fmt.Sprintf("%s(%d bytes)", strings.ToLower(node.Trace.Kind), len(node.Trace.Data)/2-1)
		} else {
			callStr = "call()"
		}
	}
	
	// Format based on call kind
	var formatted string
	switch node.Trace.Kind {
	case "CREATE", "CREATE2":
		formatted = fmt.Sprintf("%s%s%s", ColorGreen, callStr, ColorReset)
		if node.Trace.Address != (common.Address{}) {
			formatted += fmt.Sprintf(" → %s", node.Trace.Address.Hex())
		}
	case "STATICCALL":
		formatted = fmt.Sprintf("%s → %s::%s%s%s (view)", to, ColorBlue, callStr, ColorReset)
	case "DELEGATECALL":
		formatted = fmt.Sprintf("%s → %s::%s%s%s (delegate)", to, ColorYellow, callStr, ColorReset)
	default:
		formatted = fmt.Sprintf("%s → %s::%s", to, callStr)
	}
	
	// Add revert reason if failed
	if !node.Trace.Success {
		formatted += fmt.Sprintf(" %s[REVERTED]%s", ColorRed, ColorReset)
		if node.Trace.Output != "" && node.Trace.Output != "0x" {
			// Try to decode revert reason
			// TODO: Implement revert reason decoding
		}
	}
	
	return formatted
}

// formatLogEvent formats a log event for display
func (td *TransactionDisplay) formatLogEvent(log forge.LogEntry, address common.Address) string {
	var eventStr string
	
	// Check if forge decoded it
	if log.Decoded.Name != "" {
		eventStr = fmt.Sprintf("%s%s%s", ColorCyan, log.Decoded.Name, ColorReset)
		
		// Add args
		if len(log.Decoded.Params) > 0 {
			args := td.formatDecodedLogParams(log.Decoded.Params)
			if args != "" {
				eventStr += fmt.Sprintf("(%s)", args)
			}
		} else {
			eventStr += "()"
		}
	} else if len(log.RawLog.Topics) > 0 {
		// Try to identify well-known events
		if name, ok := getWellKnownEventName(log.RawLog.Topics[0]); ok {
			eventStr = fmt.Sprintf("%s%s()%s", ColorCyan, name, ColorReset)
		} else {
			// Unknown event
			eventStr = fmt.Sprintf("%sUnknownEvent(%s)%s", ColorGray, log.RawLog.Topics[0].Hex()[:10], ColorReset)
		}
	} else {
		eventStr = fmt.Sprintf("%sAnonymousEvent%s", ColorGray, ColorReset)
	}
	
	return fmt.Sprintf("📝 %s", eventStr)
}

// formatDecodedLogParams formats decoded log parameters
func (td *TransactionDisplay) formatDecodedLogParams(params [][]string) string {
	var argStrs []string
	
	for _, param := range params {
		if len(param) >= 2 {
			name := param[0]
			value := param[1]
			
			// Truncate value if needed
			formattedValue := td.formatArgValue(value)
			argStrs = append(argStrs, fmt.Sprintf("%s: %s", name, formattedValue))
		}
	}
	
	result := strings.Join(argStrs, ", ")
	
	// Truncate if too long
	if len(result) > 60 && !td.display.verbose {
		return result[:57] + "..."
	}
	
	return result
}


// formatArgValue formats a single argument value with appropriate truncation
func (td *TransactionDisplay) formatArgValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Check if it's an address (42 chars with 0x)
		if len(v) == 42 && strings.HasPrefix(v, "0x") {
			// Show first 6 and last 4 chars
			return v[:6] + "..." + v[len(v)-4:]
		}
		// For other hex strings, truncate if long
		if strings.HasPrefix(v, "0x") && len(v) > 20 {
			return v[:10] + "..."
		}
		// For regular strings, limit length
		if len(v) > 32 {
			return v[:29] + "..."
		}
		return v
	case []byte:
		// Convert byte array to hex string
		hexStr := "0x" + common.Bytes2Hex(v)
		if len(hexStr) > 20 {
			return hexStr[:10] + "..."
		}
		return hexStr
	case [32]byte:
		// Convert 32-byte array to hex string
		hexStr := "0x" + common.Bytes2Hex(v[:])
		// For 32-byte hashes, show first 10 chars
		return hexStr[:10] + "..."
	case common.Hash:
		hexStr := v.Hex()
		return hexStr[:10] + "..."
	case common.Address:
		addr := v.Hex()
		return addr[:6] + "..." + addr[len(addr)-4:]
	case *common.Address:
		if v == nil {
			return "null"
		}
		addr := v.Hex()
		return addr[:6] + "..." + addr[len(addr)-4:]
	default:
		// Check if it's a slice of integers (common for byte arrays)
		str := fmt.Sprintf("%v", v)
		if strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]") {
			// Might be a byte slice shown as [1 2 3 4...]
			// Try to detect and handle it
			if _, ok := v.([]interface{}); ok {
				// Convert to hex if it looks like bytes
				return "0x..." // Simplified for unknown byte arrays
			}
		}
		
		if len(str) > 32 {
			return str[:29] + "..."
		}
		return str
	}
}

// displayTransactionFooter displays transaction footer info
func (td *TransactionDisplay) displayTransactionFooter(tx *parser.Transaction) {
	var details []string
	
	if tx.TxHash != nil {
		details = append(details, fmt.Sprintf("Tx: %s", tx.TxHash.Hex()))
	}
	
	if tx.SafeTxHash != nil {
		details = append(details, fmt.Sprintf("Safe Tx: %s", tx.SafeTxHash.Hex()))
	}
	
	if tx.BlockNumber != nil {
		details = append(details, fmt.Sprintf("Block: %d", *tx.BlockNumber))
	}
	
	if tx.GasUsed != nil {
		details = append(details, fmt.Sprintf("Gas: %d", *tx.GasUsed))
	}
	
	if len(details) > 0 {
		fmt.Printf("└─ %s%s%s\n", ColorGray, strings.Join(details, " | "), ColorReset)
	}
}

// Helper function to get well-known event names
func getWellKnownEventName(topic0 common.Hash) (string, bool) {
	wellKnownEvents := map[string]string{
		// ERC20 events
		crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex(): "Transfer",
		crypto.Keccak256Hash([]byte("Approval(address,address,uint256)")).Hex(): "Approval",
		
		// Proxy events
		crypto.Keccak256Hash([]byte("Upgraded(address)")).Hex():                              "Upgraded",
		crypto.Keccak256Hash([]byte("AdminChanged(address,address)")).Hex():                  "AdminChanged",
		crypto.Keccak256Hash([]byte("BeaconUpgraded(address)")).Hex():                        "BeaconUpgraded",
		crypto.Keccak256Hash([]byte("ProxyDeployed(address,address,string)")).Hex():          "ProxyDeployed",
		
		// Ownable events
		crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)")).Hex():          "OwnershipTransferred",
		
		// Safe events
		crypto.Keccak256Hash([]byte("SafeSetup(address,address[],uint256,address,address)")).Hex(): "SafeSetup",
		crypto.Keccak256Hash([]byte("ExecutionSuccess(bytes32,uint256)")).Hex():                    "ExecutionSuccess",
		crypto.Keccak256Hash([]byte("ExecutionFailure(bytes32,uint256)")).Hex():                    "ExecutionFailure",
	}
	
	if name, ok := wellKnownEvents[topic0.Hex()]; ok {
		return name, true
	}
	return "", false
}
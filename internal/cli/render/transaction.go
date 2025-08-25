package render

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/adapters/abi"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// TransactionRenderer handles enhanced transaction display with event parsing
type TransactionRenderer struct {
	abiResolver  usecase.ABIResolver
	txDecoder    *abi.TransactionDecoder
	eventDecoder *abi.EventDecoder

	repo      usecase.DeploymentRepository
	log       *slog.Logger
	tx        *forge.Transaction
	execution *forge.HydratedRunResult
}

// NewTransactionRenderer creates a new transaction display handler
func NewTransactionRenderer(
	abiResolver usecase.ABIResolver,
	repo usecase.DeploymentRepository,
	log *slog.Logger,
) *TransactionRenderer {
	return &TransactionRenderer{
		repo:        repo,
		abiResolver: abiResolver,
		log:         log.With("component", "TxRenderer"),
	}
}

func (tr *TransactionRenderer) WithExecution(execution *forge.HydratedRunResult) *TransactionRenderer {
	scopedTr := *tr
	scopedTr.execution = execution
	tr.abiResolver.SetExecution(execution)
	scopedTr.txDecoder = abi.NewTransactionDecoder(
		tr.abiResolver,
		tr.repo,
		execution,
		tr.log,
	)
	scopedTr.eventDecoder = abi.NewEventDecoder(
		tr.abiResolver,
		tr.log,
	)
	return &scopedTr
}

func (tr *TransactionRenderer) WithTx(tx *forge.Transaction) *TransactionRenderer {
	scopedTr := *tr
	scopedTr.tx = tx
	return &scopedTr
}

func (tr *TransactionRenderer) senderName(address common.Address) string {
	for _, sender := range tr.execution.Senders.SenderInitConfigs {
		if sender.Account == address {
			return sender.Name
		}
	}

	return "<unknown sender>"
}

// DisplayTransactionWithEvents displays a transaction with all its events
func (tr *TransactionRenderer) DisplayTransactionWithEvents(tx *forge.Transaction) {
	scoped := tr.WithTx(tx)

	// Display basic transaction info
	scoped.displayTransactionHeader()

	// If we have trace data, display the full trace tree
	if tx.TraceData != nil && len(tx.TraceData.Arena) > 0 {
		scoped.displayTraceTree()
	} else {
		// TODO: In verbose mode without trace data, we could try to extract events
		// from the main ScriptOutput by correlating transaction IDs with events.
		// This would help when TraceOutputs are not available.
		fmt.Printf("│  %s\n", gray.Sprint("(No trace data available for detailed trace analysis)"))
	}

	// Display transaction footer with gas, block, etc.
	scoped.displayTransactionFooter()
}

func (tr *TransactionRenderer) rootTxStatus() string {
	var statusColor *color.Color
	var statusText string

	switch tr.tx.Status {
	case models.TransactionStatusSimulated:
		statusColor = color.New(color.FgWhite, color.Faint)
		statusText = "simulated"
	case models.TransactionStatusQueued:
		statusColor = color.New(color.FgYellow)
		statusText = "queued   "
	case models.TransactionStatusExecuted:
		statusColor = color.New(color.FgGreen)
		statusText = "executed "
	case models.TransactionStatusFailed:
		statusColor = color.New(color.FgRed)
		statusText = "failed   "
	default:
		statusColor = color.New(color.FgWhite)
		statusText = "unknown  "
	}

	return statusColor.Sprint(statusText)
}

func (tr *TransactionRenderer) formatDecodedTx(decoded *abi.DecodedTransaction) string {
	var method, args string
	var destination = tr.txDecoder.GetLabel(decoded.To)
	if decoded != nil && decoded.Method != "" {
		method = decoded.Method

		// Format arguments
		if len(decoded.Inputs) > 0 {
			var argStrs []string
			for _, arg := range decoded.Inputs {
				valueStr := tr.formatArgValue(arg.Value)
				if arg.Name != "" {
					argStrs = append(argStrs, fmt.Sprintf("%s: %s", arg.Name, valueStr))
				} else {
					argStrs = append(argStrs, valueStr)
				}
			}

			args = strings.Join(argStrs, ", ")
		}
	} else {
		// Fallback to function selector
		if len(tr.tx.Transaction.Data) >= 4 {
			method = fmt.Sprintf("0x%x", tr.tx.Transaction.Data[:4])
		}
	}

	return fmt.Sprintf("%s::%s(%s)", destination, yellow.Sprint(method), args)

}

// displayTransactionHeader displays the transaction header
func (tr *TransactionRenderer) displayTransactionHeader() {
	// Format transaction summary
	sender := tr.senderName(tr.tx.Sender)

	// Try to decode the transaction
	decoded := tr.txDecoder.DecodeTransaction(
		tr.tx.Transaction.To,
		tr.tx.Transaction.Data,
		tr.tx.Transaction.Value,
		tr.tx.ReturnData,
	)

	// Print header (no indent before status)
	// Extract function name and args for coloring
	fmt.Printf("\n%s %s → %s\n", tr.rootTxStatus(), green.Sprint(sender), tr.formatDecodedTx(decoded))
}

// displayTraceTree displays the full trace tree with calls and events
func (tr *TransactionRenderer) displayTraceTree() {
	trace := tr.tx.TraceData
	if len(trace.Arena) == 0 {
		return
	}

	// Start with the root node
	tr.displayTraceNode(&trace.Arena[0], trace.Arena, []bool{}, true)
}

// displayTraceNode recursively displays a trace node and its children
func (tr *TransactionRenderer) displayTraceNode(node *forge.TraceNode, arena []forge.TraceNode, isLast []bool, isRoot bool) {
	// Process items in order
	for i, orderItem := range node.Ordering {
		isLastItem := i == len(node.Ordering)-1

		if orderItem.Log != nil {
			// Display log event
			if *orderItem.Log < len(node.Logs) {
				tr.displayLogInTree(&node.Logs[*orderItem.Log], node.Trace.Address, isLast, isLastItem)
			}
		} else if orderItem.Call != nil {
			// Display subcall
			childIdx := *orderItem.Call
			if childIdx < len(node.Children) {
				childNodeIdx := node.Children[childIdx]
				if childNodeIdx < len(arena) {
					childNode := &arena[childNodeIdx]
					tr.displayCallInTree(childNode, arena, isLast, isLastItem)
				}
			}
		}
	}
}

// displayTraceNodeWithReturnCheck displays trace node children, adjusting isLast based on whether there's a return address
func (tr *TransactionRenderer) displayTraceNodeWithReturnCheck(node *forge.TraceNode, arena []forge.TraceNode, isLast []bool, hasReturnAfter bool) {
	// Process items in order
	for i, orderItem := range node.Ordering {
		// If there's a return address after, no item is truly the last
		isLastItem := i == len(node.Ordering)-1 && !hasReturnAfter

		if orderItem.Log != nil {
			// Display log event
			if *orderItem.Log < len(node.Logs) {
				tr.displayLogInTree(&node.Logs[*orderItem.Log], node.Trace.Address, isLast, isLastItem)
			}
		} else if orderItem.Call != nil {
			// Display subcall
			childIdx := *orderItem.Call
			if childIdx < len(node.Children) {
				childNodeIdx := node.Children[childIdx]
				if childNodeIdx < len(arena) {
					childNode := &arena[childNodeIdx]
					tr.displayCallInTree(childNode, arena, isLast, isLastItem)
				}
			}
		}
	}
}

// displayCallInTree displays a call node in the tree
func (tr *TransactionRenderer) displayCallInTree(node *forge.TraceNode, arena []forge.TraceNode, parentIsLast []bool, isLast bool) {
	// Build tree prefix
	prefix := tr.buildTreePrefix(parentIsLast, isLast)

	// Special handling for CREATE/CREATE2
	if node.Trace.Kind == "CREATE" || node.Trace.Kind == "CREATE2" {
		// Display the main create call with multiline constructor args
		tr.displayCreateCallWithArgs(node, prefix)

		// Create new parentIsLast for children
		newIsLast := append(append([]bool{}, parentIsLast...), isLast)

		// Check if we have a return address
		hasReturnAddress := node.Trace.Address != (common.Address{})

		// Display children (events, etc)
		tr.displayTraceNodeWithReturnCheck(node, arena, newIsLast, hasReturnAddress)

		// Display return address as last item
		if hasReturnAddress {
			returnPrefix := tr.buildTreePrefix(newIsLast, true)
			fmt.Printf("%s%s %s\n", returnPrefix, gray.Sprint("[return]"), node.Trace.Address.Hex())
		}
	} else {
		// Format regular calls
		callStr := tr.formatCall(node)
		fmt.Printf("%s%s\n", prefix, callStr)

		// Recursively display children
		newIsLast := append(append([]bool{}, parentIsLast...), isLast)
		tr.displayTraceNode(node, arena, newIsLast, false)
	}
}

// displayLogInTree displays a log event in the tree
func (tr *TransactionRenderer) displayLogInTree(log *forge.LogEntry, address common.Address, parentIsLast []bool, isLast bool) {
	// Build tree prefix
	prefix := tr.buildTreePrefix(parentIsLast, isLast)

	decodedLog, err := tr.eventDecoder.DecodeEvent(log, address)
	if err != nil {
		tr.log.Warn("Could not decode log", "error", err)
	}
	// Format the event
	eventStr := tr.formatLogEvent(decodedLog)

	// Display the event
	fmt.Printf("%s%s\n", prefix, eventStr)
}

// buildTreePrefix builds the tree prefix for proper indentation
func (tr *TransactionRenderer) buildTreePrefix(parentIsLast []bool, isLast bool) string {
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
func (tr *TransactionRenderer) formatCall(node *forge.TraceNode) string {
	// Get the target address
	decodedTx := tr.txDecoder.DecodeTraceInfo(&node.Trace)
	formatted := tr.formatDecodedTx(decodedTx)

	// Add revert reason if failed
	if !node.Trace.Success {
		formatted += red.Sprintf(" [REVERTED]")
		// TODO: Implement revert reason decoding when output is present
	}

	return formatted
}

// formatLogEvent formats a log event for display
func (tr *TransactionRenderer) formatLogEvent(log *forge.LogEntry) string {
	var eventStr string
	tr.log.Debug("Decoding event", "log", log)

	// Check if forge decoded it
	if log.Decoded.Name != "" {
		eventStr = cyan.Sprintf(log.Decoded.Name)

		// Add args
		if len(log.Decoded.Params) > 0 {
			args := tr.formatDecodedLogParams(log.Decoded.Params)
			if args != "" {
				eventStr += fmt.Sprintf("(%s)", args)
			}
		} else {
			eventStr += "()"
		}
	} else if len(log.RawLog.Topics) > 0 {
		// Try to identify well-known events
		if name, ok := getWellKnownEventName(log.RawLog.Topics[0]); ok {
			eventStr = cyan.Sprint(name)
		} else {
			// Unknown event
			eventStr = gray.Sprintf("UnknownEvent(%s)", log.RawLog.Topics[0].Hex()[:10])
		}
	} else {
		eventStr = gray.Sprintf("UnknownEvent")
	}

	return eventStr
}

// formatDecodedLogParams formats decoded log parameters
func (tr *TransactionRenderer) formatDecodedLogParams(params [][]string) string {
	var argStrs []string

	for _, param := range params {
		if len(param) >= 2 {
			name := param[0]
			value := param[1]

			// Truncate value if needed
			formattedValue := tr.formatArgValue(value)
			argStrs = append(argStrs, fmt.Sprintf("%s: %s", name, formattedValue))
		}
	}

	result := strings.Join(argStrs, ", ")

	// Truncate if too long
	if len(result) > 60 {
		return result[:57] + "..."
	}

	return result
}

// formatArgValue formats a single argument value with appropriate truncation
func (tr *TransactionRenderer) formatArgValue(value any) string {
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
func (tr *TransactionRenderer) displayTransactionFooter() {
	tx := tr.tx
	var details []string

	if tx.TxHash != nil {
		details = append(details, fmt.Sprintf("Tx: %s", tx.TxHash.Hex()))
	}

	if tx.SafeTransaction != nil {
		details = append(details, fmt.Sprintf("Safe Tx: %s", common.Hash(tx.SafeTransaction.SafeTxHash).Hex()))
	}

	if tx.BlockNumber != nil {
		details = append(details, fmt.Sprintf("Block: %d", *tx.BlockNumber))
	}

	if tx.GasUsed != nil {
		details = append(details, fmt.Sprintf("Gas: %d", *tx.GasUsed))
	}

	if len(details) > 0 {
		fmt.Printf("   %s\n", gray.Sprint(strings.Join(details, " | ")))
	}
}

// Helper function to get well-known event names
func getWellKnownEventName(topic0 common.Hash) (string, bool) {
	wellKnownEvents := map[string]string{
		// ERC20 events
		crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex(): "Transfer",
		crypto.Keccak256Hash([]byte("Approval(address,address,uint256)")).Hex(): "Approval",

		// Proxy events
		crypto.Keccak256Hash([]byte("Upgraded(address)")).Hex():                     "Upgraded",
		crypto.Keccak256Hash([]byte("AdminChanged(address,address)")).Hex():         "AdminChanged",
		crypto.Keccak256Hash([]byte("BeaconUpgraded(address)")).Hex():               "BeaconUpgraded",
		crypto.Keccak256Hash([]byte("ProxyDeployed(address,address,string)")).Hex(): "ProxyDeployed",

		// Ownable events
		crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)")).Hex(): "OwnershipTransferred",

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

func (tr *TransactionRenderer) isDeployedContract(address common.Address) bool {
	for _, dep := range tr.execution.Deployments {
		if dep.Address == address {
			return true
		}
	}

	return false
}

// displayCreateCallWithArgs displays a CREATE call with multiline constructor arguments
func (tr *TransactionRenderer) displayCreateCallWithArgs(node *forge.TraceNode, prefix string) {
	// Get contract name
	contractName := tr.getContractName(node)

	// Check if this is a deployment from transaction
	deploymentIcon := ""
	if tr.isDeployedContract(node.Trace.Address) {
		deploymentIcon = "🚀 "
	}

	// Get constructor arguments
	var decodedConstructor = tr.getDecodedConstructor(node)

	// Format the opening line
	if contractName != "" {
		if decodedConstructor != nil && len(decodedConstructor.Inputs) > 0 {
			// Display with opening parenthesis
			fmt.Printf("%s%s%s(\n", prefix, deploymentIcon, green.Sprintf("new %s", contractName))

			// For constructor arguments, we need to maintain the tree structure
			// The prefix ends with either "├─ " or "└─ " which we need to replace with proper spacing
			basePrefix := prefix
			if strings.HasSuffix(prefix, "├─ ") {
				basePrefix = prefix[:len(prefix)-len("├─ ")] + "│  │  "
			} else if strings.HasSuffix(prefix, "└─ ") {
				basePrefix = prefix[:len(prefix)-len("└─ ")] + "   │  "
			}

			// Display each argument on its own line
			for i, input := range decodedConstructor.Inputs {
				isLast := (i == len(decodedConstructor.Inputs)-1)

				// Format the argument value
				valueStr := abi.FormatValue(input.Value, input.Type)

				// Build the line with proper indentation
				argLine := basePrefix + "  "
				if input.Name != "" && !strings.HasPrefix(input.Name, "arg") {
					argLine += fmt.Sprintf("%s: %s", gray.Sprintf("%s:", input.Name), valueStr)
				} else {
					argLine += valueStr
				}

				// Add comma if not last
				if !isLast {
					argLine += ","
				}

				fmt.Printf("%s\n", argLine)
			}

			// Close parenthesis
			fmt.Printf("%s%s\n", basePrefix, green.Sprint(")"))
		} else {
			// No constructor args
			fmt.Printf("%s%s%s\n", prefix, deploymentIcon, green.Sprintf("new %s()", contractName))
		}
	} else {
		// Fallback to showing bytecode size
		fmt.Printf("%s%s%s\n", prefix, deploymentIcon, green.Sprintf("%s(%d bytes)", strings.ToLower(node.Trace.Kind), len(node.Trace.Data)/2-1))
	}
}

// getContractName extracts the contract name for a CREATE node
func (tr *TransactionRenderer) getContractName(node *forge.TraceNode) string {
	contractName := tr.txDecoder.GetLabel(node.Trace.Address)
	// Check if there's decoded info
	if strings.Contains(contractName, "Unknown") && node.Trace.Decoded != nil && node.Trace.Decoded.Label != "" {
		contractName = node.Trace.Decoded.Label
	}

	return contractName
}

// getDecodedConstructor gets decoded constructor arguments for a CREATE node
func (tr *TransactionRenderer) getDecodedConstructor(node *forge.TraceNode) *abi.DecodedConstructor {
	var decodedConstructor *abi.DecodedConstructor

	// Try to decode from deployment record
	if tr.execution != nil {
		for _, dep := range tr.execution.Deployments {
			if dep.Address == node.Trace.Address {
				if len(dep.Event.ConstructorArgs) > 0 {
					decoded, err := tr.txDecoder.DecodeConstructor(
						node.Trace.Address,
						dep.Event.ConstructorArgs,
					)
					if err == nil && decoded != nil {
						decodedConstructor = decoded
					}
				}
				break
			}
		}
	}

	// If no decoded constructor, check if forge decoded it
	if decodedConstructor == nil && node.Trace.Decoded != nil && node.Trace.Decoded.CallData != nil && len(node.Trace.Decoded.CallData.Args) > 0 {
		// Create a decoded constructor from forge's decoded args
		decodedConstructor = &abi.DecodedConstructor{
			Inputs: make([]abi.DecodedInput, len(node.Trace.Decoded.CallData.Args)),
		}
		for i, arg := range node.Trace.Decoded.CallData.Args {
			// Try to extract parameter name if available
			paramName := fmt.Sprintf("arg%d", i)
			decodedConstructor.Inputs[i] = abi.DecodedInput{
				Name:  paramName,
				Type:  "unknown",
				Value: arg,
			}
		}
	}

	return decodedConstructor
}

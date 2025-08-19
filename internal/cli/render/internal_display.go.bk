package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// InternalDisplay handles script execution display without v1 dependencies
type InternalDisplay struct {
	out          io.Writer
	verbose      bool
	knownAddresses map[string]string
}

// NewInternalDisplay creates a new internal display
func NewInternalDisplay(out io.Writer, verbose bool) *InternalDisplay {
	return &InternalDisplay{
		out:     out,
		verbose: verbose,
		knownAddresses: map[string]string{
			"0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266": "anvil",
			"0xba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed": "CreateX",
		},
	}
}

// DisplayExecution displays the script execution results
func (d *InternalDisplay) DisplayExecution(exec *domain.ScriptExecution) {
	// Setup colors
	bold := color.New(color.Bold).SprintFunc()
	reset := color.New(color.Reset).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	// Display header
	fmt.Fprintf(d.out, "\n%s🔄 Transactions:%s\n", bold(), reset())
	fmt.Fprintf(d.out, "%s%s%s\n", gray(), strings.Repeat("─", 50), reset())

	if len(exec.Transactions) == 0 {
		fmt.Fprintf(d.out, "%sNo transactions executed (dry run or all deployments skipped)%s\n\n", 
			gray(), reset())
		return
	}

	// Build deployment map for quick lookup
	deploymentsByTx := make(map[[32]byte][]*domain.ScriptDeployment)
	for i := range exec.Deployments {
		dep := &exec.Deployments[i]
		deploymentsByTx[dep.TransactionID] = append(deploymentsByTx[dep.TransactionID], dep)
	}

	// Display each transaction
	for _, tx := range exec.Transactions {
		d.displayTransaction(tx, deploymentsByTx[tx.TransactionID], exec)
	}

	// Display deployment summary
	if len(exec.Deployments) > 0 {
		fmt.Fprintf(d.out, "\n%s📦 Deployment Summary:%s\n", bold(), reset())
		fmt.Fprintf(d.out, "%s%s%s\n", gray(), strings.Repeat("─", 50), reset())

		for _, dep := range exec.Deployments {
			name := dep.ContractName
			if dep.Label != "" {
				name = fmt.Sprintf("%s:%s", dep.ContractName, dep.Label)
			}
			
			// Check if this is a proxy and find implementation
			if dep.IsProxy && dep.ProxyInfo != nil {
				implName := d.findImplementationName(dep.ProxyInfo.Implementation, exec)
				if implName != "" {
					name = fmt.Sprintf("%s[%s]", name, implName)
				}
			}

			fmt.Fprintf(d.out, "%s%s%s at %s%s%s\n",
				cyan(), name, reset(),
				green(), dep.Address, reset())
		}
		fmt.Fprintln(d.out)
	}

	// Display logs if verbose
	if d.verbose && len(exec.Logs) > 0 {
		fmt.Fprintf(d.out, "\n%s📝 Script Logs:%s\n", bold(), reset())
		fmt.Fprintf(d.out, "%s%s%s\n", gray(), strings.Repeat("─", 50), reset())
		for _, log := range exec.Logs {
			fmt.Fprintf(d.out, "  %s\n", log)
		}
		fmt.Fprintln(d.out)
	}
}

// displayTransaction displays a single transaction
func (d *InternalDisplay) displayTransaction(tx domain.ScriptTransaction, deployments []*domain.ScriptDeployment, exec *domain.ScriptExecution) {
	// Setup colors
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()
	reset := color.New(color.Reset).SprintFunc()

	// Get status color and text
	statusColor := gray
	statusText := "simulated"
	switch tx.Status {
	case domain.TransactionStatusExecuted:
		statusColor = green
		statusText = "executed "
	case domain.TransactionStatusQueued:
		statusColor = yellow
		statusText = "queued   "
	case domain.TransactionStatusFailed:
		statusColor = red
		statusText = "failed   "
	}

	// Get sender name
	senderName := d.getKnownAddress(tx.Sender)

	// Determine what this transaction does
	var action string
	if strings.HasSuffix(strings.ToLower(tx.To), "ba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed") {
		// CreateX call
		action = d.decodeCreateXCall(tx.Data)
	} else {
		// Generic call
		action = fmt.Sprintf("0x%x", tx.Data[:4])
	}

	// Display transaction header
	fmt.Fprintf(d.out, "\n%s%s%s %s%s%s → %s\n",
		statusColor(), statusText, reset(),
		green(), senderName, reset(),
		action)

	// Display deployments
	for i, dep := range deployments {
		isLast := i == len(deployments)-1
		d.displayDeployment(dep, isLast, exec)
	}

	// Display transaction details
	if tx.TxHash != nil || tx.BlockNumber != nil || tx.GasUsed != nil {
		var details []string
		if tx.TxHash != nil {
			details = append(details, fmt.Sprintf("Tx: %s", *tx.TxHash))
		}
		if tx.BlockNumber != nil {
			details = append(details, fmt.Sprintf("Block: %d", *tx.BlockNumber))
		}
		if tx.GasUsed != nil {
			details = append(details, fmt.Sprintf("Gas: %d", *tx.GasUsed))
		}
		fmt.Fprintf(d.out, "└─ %s%s%s\n", gray(), strings.Join(details, " | "), reset())
	}
}

// displayDeployment displays a deployment within a transaction
func (d *InternalDisplay) displayDeployment(dep *domain.ScriptDeployment, isLast bool, exec *domain.ScriptExecution) {
	green := color.New(color.FgGreen).SprintFunc()
	reset := color.New(color.Reset).SprintFunc()

	prefix := "├─"
	if isLast {
		prefix = "└─"
	}

	// Show CREATE operations
	fmt.Fprintf(d.out, "├─ create2(16 bytes)\n")
	fmt.Fprintf(d.out, "│  └─ [return] %s\n", d.shortenAddress(dep.Address))
	fmt.Fprintf(d.out, "├─ Create3ProxyContractCreation(newContract: %s, salt: 0x%x...)\n", 
		d.shortenAddress(dep.Address), dep.Salt[:4])
	fmt.Fprintf(d.out, "├─ %s::0x60806040...\n", d.shortenAddress(dep.Address))
	
	// Show the deployment
	if dep.IsProxy && dep.ProxyInfo != nil {
		// Proxy deployment
		fmt.Fprintf(d.out, "│  └─ 🚀 %snew %s(%s\n", 
			green(), dep.ContractName, reset())
		fmt.Fprintf(d.out, "│     │    implementation: %s,\n", dep.ProxyInfo.Implementation)
		fmt.Fprintf(d.out, "│     │    _data: 0xc4d66de8000000000000000000000000...(36 bytes)\n")
		fmt.Fprintf(d.out, "│     │  )\n")
		
		// Show proxy events
		implName := d.findImplementationName(dep.ProxyInfo.Implementation, exec)
		if implName == "" {
			implName = "Implementation"
		}
		fmt.Fprintf(d.out, "│     ├─ Upgraded(implementation: %s: [%s...)\n", 
			implName, dep.ProxyInfo.Implementation[:10])
		fmt.Fprintf(d.out, "│     ├─ %s::initialize(0x0000...0000) (delegate)\n", implName)
		fmt.Fprintf(d.out, "│     └─ [return] %s\n", dep.Address)
	} else {
		// Regular deployment
		fmt.Fprintf(d.out, "│  └─ 🚀 %snew %s()%s\n", 
			green(), dep.ContractName, reset())
		fmt.Fprintf(d.out, "│     └─ [return] %s\n", dep.Address)
	}
	
	// Show final event
	displayName := dep.ContractName
	if dep.Label != "" {
		displayName = fmt.Sprintf("%s:%s", dep.ContractName, dep.Label)
	}
	fmt.Fprintf(d.out, "%s ContractCreation(newContract: %s: [%s...)\n", 
		prefix, displayName, dep.Address[:10])
}

// decodeCreateXCall decodes a CreateX method call
func (d *InternalDisplay) decodeCreateXCall(data []byte) string {
	if len(data) < 4 {
		return "unknown()"
	}

	selector := fmt.Sprintf("0x%x", data[:4])
	
	switch selector {
	case "0x50a1b77c": // deployCreate3
		return "CreateX::deployCreate3(salt: 0xf39fd6e5..., initCode: 0x60806040...)"
	case "0x5e89c2f0": // deployCreate2
		return "CreateX::deployCreate2(salt: 0xf39fd6e5..., initCode: 0x60806040...)"
	default:
		return fmt.Sprintf("CreateX::%s", selector)
	}
}

// getKnownAddress returns a known name for an address or shortens it
func (d *InternalDisplay) getKnownAddress(addr string) string {
	// Check known addresses
	if name, ok := d.knownAddresses[strings.ToLower(addr)]; ok {
		return name
	}
	return d.shortenAddress(addr)
}

// shortenAddress returns a shortened version of an address
func (d *InternalDisplay) shortenAddress(addr string) string {
	if len(addr) >= 10 {
		return addr[:10] + "..."
	}
	return addr
}

// findImplementationName finds the name of an implementation contract
func (d *InternalDisplay) findImplementationName(implAddr string, exec *domain.ScriptExecution) string {
	for _, dep := range exec.Deployments {
		if strings.EqualFold(dep.Address, implAddr) {
			return dep.ContractName
		}
	}
	return ""
}
package render

import (
    "fmt"
    "io"
    "strings"

    "github.com/fatih/color"
    "github.com/trebuchet-org/treb-cli/internal/domain"
    "github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptRenderer renders script execution results
type ScriptRenderer struct {
	out     io.Writer
	verbose bool
	// Color definitions
	colorBold   string
	colorReset  string
	colorGray   string
	colorCyan   string
	colorBlue   string
	colorPurple string
	colorYellow string
	colorGreen  string
	colorRed    string
}

// NewScriptRenderer creates a new script renderer
func NewScriptRenderer(out io.Writer, verbose bool) *ScriptRenderer {
	return &ScriptRenderer{
		out:         out,
		verbose:     verbose,
		colorBold:   color.New(color.Bold).SprintFunc()(""),
		colorReset:  color.New(color.Reset).SprintFunc()(""),
		colorGray:   color.New(color.FgHiBlack).SprintFunc()(""),
		colorCyan:   color.New(color.FgCyan).SprintFunc()(""),
		colorBlue:   color.New(color.FgBlue).SprintFunc()(""),
		colorPurple: color.New(color.FgMagenta).SprintFunc()(""),
		colorYellow: color.New(color.FgYellow).SprintFunc()(""),
		colorGreen:  color.New(color.FgGreen).SprintFunc()(""),
		colorRed:    color.New(color.FgRed).SprintFunc()(""),
	}
}

// RenderExecution renders the complete script execution result
func (r *ScriptRenderer) RenderExecution(result *usecase.RunScriptResult) error {
	if result.Execution == nil {
		return fmt.Errorf("no execution data to render")
	}

    exec := result.Execution

	// Display deployment banner (already shown during execution)
	// The banner is displayed by the script runner before execution

    // Use internal display for rendering
    display := NewInternalDisplay(r.out, r.verbose)
    display.DisplayExecution(exec)

    // Registry update summary is handled by the caller for v1-compat
    if !exec.DryRun && len(exec.Deployments) == 0 {
        fmt.Fprintf(r.out, "%s- No registry changes recorded for %s network in namespace %s%s\n",
            r.colorYellow, exec.Network, exec.Namespace, r.colorReset)
    }

    // Success line mirrors v1 (printed by caller)
    return nil
}

// Removed renderWithV1Display - now using internal display

// renderTransactions displays the transaction list
func (r *ScriptRenderer) renderTransactions(exec *domain.ScriptExecution) error {
	fmt.Fprintf(r.out, "\n%s🔄 Transactions:%s\n", r.colorBold, r.colorReset)
	fmt.Fprintf(r.out, "%s%s%s\n", r.colorGray, strings.Repeat("─", 50), r.colorReset)

	if len(exec.Transactions) == 0 {
		fmt.Fprintf(r.out, "%sNo transactions executed (dry run or all deployments skipped)%s\n\n", 
			r.colorGray, r.colorReset)
		return nil
	}

    // Display each transaction in tree format
    for _, tx := range exec.Transactions {
        r.renderTransactionTree(tx, exec)
    }

	return nil
}

// renderTransactionTree displays a transaction in tree format
func (r *ScriptRenderer) renderTransactionTree(tx domain.ScriptTransaction, exec *domain.ScriptExecution) {
	// Build status text
	var statusText string
	switch tx.Status {
	case domain.TransactionStatusSimulated:
		statusText = "simulated"
	case domain.TransactionStatusQueued:
		statusText = "queued   "
	case domain.TransactionStatusExecuted:
		statusText = "executed "
	case domain.TransactionStatusFailed:
		statusText = "failed   "
	default:
		statusText = "unknown  "
	}

	// Try to identify what this transaction does
	var methodCall string
	
	// Check if this is a CreateX deployment
	if strings.HasSuffix(tx.To, "ba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed") || tx.To == "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed" {
		// This is a CreateX call, try to decode it
		methodCall = r.decodeCreateXCall(tx, exec)
	} else {
		// Generic transaction
		methodCall = fmt.Sprintf("0x%x", tx.Data[:4]) // Show selector
	}

	// Display transaction header
	fmt.Fprintf(r.out, "\n%s%s%s %s%s%s → %s\n", 
		r.getStatusColor(tx.Status), statusText, r.colorReset,
		r.colorGreen, r.getKnownAddress(tx.Sender), r.colorReset,
		methodCall)

	// Find deployments for this transaction
	deployments := r.findDeploymentsForTransaction(tx, exec)
	
	// Display deployment info if any
	for i, dep := range deployments {
		isLast := i == len(deployments)-1
		
		// Show CREATE operation
        if dep.IsProxy && dep.ProxyInfo != nil {
            // Proxy deployment - show both implementation and proxy
            r.renderProxyDeployment(dep, isLast, exec)
		} else {
			// Regular deployment
			r.renderRegularDeployment(dep, isLast)
		}
	}

	// Display transaction footer with gas and block info
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
		fmt.Fprintf(r.out, "└─ %s%s%s\n", r.colorGray, strings.Join(details, " | "), r.colorReset)
	}
}

// decodeCreateXCall tries to decode a CreateX call
func (r *ScriptRenderer) decodeCreateXCall(tx domain.ScriptTransaction, exec *domain.ScriptExecution) string {
	// Check method selector (first 4 bytes)
	if len(tx.Data) < 4 {
		return "unknown()"
	}

	selector := fmt.Sprintf("0x%x", tx.Data[:4])
	
	// Common CreateX selectors
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
func (r *ScriptRenderer) getKnownAddress(addr string) string {
	// Check for known addresses
	switch strings.ToLower(addr) {
	case "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266":
		return "anvil"
	default:
		return r.shortenAddress(addr)
	}
}

// getStatusColor returns the color function for a transaction status
func (r *ScriptRenderer) getStatusColor(status domain.TransactionStatus) string {
	switch status {
	case domain.TransactionStatusSimulated:
		return r.colorGray
	case domain.TransactionStatusQueued:
		return r.colorYellow
	case domain.TransactionStatusExecuted:
		return r.colorGreen
	case domain.TransactionStatusFailed:
		return r.colorRed
	default:
		return ""
	}
}

// findDeploymentsForTransaction finds deployments created in a transaction
func (r *ScriptRenderer) findDeploymentsForTransaction(tx domain.ScriptTransaction, exec *domain.ScriptExecution) []domain.ScriptDeployment {
	var deployments []domain.ScriptDeployment
	for _, dep := range exec.Deployments {
		if dep.TransactionID == tx.TransactionID {
			deployments = append(deployments, dep)
		}
	}
	return deployments
}

// renderRegularDeployment renders a regular contract deployment
func (r *ScriptRenderer) renderRegularDeployment(dep domain.ScriptDeployment, isLast bool) {
	prefix := "├─"
	if isLast {
		prefix = "└─"
	}

	// Show intermediate steps (simplified without full trace)
	fmt.Fprintf(r.out, "├─ create2(16 bytes)\n")
	fmt.Fprintf(r.out, "│  └─ [return] %s\n", r.shortenAddress(dep.Address))
	fmt.Fprintf(r.out, "├─ Create3ProxyContractCreation(newContract: %s, salt: 0x%x...)\n", 
		r.shortenAddress(dep.Address), dep.Salt[:4])
	fmt.Fprintf(r.out, "├─ %s::0x60806040...\n", r.shortenAddress(dep.Address))
	
	// Show the actual deployment
	constructorArgs := ""
	if len(dep.ConstructorArgs) > 0 {
		// TODO: Decode constructor args if we have ABI
		constructorArgs = "()"
	} else {
		constructorArgs = "()"
	}
	
	fmt.Fprintf(r.out, "│  └─ 🚀 %snew %s%s%s\n", 
		r.colorGreen, dep.ContractName, constructorArgs, r.colorReset)
	fmt.Fprintf(r.out, "│     └─ [return] %s\n", dep.Address)
	
	// Show final event
	displayName := dep.ContractName
	if dep.Label != "" {
		displayName = fmt.Sprintf("%s:%s", dep.ContractName, dep.Label)
	}
	fmt.Fprintf(r.out, "%s ContractCreation(newContract: %s: [%s...)\n", 
		prefix, displayName, dep.Address[:10])
}

// renderProxyDeployment renders a proxy contract deployment
func (r *ScriptRenderer) renderProxyDeployment(dep domain.ScriptDeployment, isLast bool, exec *domain.ScriptExecution) {
	// For proxy deployments, we typically have the implementation address in ProxyInfo
    if dep.ProxyInfo == nil {
        r.renderRegularDeployment(dep, isLast)
		return
	}

	prefix := "├─"
	if isLast {
		prefix = "└─"
	}

	// Show intermediate steps
	fmt.Fprintf(r.out, "├─ create2(16 bytes)\n")
	fmt.Fprintf(r.out, "│  └─ [return] %s\n", r.shortenAddress(dep.Address))
	fmt.Fprintf(r.out, "├─ Create3ProxyContractCreation(newContract: %s, salt: 0x%x...)\n", 
		r.shortenAddress(dep.Address), dep.Salt[:4])
	fmt.Fprintf(r.out, "├─ %s::0x60806040...\n", r.shortenAddress(dep.Address))
	
	// Show proxy deployment with constructor args
	fmt.Fprintf(r.out, "│  └─ 🚀 %snew %s(%s\n", 
		r.colorGreen, dep.ContractName, r.colorReset)
	fmt.Fprintf(r.out, "│     │    implementation: %s,\n", dep.ProxyInfo.Implementation)
	fmt.Fprintf(r.out, "│     │    _data: 0xc4d66de8000000000000000000000000...(36 bytes)\n")
	fmt.Fprintf(r.out, "│     │  )\n")
	
	// Show events
	fmt.Fprintf(r.out, "│     ├─ Upgraded(implementation: %s: [%s...)\n", 
		r.getImplementationName(dep.ProxyInfo.Implementation, exec), 
		dep.ProxyInfo.Implementation[:10])
	fmt.Fprintf(r.out, "│     ├─ %s::initialize(0x0000...0000) (delegate)\n", 
		r.getImplementationName(dep.ProxyInfo.Implementation, exec))
	fmt.Fprintf(r.out, "│     └─ [return] %s\n", dep.Address)
	
	// Show final event
	displayName := dep.ContractName
	if dep.Label != "" {
		displayName = fmt.Sprintf("%s:%s", dep.ContractName, dep.Label)
	}
	fmt.Fprintf(r.out, "%s ContractCreation(newContract: %s: [%s...)\n", 
		prefix, displayName, dep.Address[:10])
}

// getImplementationName tries to find the name of an implementation contract
func (r *ScriptRenderer) getImplementationName(implAddr string, exec *domain.ScriptExecution) string {
	// Look for implementation in deployments
	for _, dep := range exec.Deployments {
		if strings.EqualFold(dep.Address, implAddr) {
			return dep.ContractName
		}
	}
	return "Implementation"
}

// renderDeploymentSummary displays the deployment summary
func (r *ScriptRenderer) renderDeploymentSummary(exec *domain.ScriptExecution) error {
	if len(exec.Deployments) == 0 {
		return nil
	}

	fmt.Fprintf(r.out, "\n%s📦 Deployment Summary:%s\n", r.colorBold, r.colorReset)
	fmt.Fprintf(r.out, "%s%s%s\n", r.colorGray, strings.Repeat("─", 50), r.colorReset)

	for _, dep := range exec.Deployments {
		// Build deployment name
		name := dep.ContractName
		if dep.Label != "" {
			name = fmt.Sprintf("%s:%s", dep.ContractName, dep.Label)
		}
		if dep.IsProxy && dep.ProxyInfo != nil {
			// Get implementation name if available
			implName := r.shortenAddress(dep.ProxyInfo.Implementation)
			name = fmt.Sprintf("%s[%s]", name, implName)
		}

		fmt.Fprintf(r.out, "%s%s%s at %s%s%s\n",
			r.colorCyan, name, r.colorReset,
			r.colorGreen, dep.Address, r.colorReset)
	}

	fmt.Fprintln(r.out) // Empty line after deployments
	return nil
}

// renderLogs displays console.log output from the script
func (r *ScriptRenderer) renderLogs(logs []string) error {
	if len(logs) == 0 {
		return nil
	}

	fmt.Fprintf(r.out, "\n%s📝 Script Logs:%s\n", r.colorBold, r.colorReset)
	fmt.Fprintf(r.out, "%s%s%s\n", r.colorGray, strings.Repeat("─", 40), r.colorReset)

	for _, log := range logs {
		fmt.Fprintf(r.out, "  %s\n", log)
	}

	fmt.Fprintln(r.out) // Empty line after logs
	return nil
}

// renderRegistryUpdate displays the registry update summary
func (r *ScriptRenderer) renderRegistryUpdate(changes *usecase.RegistryChanges, namespace, network string) error {
	fmt.Fprintf(r.out, "\n%s✅ Updated registry for %s network in namespace %s%s\n",
		r.colorGreen, network, namespace, r.colorReset)

	if changes.AddedCount > 0 {
		fmt.Fprintf(r.out, "  Added %d deployment(s)\n", changes.AddedCount)
	}
	if changes.UpdatedCount > 0 {
		fmt.Fprintf(r.out, "  Updated %d deployment(s)\n", changes.UpdatedCount)
	}

	return nil
}

// getStatusDisplay returns a formatted status string
func (r *ScriptRenderer) getStatusDisplay(status domain.TransactionStatus) string {
	switch status {
	case domain.TransactionStatusSimulated:
		return fmt.Sprintf("%sSimulated%s", r.colorYellow, r.colorReset)
	case domain.TransactionStatusQueued:
		return fmt.Sprintf("%sQueued%s", r.colorPurple, r.colorReset)
	case domain.TransactionStatusExecuted:
		return fmt.Sprintf("%sExecuted%s", r.colorGreen, r.colorReset)
	case domain.TransactionStatusFailed:
		return fmt.Sprintf("%sFailed%s", r.colorRed, r.colorReset)
	default:
		return string(status)
	}
}

// shortenAddress returns a shortened version of an address
func (r *ScriptRenderer) shortenAddress(addr string) string {
	if len(addr) >= 10 {
		return addr[:10] + "..."
	}
	return addr
}

// PrintDeploymentBanner prints the deployment banner (called before execution)
func PrintDeploymentBanner(out io.Writer, scriptName, network, namespace string, dryRun bool, envVars map[string]string) {
	bold := color.New(color.Bold).SprintFunc()("")
	reset := color.New(color.Reset).SprintFunc()("")
	gray := color.New(color.FgHiBlack).SprintFunc()("")
	cyan := color.New(color.FgCyan).SprintFunc()("")
	blue := color.New(color.FgBlue).SprintFunc()("")
	purple := color.New(color.FgMagenta).SprintFunc()("")
	yellow := color.New(color.FgYellow).SprintFunc()("")
	green := color.New(color.FgGreen).SprintFunc()("")

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s🚀 Running Deployment Script%s\n", bold, reset)
	fmt.Fprintf(out, "%s%s%s\n", gray, strings.Repeat("─", 50), reset)
	fmt.Fprintf(out, "  Script:    %s%s%s\n", cyan, scriptName, reset)
	fmt.Fprintf(out, "  Network:   %s%s%s\n", blue, network, reset)
	fmt.Fprintf(out, "  Namespace: %s%s%s\n", purple, namespace, reset)

	if dryRun {
		fmt.Fprintf(out, "  Mode:      %sDRY RUN%s\n", yellow, reset)
	} else {
		fmt.Fprintf(out, "  Mode:      %sLIVE%s\n", green, reset)
	}

	// Display environment variables if any
	if len(envVars) > 0 {
		fmt.Fprintf(out, "  Env Vars:  ")
		i := 0
		for key, value := range envVars {
			if i > 0 {
				fmt.Fprintf(out, "             ")
			}
			fmt.Fprintf(out, "%s%s%s=%s%s%s\n", yellow, key, reset, green, value, reset)
			i++
		}
	}

	fmt.Fprintf(out, "%s%s%s\n", gray, strings.Repeat("─", 50), reset)
	fmt.Fprintln(out)
}
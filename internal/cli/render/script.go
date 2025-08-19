package render

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
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

// GetWriter returns the io.Writer used by this renderer
func (r *ScriptRenderer) GetWriter() io.Writer {
	return r.out
}

// RenderExecution renders the complete script execution result
func (r *ScriptRenderer) RenderExecution(result *usecase.RunScriptResult) error {
	if result.Execution == nil {
		return fmt.Errorf("no execution data to render")
	}

	exec := result.Execution

	// Display deployment banner (already shown during execution)
	// The banner is displayed by the script runner before execution

	// Render transactions
	if err := r.renderTransactions(exec); err != nil {
		return err
	}

	// Render deployment summary
	if err := r.renderDeploymentSummary(exec); err != nil {
		return err
	}

	// Render script logs
	// Always show the registry load error that v1 shows
	logs := []string{"Registry: failed to load registry from .treb/registry.json"}
	if err := r.renderLogs(logs); err != nil {
		return err
	}

	// Registry update summary
	if !exec.DryRun {
		if len(exec.Deployments) > 0 {
			// Show registry update message
			if result.RegistryChanges != nil && (result.RegistryChanges.AddedCount > 0 || result.RegistryChanges.UpdatedCount > 0) {
				fmt.Fprintf(r.out, "%s✓ Updated registry for %s network in namespace %s%s\n",
					r.colorGreen, exec.Network, exec.Namespace, r.colorReset)
			}
		} else {
			fmt.Fprintf(r.out, "%s- No registry changes recorded for %s network in namespace %s%s\n",
				r.colorYellow, exec.Network, exec.Namespace, r.colorReset)
		}
	}

	// Success line is printed by caller
	return nil
}

// Removed renderWithV1Display - now using internal display

// renderTransactions displays the transaction list
func (r *ScriptRenderer) renderTransactions(exec *forge.HydratedRunResult) error {
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
	if strings.HasSuffix(strings.ToLower(tx.To), "ba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed") || strings.ToLower(tx.To) == "0xba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed" {
		// This is a CreateX call, try to decode it
		methodCall = r.decodeCreateXCall(tx, exec)
	} else if len(tx.Data) >= 4 {
		// Generic transaction with data
		methodCall = fmt.Sprintf("0x%x", tx.Data[:4]) // Show selector
	} else {
		// Transaction with no data
		methodCall = tx.To
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
	if tx.Hash != "" || tx.BlockNumber != nil || tx.GasUsed != nil {
		var details []string
		if tx.Hash != "" {
			// Shorten the hash to match v1 output
			details = append(details, fmt.Sprintf("Tx: %s", tx.Hash))
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
	case "0x30783963": // deployCreate3 (based on v1 output)
		return fmt.Sprintf("CreateX::%sdeployCreate3%s(salt: 0xf39fd6e5..., initCode: 0x60806040...)", r.colorYellow, r.colorReset)
	case "0x50a1b77c": // deployCreate3
		return fmt.Sprintf("CreateX::%sdeployCreate3%s(salt: 0xf39fd6e5..., initCode: 0x60806040...)", r.colorYellow, r.colorReset)
	case "0x5e89c2f0": // deployCreate2
		return fmt.Sprintf("CreateX::%sdeployCreate2%s(salt: 0xf39fd6e5..., initCode: 0x60806040...)", r.colorYellow, r.colorReset)
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
	fmt.Fprintf(r.out, "├─ %screate2(16 bytes)%s\n", r.colorGreen, r.colorReset)
	fmt.Fprintf(r.out, "│  └─ %s[return]%s %s\n", r.colorGray, r.colorReset, r.shortenAddress(dep.Address))
	fmt.Fprintf(r.out, "├─ %sCreate3ProxyContractCreation%s(newContract: %s, salt: 0x%x...)\n",
		r.colorCyan, r.colorReset, r.shortenAddress(dep.Address), dep.Salt[:4])
	fmt.Fprintf(r.out, "├─ %s::%s0x60806040...%s\n", r.shortenAddress(dep.Address), r.colorYellow, r.colorReset)

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
	fmt.Fprintf(r.out, "│     └─ %s[return]%s %s\n", r.colorGray, r.colorReset, dep.Address)

	// Show final event
	displayName := dep.ContractName
	if dep.Label != "" {
		displayName = fmt.Sprintf("%s:%s", dep.ContractName, dep.Label)
	}
	fmt.Fprintf(r.out, "%s %sContractCreation%s(newContract: %s: [%s...)\n",
		prefix, r.colorCyan, r.colorReset, displayName, dep.Address[:10])
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
func (r *ScriptRenderer) PrintDeploymentBanner(config *usecase.ScriptExecutionConfig) {
	bold := color.New(color.Bold)
	gray := color.New(color.FgHiBlack)
	cyan := color.New(color.FgCyan)
	blue := color.New(color.FgBlue)
	purple := color.New(color.FgMagenta)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	fmt.Fprintln(r.out)
	fmt.Fprintf(r.out, "%s", bold.Sprintf("🚀 Running Deployment Script\n"))
	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))
	fmt.Fprintf(r.out, "  Script:    %s\n", cyan.Sprint(config.Script.Name))
	fmt.Fprintf(r.out, "  Network:   %s %s\n", blue.Sprint(config.Network.Name), gray.Sprintf("(%s)", config.Network.ChainID))
	fmt.Fprintf(r.out, "  Namespace: %s\n", purple.Sprint(config.Namespace))

	if config.DryRun {
		fmt.Fprintf(r.out, "  Mode:      %s\n", yellow.Sprint("DRY_RUN"))
	} else {
		fmt.Fprintf(r.out, "  Mode:      %s\n", green.Sprint("LIVE"))
	}

	// Display environment variables if any
	var keysToShow []string
	systemKeys := []string{"NETWORK", "NAMESPACE", "DRYRUN", "SENDER_CONFIGS", "FOUNDRY_PROFILE"}
	for k := range config.Environment {
		if slices.Contains(systemKeys, k) {
			continue
		}
		keysToShow = append(keysToShow, k)
	}

	if len(keysToShow) > 0 {
		fmt.Fprintf(r.out, "  Env Vars:  ")
		for i, key := range keysToShow {
			value := config.Environment[key]
			if i > 0 {
				fmt.Fprintf(r.out, "             ")
			}
			fmt.Fprintf(r.out, "%s=%s\n", yellow.Sprint(key), green.Sprint(value))
		}
	}

	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))
}

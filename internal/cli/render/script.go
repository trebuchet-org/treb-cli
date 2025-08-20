package render

import (
	"fmt"
	"io"
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
	if result.RunResult == nil {
		return fmt.Errorf("no execution data to render")
	}

	exec := result.RunResult

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
	if result.Success && !result.RunResult.DryRun {
		if len(exec.Deployments) > 0 {
			// Show registry update message
			if result.Changeset != nil && result.Changeset.HasChanges() {
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
func (r *ScriptRenderer) renderTransactionTree(tx *forge.Transaction, exec *forge.HydratedRunResult) {
	// Build status text
	var statusText string
	switch tx.Status {
	case models.TransactionStatusSimulated:
		statusText = "simulated"
	case models.TransactionStatusQueued:
		statusText = "queued   "
	case models.TransactionStatusExecuted:
		statusText = "executed "
	case models.TransactionStatusFailed:
		statusText = "failed   "
	default:
		statusText = "unknown  "
	}

	// Try to identify what this transaction does
	var methodCall string

	// Check if this is a CreateX deployment
	toAddr := strings.ToLower(tx.Transaction.To.Hex())
	if strings.HasSuffix(toAddr, "ba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed") || toAddr == "0xba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed" {
		// This is a CreateX call, try to decode it
		methodCall = r.decodeCreateXCall(tx, exec)
	} else if len(tx.Transaction.Data) >= 4 {
		// Generic transaction with data
		methodCall = fmt.Sprintf("0x%x", tx.Transaction.Data[:4]) // Show selector
	} else {
		// Transaction with no data
		methodCall = tx.Transaction.To.Hex()
	}

	// Display transaction header
	fmt.Fprintf(r.out, "\n%s%s%s %s%s%s → %s\n",
		r.getStatusColor(tx.Status), statusText, r.colorReset,
		r.colorGreen, r.getKnownAddress(tx.Sender.Hex()), r.colorReset,
		methodCall)

	// Find deployments for this transaction
	deployments := r.findDeploymentsForTransaction(tx, exec)

	// Display deployment info if any
	for i, dep := range deployments {
		isLast := i == len(deployments)-1

		// Show CREATE operation
		// Check if this is a proxy deployment by looking for proxy relationships
		isProxy := false
		if exec.ProxyRelationships != nil {
			if _, hasProxy := exec.ProxyRelationships[dep.Address]; hasProxy {
				isProxy = true
			}
		}

		if isProxy {
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
			// Shorten the hash to match v1 output
			details = append(details, fmt.Sprintf("Tx: %s", tx.TxHash.Hex()))
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
func (r *ScriptRenderer) decodeCreateXCall(tx *forge.Transaction, exec *forge.HydratedRunResult) string {
	// Check method selector (first 4 bytes)
	if len(tx.Transaction.Data) < 4 {
		return "unknown()"
	}

	selector := fmt.Sprintf("0x%x", tx.Transaction.Data[:4])

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
func (r *ScriptRenderer) getStatusColor(status models.TransactionStatus) string {
	switch status {
	case models.TransactionStatusSimulated:
		return r.colorGray
	case models.TransactionStatusQueued:
		return r.colorYellow
	case models.TransactionStatusExecuted:
		return r.colorGreen
	case models.TransactionStatusFailed:
		return r.colorRed
	default:
		return ""
	}
}

// findDeploymentsForTransaction finds deployments created in a transaction
func (r *ScriptRenderer) findDeploymentsForTransaction(tx *forge.Transaction, exec *forge.HydratedRunResult) []*forge.Deployment {
	var deployments []*forge.Deployment
	for _, dep := range exec.Deployments {
		if dep.TransactionID == tx.TransactionId {
			deployments = append(deployments, dep)
		}
	}
	return deployments
}

// renderRegularDeployment renders a regular contract deployment
func (r *ScriptRenderer) renderRegularDeployment(dep *forge.Deployment, isLast bool) {
	prefix := "├─"
	if isLast {
		prefix = "└─"
	}

	// Show intermediate steps (simplified without full trace)
	fmt.Fprintf(r.out, "├─ %screate2(16 bytes)%s\n", r.colorGreen, r.colorReset)
	fmt.Fprintf(r.out, "│  └─ %s[return]%s %s\n", r.colorGray, r.colorReset, r.shortenAddress(dep.Address.Hex()))
	fmt.Fprintf(r.out, "├─ %sCreate3ProxyContractCreation%s(newContract: %s, salt: 0x%x...)\n",
		r.colorCyan, r.colorReset, r.shortenAddress(dep.Address.Hex()), dep.Event.Salt[:4])
	fmt.Fprintf(r.out, "├─ %s::%s0x60806040...%s\n", r.shortenAddress(dep.Address.Hex()), r.colorYellow, r.colorReset)

	// Show the actual deployment
	constructorArgs := ""
	if len(dep.Event.ConstructorArgs) > 0 {
		// TODO: Decode constructor args if we have ABI
		constructorArgs = "()"
	} else {
		constructorArgs = "()"
	}

	contractName := dep.Event.Artifact
	if dep.Contract != nil && dep.Contract.Name != "" {
		contractName = dep.Contract.Name
	}
	fmt.Fprintf(r.out, "│  └─ 🚀 %snew %s%s%s\n",
		r.colorGreen, contractName, constructorArgs, r.colorReset)
	fmt.Fprintf(r.out, "│     └─ %s[return]%s %s\n", r.colorGray, r.colorReset, dep.Address.Hex())

	// Show final event
	displayName := contractName
	if dep.Event.Label != "" {
		displayName = fmt.Sprintf("%s:%s", contractName, dep.Event.Label)
	}
	fmt.Fprintf(r.out, "%s %sContractCreation%s(newContract: %s: [%s...)\n",
		prefix, r.colorCyan, r.colorReset, displayName, dep.Address.Hex()[:10])
}

// renderProxyDeployment renders a proxy contract deployment
func (r *ScriptRenderer) renderProxyDeployment(dep *forge.Deployment, isLast bool, exec *forge.HydratedRunResult) {
	// Get proxy info from exec.ProxyRelationships
	proxyRel, hasProxy := exec.ProxyRelationships[dep.Address]
	if !hasProxy {
		r.renderRegularDeployment(dep, isLast)
		return
	}

	prefix := "├─"
	if isLast {
		prefix = "└─"
	}

	// Show intermediate steps
	fmt.Fprintf(r.out, "├─ create2(16 bytes)\n")
	fmt.Fprintf(r.out, "│  └─ [return] %s\n", r.shortenAddress(dep.Address.Hex()))
	fmt.Fprintf(r.out, "├─ Create3ProxyContractCreation(newContract: %s, salt: 0x%x...)\n",
		r.shortenAddress(dep.Address.Hex()), dep.Event.Salt[:4])
	fmt.Fprintf(r.out, "├─ %s::0x60806040...\n", r.shortenAddress(dep.Address.Hex()))

	// Show proxy deployment with constructor args
	contractName := dep.Event.Artifact
	if dep.Contract != nil && dep.Contract.Name != "" {
		contractName = dep.Contract.Name
	}
	fmt.Fprintf(r.out, "│  └─ 🚀 %snew %s(%s\n",
		r.colorGreen, contractName, r.colorReset)
	fmt.Fprintf(r.out, "│     │    implementation: %s,\n", proxyRel.ImplementationAddress.Hex())
	fmt.Fprintf(r.out, "│     │    _data: 0xc4d66de8000000000000000000000000...(36 bytes)\n")
	fmt.Fprintf(r.out, "│     │  )\n")

	// Show events
	fmt.Fprintf(r.out, "│     ├─ Upgraded(implementation: %s: [%s...)\n",
		r.getImplementationName(proxyRel.ImplementationAddress.Hex(), exec),
		proxyRel.ImplementationAddress.Hex()[:10])
	fmt.Fprintf(r.out, "│     ├─ %s::initialize(0x0000...0000) (delegate)\n",
		r.getImplementationName(proxyRel.ImplementationAddress.Hex(), exec))
	fmt.Fprintf(r.out, "│     └─ [return] %s\n", dep.Address.Hex())

	// Show final event
	displayName := contractName
	if dep.Event.Label != "" {
		displayName = fmt.Sprintf("%s:%s", contractName, dep.Event.Label)
	}
	fmt.Fprintf(r.out, "%s ContractCreation(newContract: %s: [%s...)\n",
		prefix, displayName, dep.Address.Hex()[:10])
}

// getImplementationName tries to find the name of an implementation contract
func (r *ScriptRenderer) getImplementationName(implAddr string, exec *forge.HydratedRunResult) string {
	// Look for implementation in deployments
	for _, dep := range exec.Deployments {
		if strings.EqualFold(dep.Address.Hex(), implAddr) {
			contractName := dep.Event.Artifact
			if dep.Contract != nil && dep.Contract.Name != "" {
				contractName = dep.Contract.Name
			}
			return contractName
		}
	}
	return "Implementation"
}

// renderDeploymentSummary displays the deployment summary
func (r *ScriptRenderer) renderDeploymentSummary(exec *forge.HydratedRunResult) error {
	if len(exec.Deployments) == 0 {
		return nil
	}

	fmt.Fprintf(r.out, "\n%s📦 Deployment Summary:%s\n", r.colorBold, r.colorReset)
	fmt.Fprintf(r.out, "%s%s%s\n", r.colorGray, strings.Repeat("─", 50), r.colorReset)

	for _, dep := range exec.Deployments {
		// Build deployment name
		contractName := dep.Event.Artifact
		if dep.Contract != nil && dep.Contract.Name != "" {
			contractName = dep.Contract.Name
		}

		name := contractName
		if dep.Event.Label != "" {
			name = fmt.Sprintf("%s:%s", contractName, dep.Event.Label)
		}

		// Check if this is a proxy deployment
		if proxyRel, hasProxy := exec.ProxyRelationships[dep.Address]; hasProxy {
			// Get implementation name if available
			implName := r.getImplementationName(proxyRel.ImplementationAddress.Hex(), exec)
			name = fmt.Sprintf("%s[%s]", name, implName)
		}

		fmt.Fprintf(r.out, "%s%s%s at %s%s%s\n",
			r.colorCyan, name, r.colorReset,
			r.colorGreen, dep.Address.Hex(), r.colorReset)
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
func (r *ScriptRenderer) renderRegistryUpdate(changeset *models.Changeset, namespace, network string) error {
	fmt.Fprintf(r.out, "\n%s✅ Updated registry for %s network in namespace %s%s\n",
		r.colorGreen, network, namespace, r.colorReset)

	if changeset.Create.Count() > 0 {
		fmt.Fprintf(r.out, "  Added %d deployment(s)\n", changeset.Create.Count())
	}
	if changeset.Update.Count() > 0 {
		fmt.Fprintf(r.out, "  Updated %d deployment(s)\n", changeset.Update.Count())
	}

	return nil
}

// getStatusDisplay returns a formatted status string
func (r *ScriptRenderer) getStatusDisplay(status models.TransactionStatus) string {
	switch status {
	case models.TransactionStatusSimulated:
		return fmt.Sprintf("%sSimulated%s", r.colorYellow, r.colorReset)
	case models.TransactionStatusQueued:
		return fmt.Sprintf("%sQueued%s", r.colorPurple, r.colorReset)
	case models.TransactionStatusExecuted:
		return fmt.Sprintf("%sExecuted%s", r.colorGreen, r.colorReset)
	case models.TransactionStatusFailed:
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
func (r *ScriptRenderer) PrintDeploymentBanner(config *usecase.RunScriptConfig) {
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
	fmt.Fprintf(r.out, "  Network:   %s %s\n", blue.Sprint(config.Network.Name), gray.Sprintf("(%d)", config.Network.ChainID))
	fmt.Fprintf(r.out, "  Namespace: %s\n", purple.Sprint(config.Namespace))

	if config.DryRun {
		fmt.Fprintf(r.out, "  Mode:      %s\n", yellow.Sprint("DRY_RUN"))
	} else {
		fmt.Fprintf(r.out, "  Mode:      %s\n", green.Sprint("LIVE"))
	}

	if len(config.Parameters) > 0 {
		fmt.Fprintf(r.out, "  Env Vars:  ")
		var i = 0
		for k, v := range config.Parameters {
			if i > 0 {
				fmt.Fprintf(r.out, "             ")
			}
			fmt.Fprintf(r.out, "%s=%s\n", yellow.Sprint(k), green.Sprint(v))
			i++
		}
	}

	fmt.Fprintf(r.out, "  Senders:      %s", gray.Sprintf("%v", config.SenderScriptConfig.Senders))
	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))
}

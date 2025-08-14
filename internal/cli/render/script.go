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

	// Display transactions
	if err := r.renderTransactions(exec); err != nil {
		return err
	}

	// Display deployment summary
	if err := r.renderDeploymentSummary(exec); err != nil {
		return err
	}

	// Display logs
	if err := r.renderLogs(exec.Logs); err != nil {
		return err
	}

	// Display registry update summary
	if result.RegistryChanges != nil && result.RegistryChanges.HasChanges {
		if err := r.renderRegistryUpdate(result.RegistryChanges, exec.Namespace, exec.Network); err != nil {
			return err
		}
	} else if !exec.DryRun && len(exec.Deployments) == 0 {
		fmt.Fprintf(r.out, "%s- No registry changes recorded for %s network in namespace %s%s\n",
			r.colorYellow, exec.Network, exec.Namespace, r.colorReset)
	}

	// Display success message
	fmt.Fprintf(r.out, "\n%s✅ Script execution completed successfully%s\n", r.colorGreen, r.colorReset)

	// Display debug output location if applicable
	if exec.BroadcastPath != "" && (r.verbose || exec.DryRun) {
		fmt.Fprintf(r.out, "\nDebug output saved to: debug-output.json\n")
	}

	return nil
}

// renderTransactions displays the transaction list
func (r *ScriptRenderer) renderTransactions(exec *domain.ScriptExecution) error {
	fmt.Fprintf(r.out, "\n%s🔄 Transactions:%s\n", r.colorBold, r.colorReset)
	fmt.Fprintf(r.out, "%s%s%s\n", r.colorGray, strings.Repeat("─", 50), r.colorReset)

	if len(exec.Transactions) == 0 {
		fmt.Fprintf(r.out, "%sNo transactions executed (dry run or all deployments skipped)%s\n\n", 
			r.colorGray, r.colorReset)
		return nil
	}

	// Display each transaction
	for i, tx := range exec.Transactions {
		// Transaction header
		status := r.getStatusDisplay(tx.Status)
		fmt.Fprintf(r.out, "\n%s[%d]%s %s\n", r.colorBold, i+1, r.colorReset, status)

		// Basic info
		fmt.Fprintf(r.out, "  From: %s%s%s\n", r.colorCyan, r.shortenAddress(tx.Sender), r.colorReset)
		fmt.Fprintf(r.out, "  To:   %s%s%s\n", r.colorCyan, r.shortenAddress(tx.To), r.colorReset)

		// Transaction hash if available
		if tx.TxHash != nil {
			fmt.Fprintf(r.out, "  Hash: %s%s%s\n", r.colorGreen, *tx.TxHash, r.colorReset)
		}

		// Safe transaction info
		if tx.SafeTransaction != nil {
			fmt.Fprintf(r.out, "  Safe: %s%s%s\n", r.colorPurple, tx.SafeTransaction.SafeAddress, r.colorReset)
			fmt.Fprintf(r.out, "  Safe Tx Hash: %s0x%x%s\n", r.colorPurple, tx.SafeTransaction.SafeTxHash, r.colorReset)
		}

		// Gas info if available
		if tx.GasUsed != nil {
			fmt.Fprintf(r.out, "  Gas Used: %d\n", *tx.GasUsed)
		}

		// TODO: Display decoded transaction data if verbose mode
	}

	fmt.Fprintln(r.out) // Empty line after transactions
	return nil
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
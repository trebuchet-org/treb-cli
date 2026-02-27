package render

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

var (
	bold   = color.New(color.Bold)
	gray   = color.New(color.FgHiBlack)
	cyan   = color.New(color.FgCyan)
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
)

// ScriptRenderer renders script execution results
type ScriptRenderer struct {
	out             io.Writer
	deploymentsRepo usecase.DeploymentRepository
	txRenderer      *TransactionRenderer
	log             *slog.Logger
}

// NewScriptRenderer creates a new script renderer
func NewScriptRenderer(out io.Writer, deploymentsRepo usecase.DeploymentRepository, abiResolver usecase.ABIResolver, log *slog.Logger) *ScriptRenderer {
	return &ScriptRenderer{
		out:             out,
		deploymentsRepo: deploymentsRepo,
		log:             log.With("component", "ScriptRenderer"),
		txRenderer: NewTransactionRenderer(
			abiResolver,
			deploymentsRepo,
			log,
		),
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

	// Render transactions
	if err := r.renderTransactions(exec); err != nil {
		return err
	}

	// Render deployment summary
	if err := r.renderDeploymentSummary(exec); err != nil {
		return err
	}

	// Render script logs
	if err := r.renderLogs(exec); err != nil {
		return err
	}

	// Registry update summary
	if result.Success && !result.RunResult.DryRun {
		if len(exec.Deployments) > 0 {
			// Show registry update message
			if result.Changeset != nil && result.Changeset.HasChanges() {
				fmt.Fprint(r.out, green.Sprintf("✓ Updated registry for %s network in namespace %s\n",
					exec.Network, exec.Namespace))
			}
		} else {
			fmt.Fprint(r.out, yellow.Sprintf("- No registry changes recorded for %s network in namespace %s\n",
				exec.Network, exec.Namespace))
		}
	}

	// Success line is printed by caller
	return nil
}

// Removed renderWithV1Display - now using internal display

// renderTransactions displays the transaction list
func (r *ScriptRenderer) renderTransactions(exec *forge.HydratedRunResult) error {
	fmt.Fprintf(r.out, "\n%s\n", bold.Sprint("🔄 Transactions:"))
	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))

	if len(exec.Transactions) == 0 {
		fmt.Fprintf(r.out, "%s\n\n",
			gray.Sprint("No transactions executed (dry run or all deployments skipped)"))
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
	// getImplementationName tries to find the name of an implementation contract
	r.txRenderer.WithExecution(exec).DisplayTransactionWithEvents(tx)
}

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

	deployment, err := r.deploymentsRepo.GetDeploymentByAddress(context.Background(), exec.ChainID, implAddr)
	if err == nil {
		return deployment.ContractName
	}

	return "UnknownImplementation"
}

// renderDeploymentSummary displays the deployment summary
func (r *ScriptRenderer) renderDeploymentSummary(exec *forge.HydratedRunResult) error {
	if len(exec.Collisions) > 0 {
		fmt.Fprintf(r.out, "\n%s\n", yellow.Sprint("⚠️ Deployment Collisions Detected:"))
		fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))

		for address, collision := range exec.Collisions {
			contractName := extractContractName(collision.DeploymentDetails.Artifact)
			fmt.Fprintf(r.out, "%s already deployed at %s\n",
				cyan.Sprint(contractName),
				yellow.Sprint(address.Hex()))

			// Show details if verbose
			if collision.DeploymentDetails.Label != "" {
				fmt.Fprintf(r.out, "    Label: %s\n", collision.DeploymentDetails.Label)
			}
			if collision.DeploymentDetails.Entropy != "" {
				fmt.Fprintf(r.out, "    Entropy: %s\n", collision.DeploymentDetails.Entropy)
			}
		}

		fmt.Fprintf(r.out, "%s\n\n",
			gray.Sprint("Note: These contracts were already deployed and deployment was skipped."))
	}
	if len(exec.Deployments) == 0 {
		return nil
	}

	fmt.Fprintf(r.out, "\n%s\n", bold.Sprint("📦 Deployment Summary:"))
	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))

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

		fmt.Fprintf(r.out, "%s at %s\n",
			cyan.Sprint(name),
			green.Sprint(dep.Address.Hex()))
	}

	fmt.Fprintln(r.out) // Empty line after deployments
	return nil
}

// renderLogs displays console.log output from the script
func (r *ScriptRenderer) renderLogs(exec *forge.HydratedRunResult) error {
	if exec.ParsedOutput == nil {
		return nil
	}
	logs := exec.ParsedOutput.ConsoleLogs
	if len(logs) == 0 {
		return nil
	}

	fmt.Fprintf(r.out, "\n%s\n", bold.Sprint("📝 Script Logs:"))
	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 40)))

	for _, log := range logs {
		fmt.Fprintf(r.out, "  %s\n", log)
	}

	fmt.Fprintln(r.out) // Empty line after logs
	return nil
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
	} else if len(config.ForkEnvOverrides) > 0 {
		fmt.Fprintf(r.out, "  Mode:      %s\n", purple.Sprint("FORK"))
	} else {
		fmt.Fprintf(r.out, "  Mode:      %s\n", green.Sprint("LIVE"))
	}

	if len(config.Parameters) > 0 {
		keys := make([]string, 0, len(config.Parameters))
		for k := range config.Parameters {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Fprintf(r.out, "  Env Vars:  ")
		for i, param := range keys {
			if i > 0 {
				fmt.Fprintf(r.out, "             ")
			}
			fmt.Fprintf(r.out, "%s=%s\n", yellow.Sprint(param), green.Sprint(config.Parameters[param]))
		}
	}

	fmt.Fprintf(r.out, "  Senders:   %s\n", gray.Sprintf("%v", config.SenderScriptConfig.Senders))
	fmt.Fprintf(r.out, "%s\n", gray.Sprint(strings.Repeat("─", 50)))
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

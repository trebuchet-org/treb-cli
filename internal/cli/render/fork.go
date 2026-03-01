package render

import (
	"fmt"
	"io"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ForkRenderer handles rendering of fork command results
type ForkRenderer struct {
	out io.Writer
}

// NewForkRenderer creates a new ForkRenderer
func NewForkRenderer(out io.Writer) *ForkRenderer {
	return &ForkRenderer{out: out}
}

// RenderEnter renders the result of fork enter
func (r *ForkRenderer) RenderEnter(result *usecase.EnterForkResult) error {
	entry := result.ForkEntry

	fmt.Fprintln(r.out, result.Message)
	fmt.Fprintln(r.out)
	fmt.Fprintf(r.out, "  Network:      %s\n", entry.Network)
	fmt.Fprintf(r.out, "  Chain ID:     %d\n", entry.ChainID)
	fmt.Fprintf(r.out, "  Fork URL:     %s\n", entry.ForkURL)
	if entry.External {
		fmt.Fprintf(r.out, "  Mode:         external\n")
	} else {
		fmt.Fprintf(r.out, "  Anvil PID:    %d\n", entry.AnvilPID)
	}
	fmt.Fprintf(r.out, "  Env Override: %s=%s\n", entry.EnvVarName, entry.ForkURL)
	if entry.LogFile != "" {
		fmt.Fprintf(r.out, "  Logs:         %s\n", entry.LogFile)
	}
	if result.SetupScriptRan {
		fmt.Fprintf(r.out, "  Setup:        executed successfully\n")
	}
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "Run 'treb fork status' to check fork state")
	fmt.Fprintln(r.out, "Run 'treb fork exit' to stop fork and restore original state")

	return nil
}

// RenderExit renders the result of fork exit
func (r *ForkRenderer) RenderExit(result *usecase.ExitForkResult) error {
	fmt.Fprintln(r.out, result.Message)
	fmt.Fprintln(r.out)
	for _, network := range result.ExitedNetworks {
		fmt.Fprintf(r.out, "  - %s: registry restored, fork cleaned up\n", network)
	}
	return nil
}

// RenderStatus renders the result of fork status
func (r *ForkRenderer) RenderStatus(result *usecase.ForkStatusResult) error {
	if !result.HasForks {
		fmt.Fprintln(r.out, "No active forks")
		return nil
	}

	fmt.Fprintln(r.out, "Active Forks")
	fmt.Fprintln(r.out)

	for _, e := range result.Entries {
		currentMarker := ""
		if e.IsCurrent {
			currentMarker = " (current)"
		}

		externalLabel := ""
		if e.External {
			externalLabel = " [external]"
		}

		fmt.Fprintf(r.out, "  %s%s%s\n", e.Network, currentMarker, externalLabel)
		fmt.Fprintf(r.out, "    Chain ID:     %d\n", e.ChainID)
		fmt.Fprintf(r.out, "    Fork URL:     %s\n", e.ForkURL)
		if !e.External {
			fmt.Fprintf(r.out, "    Anvil PID:    %d\n", e.AnvilPID)
		}
		fmt.Fprintf(r.out, "    Status:       %s\n", e.HealthDetail)
		if !e.External {
			fmt.Fprintf(r.out, "    Uptime:       %s\n", formatDuration(e.Uptime))
		}
		fmt.Fprintf(r.out, "    Snapshots:    %d\n", e.SnapshotCount)
		fmt.Fprintf(r.out, "    Fork Deploys: %d\n", e.ForkDeployments)
		if e.LogFile != "" {
			fmt.Fprintf(r.out, "    Logs:         %s\n", e.LogFile)
		}
		fmt.Fprintln(r.out)
	}

	return nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// RenderRevert renders the result of fork revert
func (r *ForkRenderer) RenderRevert(result *usecase.RevertForkResult) error {
	fmt.Fprintln(r.out, result.Message)
	fmt.Fprintln(r.out)
	if result.RevertedCommand != "" {
		fmt.Fprintf(r.out, "  Reverted:   %s\n", result.RevertedCommand)
	}
	fmt.Fprintf(r.out, "  Reverted:   %d snapshot(s)\n", result.RevertedCount)
	fmt.Fprintf(r.out, "  Remaining:  %d snapshot(s)\n", result.RemainingSnapshots)
	return nil
}

// RenderRestart renders the result of fork restart
func (r *ForkRenderer) RenderRestart(result *usecase.RestartForkResult) error {
	entry := result.ForkEntry

	fmt.Fprintln(r.out, result.Message)
	fmt.Fprintln(r.out)
	fmt.Fprintf(r.out, "  Network:      %s\n", entry.Network)
	fmt.Fprintf(r.out, "  Chain ID:     %d\n", entry.ChainID)
	fmt.Fprintf(r.out, "  Fork URL:     %s\n", entry.ForkURL)
	fmt.Fprintf(r.out, "  Anvil PID:    %d\n", entry.AnvilPID)
	fmt.Fprintf(r.out, "  Env Override: %s=%s\n", entry.EnvVarName, entry.ForkURL)
	fmt.Fprintf(r.out, "  Logs:         %s\n", entry.LogFile)
	if result.SetupScriptRan {
		fmt.Fprintf(r.out, "  Setup:        executed successfully\n")
	}
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "Registry restored to initial fork state. All previous snapshots cleared.")

	return nil
}

// RenderHistory renders the result of fork history
func (r *ForkRenderer) RenderHistory(result *usecase.ForkHistoryResult) error {
	fmt.Fprintf(r.out, "Fork History: %s\n", result.Network)
	fmt.Fprintln(r.out)

	for _, e := range result.Entries {
		marker := "  "
		if e.IsCurrent {
			marker = "→ "
		}

		label := ""
		if e.IsInitial {
			label = "initial"
		} else {
			label = e.Command
		}

		fmt.Fprintf(r.out, "  %s[%d] %s  (%s)\n", marker, e.Index, label, e.Timestamp)
	}

	fmt.Fprintln(r.out)
	return nil
}

// RenderDiff renders the result of fork diff
func (r *ForkRenderer) RenderDiff(result *usecase.ForkDiffResult) error {
	fmt.Fprintf(r.out, "Fork Diff: %s\n", result.Network)
	fmt.Fprintln(r.out)

	if !result.HasChanges {
		fmt.Fprintln(r.out, "No changes since fork entered.")
		return nil
	}

	if len(result.NewDeployments) > 0 {
		fmt.Fprintf(r.out, "New Deployments (%d):\n", len(result.NewDeployments))
		for _, dep := range result.NewDeployments {
			fmt.Fprintf(r.out, "  + %-20s %s  %s\n", dep.ContractName, dep.Address, dep.Type)
		}
		fmt.Fprintln(r.out)
	}

	if len(result.ModifiedDeployments) > 0 {
		fmt.Fprintf(r.out, "Modified Deployments (%d):\n", len(result.ModifiedDeployments))
		for _, dep := range result.ModifiedDeployments {
			fmt.Fprintf(r.out, "  ~ %-20s %s  %s\n", dep.ContractName, dep.Address, dep.Type)
		}
		fmt.Fprintln(r.out)
	}

	if result.NewTransactionCount > 0 {
		fmt.Fprintf(r.out, "New Transactions: %d\n", result.NewTransactionCount)
		fmt.Fprintln(r.out)
	}

	return nil
}

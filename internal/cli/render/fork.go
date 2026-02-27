package render

import (
	"fmt"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ForkRenderer handles rendering of fork command results
type ForkRenderer struct{}

// NewForkRenderer creates a new ForkRenderer
func NewForkRenderer() *ForkRenderer {
	return &ForkRenderer{}
}

// RenderEnter renders the result of fork enter
func (r *ForkRenderer) RenderEnter(result *usecase.EnterForkResult) error {
	entry := result.ForkEntry

	fmt.Println(result.Message)
	fmt.Println()
	fmt.Printf("  Network:      %s\n", entry.Network)
	fmt.Printf("  Chain ID:     %d\n", entry.ChainID)
	fmt.Printf("  Fork URL:     %s\n", entry.ForkURL)
	fmt.Printf("  Anvil PID:    %d\n", entry.AnvilPID)
	fmt.Printf("  Env Override: %s=%s\n", entry.EnvVarName, entry.ForkURL)
	if result.SetupScriptRan {
		fmt.Printf("  Setup:        executed successfully\n")
	}
	fmt.Println()
	fmt.Println("Run 'treb fork status' to check fork state")
	fmt.Println("Run 'treb fork exit' to stop fork and restore original state")

	return nil
}

// RenderExit renders the result of fork exit
func (r *ForkRenderer) RenderExit(result *usecase.ExitForkResult) error {
	fmt.Println(result.Message)
	fmt.Println()
	for _, network := range result.ExitedNetworks {
		fmt.Printf("  - %s: registry restored, fork cleaned up\n", network)
	}
	return nil
}

// RenderStatus renders the result of fork status
func (r *ForkRenderer) RenderStatus(result *usecase.ForkStatusResult) error {
	if !result.HasForks {
		fmt.Println("No active forks")
		return nil
	}

	fmt.Println("Active Forks")
	fmt.Println()

	for _, e := range result.Entries {
		currentMarker := ""
		if e.IsCurrent {
			currentMarker = " (current)"
		}

		fmt.Printf("  %s%s\n", e.Network, currentMarker)
		fmt.Printf("    Chain ID:     %d\n", e.ChainID)
		fmt.Printf("    Fork URL:     %s\n", e.ForkURL)
		fmt.Printf("    Anvil PID:    %d\n", e.AnvilPID)
		fmt.Printf("    Status:       %s\n", e.HealthDetail)
		fmt.Printf("    Uptime:       %s\n", formatDuration(e.Uptime))
		fmt.Printf("    Snapshots:    %d\n", e.SnapshotCount)
		fmt.Printf("    Fork Deploys: %d\n", e.ForkDeployments)
		fmt.Println()
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
	fmt.Println(result.Message)
	fmt.Println()
	if result.RevertedCommand != "" {
		fmt.Printf("  Reverted:   %s\n", result.RevertedCommand)
	}
	fmt.Printf("  Reverted:   %d snapshot(s)\n", result.RevertedCount)
	fmt.Printf("  Remaining:  %d snapshot(s)\n", result.RemainingSnapshots)
	return nil
}

// RenderHistory renders the result of fork history
func (r *ForkRenderer) RenderHistory(result *usecase.ForkHistoryResult) error {
	fmt.Printf("Fork History: %s\n", result.Network)
	fmt.Println()

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

		fmt.Printf("  %s[%d] %s  (%s)\n", marker, e.Index, label, e.Timestamp)
	}

	fmt.Println()
	return nil
}

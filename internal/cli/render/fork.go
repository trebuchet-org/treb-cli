package render

import (
	"fmt"

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

package forge

import (
	"context"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ForgeExecutorAdapter wraps the existing forge.Forge to implement ForgeExecutor
type ForgeExecutorAdapter struct {
	forge *forge.Forge
}

// NewForgeExecutorAdapter creates a new adapter wrapping the existing forge executor
func NewForgeExecutorAdapter(cfg *config.RuntimeConfig) (*ForgeExecutorAdapter, error) {
	f := forge.NewForge(cfg.ProjectRoot)
	
	// Check that forge is installed
	if err := f.CheckInstallation(); err != nil {
		return nil, err
	}
	
	return &ForgeExecutorAdapter{forge: f}, nil
}

// Build runs forge build
func (f *ForgeExecutorAdapter) Build(ctx context.Context) error {
	return f.forge.Build()
}

// RunScript executes a forge script
func (f *ForgeExecutorAdapter) RunScript(ctx context.Context, config usecase.ScriptConfig) (*usecase.ScriptResult, error) {
	// Convert config to forge flags
	var flags []string
	
	// Add network/RPC URL if specified
	if config.Network != "" {
		// The network parameter could be an RPC URL or a network name
		if strings.HasPrefix(config.Network, "http://") || strings.HasPrefix(config.Network, "https://") {
			flags = append(flags, "--rpc-url", config.Network)
		} else {
			// Assume it's a network name that will be resolved by forge
			flags = append(flags, "--fork-url", config.Network)
		}
	}
	
	// Add dry run flag
	if config.DryRun {
		flags = append(flags, "--simulate")
	}
	
	// Add debug flag
	if config.Debug {
		flags = append(flags, "-vvvv")
	}
	
	// Add sender if specified
	if config.Sender != "" {
		flags = append(flags, "--sender", config.Sender)
	}
	
	// Add any additional args
	flags = append(flags, config.Args...)
	
	// Run the script
	output, err := f.forge.RunScript(config.Path, flags, config.Environment)
	
	// Create result
	result := &usecase.ScriptResult{
		Success: err == nil,
		Output:  output,
		Error:   err,
	}
	
	// Extract broadcast files from output if successful
	if err == nil {
		result.Broadcasts = extractBroadcastFiles(output)
	}
	
	return result, nil
}

// extractBroadcastFiles extracts broadcast file paths from script output
func extractBroadcastFiles(output string) []string {
	var broadcasts []string
	
	// Look for broadcast file patterns in output
	// Example: "Transactions saved to: broadcast/DeployCounter.s.sol/31337/run-latest.json"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "broadcast/") && strings.Contains(line, ".json") {
			// Extract the path
			if idx := strings.Index(line, "broadcast/"); idx != -1 {
				path := line[idx:]
				// Clean up the path
				path = strings.TrimSpace(path)
				if strings.HasSuffix(path, ".json") {
					broadcasts = append(broadcasts, path)
				}
			}
		}
	}
	
	return broadcasts
}

// Ensure the adapter implements the interface
var _ usecase.ForgeExecutor = (*ForgeExecutorAdapter)(nil)
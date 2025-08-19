package forge

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptExecutorAdapter adapts the forge executor to the ScriptExecutor interface
type ScriptExecutorAdapter struct {
	projectPath string
	forge       *ForgeExecutor
	cfg         *config.RuntimeConfig
}

// NewScriptExecutorAdapter creates a new forge script executor adapter
func NewScriptExecutorAdapter(cfg *config.RuntimeConfig) *ScriptExecutorAdapter {
	return &ScriptExecutorAdapter{
		projectPath: cfg.ProjectRoot,
		forge:       NewForgeExecutor(cfg.ProjectRoot),
		cfg:         cfg,
	}
}

// Execute runs a Foundry script
func (a *ScriptExecutorAdapter) Execute(ctx context.Context, config usecase.ScriptExecutionConfig) (*usecase.ScriptExecutionOutput, error) {
	// Build ScriptOptions from config
	opts := ScriptOptions{
		Script:         config.Script.Path,
		Network:        config.Network.Name,
		Profile:        config.Namespace,
		EnvVars:        config.Environment,
		DryRun:         config.DryRun,
		Broadcast:      !config.DryRun && !config.Debug,
		Debug:          config.Debug,
		JSON:           !config.Debug, // Use JSON output when not in debug mode
		VerifyContract: false,         // Can be added to config later
	}

	// Check for hardware wallet flags in environment
	if hwType, ok := config.Environment["HARDWARE_WALLET_TYPE"]; ok {
		switch hwType {
		case "ledger":
			opts.UseLedger = true
			if paths, ok := config.Environment["DERIVATION_PATHS"]; ok {
				opts.DerivationPaths = strings.Split(paths, ",")
			}
		case "trezor":
			opts.UseTrezor = true
			if paths, ok := config.Environment["DERIVATION_PATHS"]; ok {
				opts.DerivationPaths = strings.Split(paths, ",")
			}
		}
	}

	// Add deployed libraries if present
	if libs, ok := config.Environment["DEPLOYED_LIBRARIES"]; ok && libs != "" {
		// Parse library references from environment
		// Format: "path:name:address path:name:address"
		opts.Libraries = strings.Split(libs, " ")
	}

	// Execute the script with the new ForgeExecutor
	result, err := a.forge.Run(opts)
	
	// Convert result to ScriptExecutionOutput
	if err != nil || !result.Success {
		return &usecase.ScriptExecutionOutput{
			Success:       false,
			RawOutput:     result.RawOutput,
			ParsedOutput:  string(result.RawOutput),
			ForgeOutput:   result.ParsedOutput,
			BroadcastPath: result.BroadcastPath,
		}, result.Error
	}

	// Find broadcast file if successful and not dry-run
	broadcastPath := result.BroadcastPath
	if broadcastPath == "" && !config.DryRun {
		// Try to find it from the status output
		if result.ParsedOutput != nil && result.ParsedOutput.StatusOutput != nil {
			broadcastPath = result.ParsedOutput.StatusOutput.Transactions
		}
		
		// Otherwise construct the expected path
		if broadcastPath == "" {
			contractFile := filepath.Base(config.Script.Path)
			chainID := "31337" // default local chain
			if config.Network != nil && config.Network.ChainID != 0 {
				chainID = fmt.Sprintf("%d", config.Network.ChainID)
			}
			broadcastPath = filepath.Join(a.projectPath, "broadcast", contractFile, chainID, "run-latest.json")
		}
	}

	return &usecase.ScriptExecutionOutput{
		Success:       true,
		RawOutput:     result.RawOutput,
		ParsedOutput:  string(result.RawOutput),
		ForgeOutput:   result.ParsedOutput,
		BroadcastPath: broadcastPath,
	}, nil
}

// Build runs forge build
func (a *ScriptExecutorAdapter) Build(ctx context.Context) error {
	return a.forge.Build()
}

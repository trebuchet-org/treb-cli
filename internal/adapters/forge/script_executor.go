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
	forge       *InternalForgeExecutor
	cfg         *config.RuntimeConfig
}

// NewScriptExecutorAdapter creates a new forge script executor adapter
func NewScriptExecutorAdapter(cfg *config.RuntimeConfig) *ScriptExecutorAdapter {
	return &ScriptExecutorAdapter{
		projectPath: cfg.ProjectRoot,
		forge:       NewInternalForgeExecutor(cfg.ProjectRoot),
		cfg:         cfg,
	}
}

// Execute runs a Foundry script
func (a *ScriptExecutorAdapter) Execute(ctx context.Context, config usecase.ScriptExecutionConfig) (*usecase.ScriptExecutionOutput, error) {
	// Build forge script arguments
	var flags []string
	
	// Add network flags
	// For local networks (anvil), we always need to provide the RPC URL
	if config.NetworkInfo != nil && config.NetworkInfo.RPCURL != "" {
		flags = append(flags, "--rpc-url", config.NetworkInfo.RPCURL)
	}
	
	// Add profile if not default
	if config.Namespace != "" && config.Namespace != "default" {
		flags = append(flags, "--profile", config.Namespace)
	}
	
	// Add broadcast/dry-run flags
	if config.DryRun {
		// No broadcast in dry-run mode
	} else {
		flags = append(flags, "--broadcast")
	}
	
	// Add debug flags
	if config.Debug {
		flags = append(flags, "-vvvv")
		// Also set DEBUG env var so the executor knows to print debug info
		config.Environment["DEBUG"] = "true"
	} else if config.DebugJSON {
		flags = append(flags, "--json")
	}
	
	// Check for hardware wallet flags in environment
	if hwType, ok := config.Environment["HARDWARE_WALLET_TYPE"]; ok {
		switch hwType {
		case "ledger":
			flags = append(flags, "--ledger")
			if paths, ok := config.Environment["DERIVATION_PATHS"]; ok {
				// Add each derivation path
				for _, path := range strings.Split(paths, ",") {
					if path != "" {
						flags = append(flags, "--mnemonic-derivation-paths", path)
					}
				}
			}
		case "trezor":
			flags = append(flags, "--trezor")
			if paths, ok := config.Environment["DERIVATION_PATHS"]; ok {
				// Add each derivation path
				for _, path := range strings.Split(paths, ",") {
					if path != "" {
						flags = append(flags, "--mnemonic-derivation-paths", path)
					}
				}
			}
		}
	}
	
	// Add deployed libraries if present
	if libs, ok := config.Environment["DEPLOYED_LIBRARIES"]; ok && libs != "" {
		// Parse library references from environment
		// Format: "path:name:address path:name:address"
		for _, lib := range strings.Split(libs, " ") {
			if lib != "" {
				flags = append(flags, "--libraries", lib)
			}
		}
	}
	
	// Execute the script
	scriptPath := config.Script.Path
	output, err := a.forge.RunScript(scriptPath, flags, config.Environment)
	
	// If there's an error, return it properly
	if err != nil {
		return &usecase.ScriptExecutionOutput{
			Success:       false,
			RawOutput:     []byte(output),
			ParsedOutput:  output,
			BroadcastPath: "",
		}, err
	}
	
	// Find broadcast file if successful and not dry-run
	var broadcastPath string
	if !config.DryRun {
		// Broadcast files are in broadcast/{contract_file}/{chain_id}/run-latest.json
		contractFile := filepath.Base(scriptPath)
		chainID := "31337" // default local chain
		if config.NetworkInfo != nil && config.NetworkInfo.ChainID != 0 {
			chainID = fmt.Sprintf("%d", config.NetworkInfo.ChainID)
		}
		broadcastPath = filepath.Join(a.projectPath, "broadcast", contractFile, chainID, "run-latest.json")
	}
	
	return &usecase.ScriptExecutionOutput{
		Success:       true,
		RawOutput:     []byte(output),
		ParsedOutput:  output, // For now, raw and parsed are the same
		BroadcastPath: broadcastPath,
	}, nil
}

// Build runs forge build
func (a *ScriptExecutorAdapter) Build(ctx context.Context) error {
	return a.forge.Build()
}
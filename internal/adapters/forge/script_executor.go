package forge

import (
	"context"
	"fmt"
	"os"
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
	flags = append(flags, "--rpc-url", config.Network.Name)

	// Add profile if not default
	if config.Namespace != "" && config.Namespace != "default" {
		flags = append(flags, "--profile", config.Namespace)
	}

	// Add broadcast/dry-run flags
	if !config.DryRun && !config.Debug {
		flags = append(flags, "--broadcast")
	}

	// Add debug flags
	if config.Debug {
		flags = append(flags, "-vvvv")
		// Also set DEBUG env var so the executor knows to print debug info
		config.Environment["DEBUG"] = "true"
	} else {
		// Always use JSON output for parsing events
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
	if config.Debug {
		fmt.Printf("Forge output: %s\n", output)
	}

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
		if config.Network != nil && config.Network.ChainID != 0 {
			chainID = fmt.Sprintf("%d", config.Network.ChainID)
		}
		broadcastPath = filepath.Join(a.projectPath, "broadcast", contractFile, chainID, "run-latest.json")
	}

	// Parse JSON output if not in debug mode
	var jsonOutput *ForgeJSONOutput
	if !config.Debug {
		parsed, err := ParseForgeJSONOutput(output)
		if err != nil && os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: Failed to parse forge JSON: %v\n", err)
		}
		jsonOutput = parsed

		// Debug: print parsed output
		if jsonOutput != nil && os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: Found %d raw logs in forge output\n", len(jsonOutput.RawLogs))
		}
	}

	return &usecase.ScriptExecutionOutput{
		Success:       true,
		RawOutput:     []byte(output),
		ParsedOutput:  output,
		JSONOutput:    jsonOutput,
		BroadcastPath: broadcastPath,
	}, nil
}

// Build runs forge build
func (a *ScriptExecutorAdapter) Build(ctx context.Context) error {
	return a.forge.Build()
}

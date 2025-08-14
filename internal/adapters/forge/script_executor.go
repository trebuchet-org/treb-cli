package forge

import (
	"context"
	"fmt"
	"strings"

	pkgconfig "github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/senders"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptExecutorAdapter adapts the existing forge package to the ScriptExecutor interface
type ScriptExecutorAdapter struct {
	projectPath string
	forge       *forge.Forge
}

// NewScriptExecutorAdapter creates a new forge script executor adapter
func NewScriptExecutorAdapter(cfg *config.RuntimeConfig) *ScriptExecutorAdapter {
	return &ScriptExecutorAdapter{
		projectPath: cfg.ProjectRoot,
		forge:       forge.NewForge(cfg.ProjectRoot),
	}
}

// Execute runs a Foundry script using the existing forge package
func (a *ScriptExecutorAdapter) Execute(ctx context.Context, config usecase.ScriptExecutionConfig) (*usecase.ScriptExecutionOutput, error) {
	// Convert domain script info to v1 contract info
	contractInfo := &types.ContractInfo{
		Name: config.Script.ContractName,
		Path: config.Script.Path,
	}
	if config.Script.Artifact != nil {
		// TODO: Map artifact data if needed
	}

	// Build sender configs from environment
	senderConfigs, err := a.buildSenderConfigs(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build sender configs: %w", err)
	}

	// Get deployed libraries from environment
	var libraries []string
	if libs, ok := config.Environment["DEPLOYED_LIBRARIES"]; ok && libs != "" {
		// Parse library references from environment
		// Format: "path:name:address,path:name:address"
		// This would be set by the EnvironmentBuilder
		libraries = parseLibraryReferences(libs)
	}

	// Build forge options
	forgeOpts := forge.ScriptOptions{
		Script:          contractInfo,
		Network:         config.Network,
		RpcUrl:          config.NetworkInfo.RPCURL,
		Profile:         config.Namespace,
		DryRun:          config.DryRun,
		Broadcast:       !config.DryRun,
		Debug:           config.Debug,
		JSON:            !config.Debug || config.DebugJSON,
		EnvVars:         config.Environment,
		UseLedger:       senders.RequiresLedgerFlag(senderConfigs),
		UseTrezor:       senders.RequiresTrezorFlag(senderConfigs),
		DerivationPaths: senders.GetDerivationPaths(senderConfigs),
		Libraries:       libraries,
	}

	// Execute the script
	result, err := a.forge.Run(forgeOpts)
	if err != nil {
		return nil, err
	}

	// Convert to use case output
	return &usecase.ScriptExecutionOutput{
		Success:       result.Success,
		RawOutput:     result.RawOutput,
		ParsedOutput:  result.ParsedOutput,
		BroadcastPath: result.BroadcastPath,
	}, nil
}

// buildSenderConfigs builds sender configs from the execution config
func (a *ScriptExecutorAdapter) buildSenderConfigs(config usecase.ScriptExecutionConfig) (*pkgconfig.SenderConfigs, error) {
	// The sender configs are already encoded in the environment by EnvironmentBuilder
	// We need to extract hardware wallet info for forge flags
	
	// For now, return empty configs as the actual configs are in env vars
	// The forge executor only needs this to detect hardware wallet flags
	configs := &pkgconfig.SenderConfigs{
		Configs: []pkgconfig.SenderInitConfig{},
	}
	
	return configs, nil
}

// parseLibraryReferences parses library references from environment string
func parseLibraryReferences(libs string) []string {
	// Format expected: "path:name:address path:name:address"
	// This matches the format produced by EnvironmentBuilder.encodeLibraries
	if libs == "" {
		return nil
	}
	
	// Split by spaces to get individual library references
	return strings.Split(libs, " ")
}
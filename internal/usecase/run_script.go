package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// RunScriptParams contains parameters for running a script
type RunScriptParams struct {
	ScriptRef      string
	Parameters     map[string]string
	DryRun         bool
	Debug          bool
	DebugJSON      bool
	Verbose        bool
	NonInteractive bool
	Progress       ProgressSink
}

// RunScriptResult contains the result of running a script
type RunScriptResult struct {
	Execution       *domain.ScriptExecution
	RegistryChanges *RegistryChanges
	Success         bool
	Error           error
}

// RunScript is the main use case for running deployment scripts
type RunScript struct {
	config         *config.RuntimeConfig
	scriptResolver ScriptResolver
	paramResolver  ParameterResolver
	// paramPrompter   ParameterPrompter
	scriptExecutor  ScriptExecutor
	executionParser ExecutionParser
	registryUpdater RegistryUpdater
	envBuilder      EnvironmentBuilder
	libraryResolver LibraryResolver
	progress        ProgressSink
}

// NewRunScript creates a new RunScript use case
func NewRunScript(
	cfg *config.RuntimeConfig,
	scriptResolver ScriptResolver,
	paramResolver ParameterResolver,
	// paramPrompter ParameterPrompter,
	scriptExecutor ScriptExecutor,
	executionParser ExecutionParser,
	registryUpdater RegistryUpdater,
	envBuilder EnvironmentBuilder,
	libraryResolver LibraryResolver,
	progress ProgressSink,
) *RunScript {
	return &RunScript{
		config:         cfg,
		scriptResolver: scriptResolver,
		paramResolver:  paramResolver,
		// paramPrompter:   paramPrompter,
		scriptExecutor:  scriptExecutor,
		executionParser: executionParser,
		registryUpdater: registryUpdater,
		envBuilder:      envBuilder,
		libraryResolver: libraryResolver,
		progress:        progress,
	}
}

// Run executes the script with the given parameters
func (uc *RunScript) Run(ctx context.Context, params RunScriptParams) (*RunScriptResult, error) {
	progress := params.Progress
	startTime := time.Now()

	// Initialize result
	result := &RunScriptResult{
		Success: false,
	}

	// Stage 1: Resolve script
	script, err := uc.scriptResolver.ResolveScript(ctx, params.ScriptRef)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve script: %w", err)
		return result, nil
	}

	// Stage 2: Resolve parameters
	scriptParams, err := uc.scriptResolver.GetScriptParameters(ctx, script)
	if err != nil {
		result.Error = fmt.Errorf("failed to get script parameters: %w", err)
		return result, nil
	}

	// Resolve parameter values
	resolvedParams, err := uc.paramResolver.ResolveParameters(ctx, scriptParams, params.Parameters)
	if err != nil {
		return nil, err
		// TODO: fix this after fixing parameters
		// if !params.NonInteractive {
		// 	// Try prompting for missing parameters
		// 	prompted, promptErr := uc.paramPrompter.PromptForParameters(ctx, scriptParams, resolvedParams)
		// 	if promptErr != nil {
		// 		result.Error = fmt.Errorf("failed to prompt for parameters: %w", promptErr)
		// 		return result, nil
		// 	}
		// 	resolvedParams = prompted
		// } else {
		// 	result.Error = fmt.Errorf("parameter resolution failed: %w", err)
		// 	return result, nil
		// }
	}

	// Validate all required parameters have values
	if err := uc.paramResolver.ValidateParameters(ctx, scriptParams, resolvedParams); err != nil {
		result.Error = fmt.Errorf("parameter validation failed: %w", err)
		return result, nil
	}

	// Use network info from RuntimeConfig if available
	if uc.config.Network == nil {
		// If network not in config, we need to resolve it
		// This would require a NetworkResolver port
		result.Error = fmt.Errorf("could not resolve network")
		return result, nil
	}

	// Convert config.TrebConfig to domain.TrebConfig
	var trebConfig *domain.TrebConfig
	if uc.config.TrebConfig != nil {
		trebConfig = &domain.TrebConfig{
			Senders: make(map[string]domain.SenderConfig),
		}
		for name, sender := range uc.config.TrebConfig.Senders {
			domainSender := domain.SenderConfig{
				Type:           sender.Type,
				Account:        sender.Account,
				PrivateKey:     sender.PrivateKey,
				Safe:           sender.Safe,
				DerivationPath: sender.DerivationPath,
				Signer:         sender.Signer, // Map legacy v1 field
			}
			if sender.Proposer != nil {
				domainSender.Proposer = &domain.ProposerConfig{
					Type:           sender.Proposer.Type,
					PrivateKey:     sender.Proposer.PrivateKey,
					DerivationPath: sender.Proposer.DerivationPath,
				}
			}
			trebConfig.Senders[name] = domainSender
		}
	}

	// Get deployed libraries
	libraries, err := uc.libraryResolver.GetDeployedLibraries(
		ctx,
		uc.config.Namespace,
		uc.config.Network.ChainID,
	)
	if err != nil {
		// Log warning but continue
		uc.progress.Info(fmt.Sprintf("Warning: Failed to load deployed libraries: %v", err))
		libraries = []LibraryReference{}
	}

	// Build environment
	envParams := BuildEnvironmentParams{
		Network:           uc.config.Network.Name,
		Namespace:         uc.config.Namespace,
		TrebConfig:        uc.config.TrebConfig,
		Parameters:        resolvedParams,
		DryRun:            params.DryRun,
		DeployedLibraries: libraries,
	}
	environment, err := uc.envBuilder.BuildEnvironment(ctx, envParams)
	if err != nil {
		result.Error = fmt.Errorf("failed to build environment: %w", err)
		return result, nil
	}

	execConfig := ScriptExecutionConfig{
		Network:     uc.config.Network,
		Namespace:   uc.config.Namespace,
		Script:      script,
		Environment: environment,
		DryRun:      params.DryRun,
		Debug:       params.Debug,
		DebugJSON:   params.DebugJSON,
		Progress:    progress,
	}

	// Stage 3: Execute script
	progress.OnProgress(ctx, ProgressEvent{
		Stage:    string(StageSimulating),
		Message:  "Simulating",
		Metadata: &execConfig,
	})

	output, err := uc.scriptExecutor.Execute(ctx, execConfig)
	if err != nil {
		result.Error = fmt.Errorf("script execution failed: %w", err)
		return result, nil
	}

	if !output.Success {
		result.Error = fmt.Errorf("script execution failed")
		return result, nil
	}

	// Stage 4: Parse execution
	progress.OnProgress(ctx, ProgressEvent{
		Stage:   string(StageParsing),
		Message: "Parsing",
	})
	execution, err := uc.executionParser.ParseExecution(ctx, output)

	if err != nil {
		result.Error = fmt.Errorf("failed to parse execution: %w", err)
		return result, nil
	}

	// Set execution metadata
	execution.ScriptPath = script.Path
	execution.ScriptName = script.Name
	execution.Network = *uc.config.Network
	execution.Namespace = uc.config.Namespace
	execution.DryRun = params.DryRun
	execution.ExecutedAt = startTime
	execution.ExecutionTime = time.Since(startTime)

	// Enrich from broadcast if available
	if output.BroadcastPath != "" && !params.DryRun {
		if err := uc.executionParser.EnrichFromBroadcast(ctx, execution, output.BroadcastPath); err != nil {
			// Log warning but continue
			progress.OnProgress(ctx, ProgressEvent{
				Stage:   string(StageParsing),
				Message: fmt.Sprintf("Warning: Failed to enrich from broadcast: %v", err),
			})
		}
	}

	result.Execution = execution

	// Stage 5: Update registry (if not dry run)
	if !params.DryRun && len(execution.Deployments) > 0 {
		progress.OnProgress(ctx, ProgressEvent{
			Stage: string(StageParsing),
		})

		changes, err := uc.registryUpdater.PrepareUpdates(ctx, execution)
		if err != nil {
			result.Error = fmt.Errorf("failed to prepare registry updates: %w", err)
			return result, nil
		}

		if uc.registryUpdater.HasChanges(changes) {
			if err := uc.registryUpdater.ApplyUpdates(ctx, changes); err != nil {
				result.Error = fmt.Errorf("failed to update registry: %w", err)
				return result, nil
			}
			result.RegistryChanges = changes
		}
	}

	// Stage 6: Complete
	progress.OnProgress(ctx, ProgressEvent{
		Stage: string(StageCompleted),
	})

	result.Success = true
	return result, nil
}

package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
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
	RunResult       *forge.HydratedRunResult
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
	forgeScriptRunner ForgeScriptRunner
	runResultHydrator RunResultHydrator
	registryUpdater   RegistryUpdater
	libraryResolver   LibraryResolver
	progress          ProgressSink
}

// NewRunScript creates a new RunScript use case
func NewRunScript(
	cfg *config.RuntimeConfig,
	scriptResolver ScriptResolver,
	paramResolver ParameterResolver,
	// paramPrompter ParameterPrompter,
	forgeScriptRunner ForgeScriptRunner,
	runResultHydrator RunResultHydrator,
	registryUpdater RegistryUpdater,
	libraryResolver LibraryResolver,
	progress ProgressSink,
) *RunScript {
	return &RunScript{
		config:         cfg,
		scriptResolver: scriptResolver,
		paramResolver:  paramResolver,
		// paramPrompter:   paramPrompter,
		forgeScriptRunner: forgeScriptRunner,
		runResultHydrator: runResultHydrator,
		registryUpdater:   registryUpdater,
		libraryResolver:   libraryResolver,
		progress:          progress,
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

	fmt.Println("DEBUG: RunScript.Run started")

	// Stage 1: Resolve script
	fmt.Printf("DEBUG: Resolving script: %s\n", params.ScriptRef)
	script, err := uc.scriptResolver.ResolveScript(ctx, params.ScriptRef)
	if err != nil {
		return result, fmt.Errorf("failed to resolve script: %w", err)

	}
	fmt.Printf("DEBUG: Script resolved: %s\n", script.Path)

	// Stage 2: Resolve parameters
	fmt.Println("DEBUG: Getting script parameters")
	scriptParams, err := uc.scriptResolver.GetScriptParameters(ctx, script)
	if err != nil {
		return result, fmt.Errorf("failed to get script parameters: %w", err)
	}
	fmt.Printf("DEBUG: Found %d script parameters\n", len(scriptParams))

	// Resolve parameter values
	fmt.Println("DEBUG: Resolving parameter values")
	resolvedParams, err := uc.paramResolver.ResolveParameters(ctx, scriptParams, params.Parameters)
	if err != nil {
		return result, err
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
	fmt.Println("DEBUG: Validating parameters")
	if err := uc.paramResolver.ValidateParameters(ctx, scriptParams, resolvedParams); err != nil {
		return result, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Use network info from RuntimeConfig if available
	if uc.config.Network == nil {
		return result, fmt.Errorf("could not resolve network")
	}
	fmt.Printf("DEBUG: Using network: %s (Chain ID: %d)\n", uc.config.Network.Name, uc.config.Network.ChainID)

	// Get deployed libraries
	fmt.Println("DEBUG: Getting deployed libraries")
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
	fmt.Printf("DEBUG: Found %d libraries\n", len(libraries))
	libraryStrings := make([]string, len(libraries))
	for i, library := range libraries {
		libraryStrings[i] = fmt.Sprintf(
			"%s:%s:%s",
			library.Path,
			library.Name,
			library.Address,
		)
	}

	// Build environment
	// envParams := BuildEnvironmentParams{
	// 	Network:           uc.config.Network.Name,
	// 	Namespace:         uc.config.Namespace,
	// 	TrebConfig:        uc.config.TrebConfig,
	// 	Parameters:        resolvedParams,
	// 	DryRun:            params.DryRun,
	// 	DeployedLibraries: libraries,
	// }
	// environment, err := uc.envBuilder.BuildEnvironment(ctx, envParams)
	// if err != nil {
	// 	result.Error = fmt.Errorf("failed to build environment: %w", err)
	// 	return result, nil
	// }

	// Stage 3: Execute script
	// progress.OnProgress(ctx, ProgressEvent{
	// 	Stage:    string(StageSimulating),
	// 	Message:  "Simulating",
	// 	Metadata: &execConfig,
	// })

	fmt.Println("DEBUG: Building sender config")
	senderManager := config.NewSendersManager(uc.config.TrebConfig)
	senderScriptConfig, err := senderManager.BuildSenderScriptConfig(script.Artifact)
	if err != nil {
		return result, err
	}
	fmt.Printf("DEBUG: Sender config built, encoded config length: %d\n", len(senderScriptConfig.EncodedConfig))

	fmt.Println("DEBUG: Calling forgeScriptRunner.RunScript")
	runResult, err := uc.forgeScriptRunner.RunScript(ctx, RunScriptConfig{
		Network:            uc.config.Network,
		Namespace:          uc.config.Namespace,
		Script:             script,
		Parameters:         resolvedParams,
		Libraries:          libraryStrings,
		DryRun:             params.DryRun,
		Debug:              params.Debug,
		DebugJSON:          params.DebugJSON,
		Progress:           progress,
		SenderScriptConfig: *senderScriptConfig,
	})
	fmt.Println("DEBUG: forgeScriptRunner.RunScript returned")

	result.RunResult = &forge.HydratedRunResult{RunResult: *runResult}

	if err != nil {
		result.Error = fmt.Errorf("script execution failed: %w", err)
		return result, nil
	}

	if !runResult.Success {
		result.Error = fmt.Errorf("script execution failed")
		return result, nil
	}

	// Stage 4: Parse execution
	progress.OnProgress(ctx, ProgressEvent{
		Stage:   string(StageParsing),
		Message: "Parsing",
	})

	hydratedRunResult, err := uc.runResultHydrator.Hydrate(ctx, runResult)
	if err != nil {
		result.Error = fmt.Errorf("failed to hydrate run result: %w", err)
		return result, nil
	}
	fmt.Println("DEBUG: runResultHydrator.Hydrate returned")

	// Set execution metadata
	hydratedRunResult.ExecutedAt = startTime
	hydratedRunResult.ExecutionTime = time.Since(startTime)

	result.RunResult = hydratedRunResult

	// fmt.Println("%v", result.RunResult.Deployments)

	// Stage 5: Update registry (if not dry run)
	if !params.DryRun && len(result.RunResult.Deployments) > 0 {
		progress.OnProgress(ctx, ProgressEvent{
			Stage: string(StageParsing),
		})

		changes, err := uc.registryUpdater.PrepareUpdates(ctx, result.RunResult)
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
		fmt.Println("DEBUG: deployment updated")
	}

	// Stage 6: Complete
	progress.OnProgress(ctx, ProgressEvent{
		Stage: string(StageCompleted),
	})

	result.Success = true
	return result, nil
}

package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
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
	Slow           bool
	DumpCommand    bool
}

// RunScriptResult contains the result of running a script
type RunScriptResult struct {
	RunResult     *forge.HydratedRunResult
	Changeset     *models.Changeset
	Success       bool
	Error         error
	DumpedCommand string
}

// RunScript is the main use case for running deployment scripts
type RunScript struct {
	config            *config.RuntimeConfig
	scriptResolver    ScriptResolver
	paramResolver     ParameterResolver
	sendersManager    SendersManager
	forgeScriptRunner ForgeScriptRunner
	runResultHydrator RunResultHydrator
	registryUpdater   DeploymentRepositoryUpdater
	libraryResolver   LibraryResolver
	progress          RunProgressSink
	forkStateStore    ForkStateStore
	anvilManager      AnvilManager
	forkFileManager   ForkFileManager
	// paramPrompter   ParameterPrompter
}

// NewRunScript creates a new RunScript use case
func NewRunScript(
	cfg *config.RuntimeConfig,
	scriptResolver ScriptResolver,
	paramResolver ParameterResolver,
	sendersManager SendersManager,
	runResultHydrator RunResultHydrator,
	registryUpdater DeploymentRepositoryUpdater,
	libraryResolver LibraryResolver,
	progress RunProgressSink,
	forgeScriptRunner ForgeScriptRunner,
	forkStateStore ForkStateStore,
	anvilManager AnvilManager,
	forkFileManager ForkFileManager,
) *RunScript {
	return &RunScript{
		config:            cfg,
		scriptResolver:    scriptResolver,
		paramResolver:     paramResolver,
		sendersManager:    sendersManager,
		forgeScriptRunner: forgeScriptRunner,
		runResultHydrator: runResultHydrator,
		registryUpdater:   registryUpdater,
		libraryResolver:   libraryResolver,
		progress:          progress,
		forkStateStore:    forkStateStore,
		anvilManager:      anvilManager,
		forkFileManager:   forkFileManager,
		// paramPrompter:   paramPrompter,
	}
}

// Run executes the script with the given parameters
func (uc *RunScript) Run(ctx context.Context, params RunScriptParams) (*RunScriptResult, error) {
	startTime := time.Now()

	// Initialize result
	result := &RunScriptResult{
		Success: false,
	}

	// Stage 1: Resolve script
	script, err := uc.scriptResolver.ResolveScript(ctx, params.ScriptRef)
	if err != nil {
		return result, fmt.Errorf("failed to resolve script: %w", err)

	}

	// Stage 2: Resolve parameters
	scriptParams, err := uc.scriptResolver.GetScriptParameters(ctx, script)
	if err != nil {
		return result, fmt.Errorf("failed to get script parameters: %w", err)
	}

	// Resolve parameter values
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
	if err := uc.paramResolver.ValidateParameters(ctx, scriptParams, resolvedParams); err != nil {
		return result, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Use network info from RuntimeConfig if available
	if uc.config.Network == nil {
		return result, fmt.Errorf("could not resolve network")
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
	libraryStrings := make([]string, len(libraries))
	for i, library := range libraries {
		libraryStrings[i] = fmt.Sprintf(
			"%s:%s:%s",
			library.Path,
			library.Name,
			library.Address,
		)
	}

	senderScriptConfig, err := uc.sendersManager.BuildSenderScriptConfig(script.Artifact)
	if err != nil {
		return result, err
	}

	// Check for active fork and build env overrides
	var forkEnvOverrides map[string]string
	if uc.config.Network != nil {
		forkState, forkErr := uc.forkStateStore.Load(ctx)
		if forkErr == nil {
			if fork := forkState.GetActiveFork(uc.config.Network.Name); fork != nil && fork.EnvVarName != "" {
				forkEnvOverrides = map[string]string{
					fork.EnvVarName: fork.ForkURL,
				}
			}
		}
	}

	runScriptConfig := RunScriptConfig{
		Network:            uc.config.Network,
		Namespace:          uc.config.Namespace,
		Script:             script,
		Parameters:         resolvedParams,
		Libraries:          libraryStrings,
		DryRun:             params.DryRun,
		Debug:              params.Debug,
		DebugJSON:          params.DebugJSON,
		Progress:           uc.progress,
		SenderScriptConfig: *senderScriptConfig,
		Slow:               params.Slow,
		ForkEnvOverrides:   forkEnvOverrides,
	}

	// Pre-run snapshot in fork mode: take EVM snapshot and backup files before execution
	if forkEnvOverrides != nil && uc.config.Network != nil {
		if snapshotErr := uc.takePreRunSnapshot(ctx, params.ScriptRef); snapshotErr != nil {
			return result, fmt.Errorf("failed to take pre-run fork snapshot: %w", snapshotErr)
		}
	}

	if params.DumpCommand {
		dumper, ok := uc.forgeScriptRunner.(interface {
			DumpScriptCommand(config RunScriptConfig) (string, error)
		})
		if !ok {
			return result, fmt.Errorf("forge runner does not support dumping command")
		}
		cmdLine, err := dumper.DumpScriptCommand(runScriptConfig)
		if err != nil {
			return result, err
		}
		result.DumpedCommand = cmdLine
		result.Success = true
		return result, nil
	}

	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage:    string(StageSimulating),
		Message:  "Simulating",
		Metadata: &runScriptConfig,
	})

	runResult, err := uc.forgeScriptRunner.RunScript(ctx, runScriptConfig)
	result.RunResult = &forge.HydratedRunResult{RunResult: runResult}

	if err != nil {
		result.Error = fmt.Errorf("script execution failed: %w", err)
		return result, nil
	}

	if !runResult.Success {
		result.Error = fmt.Errorf("script execution failed")
		return result, nil
	}

	// Stage 4: Parse execution
	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage:   string(StageParsing),
		Message: "Parsing",
	})

	hydratedRunResult, err := uc.runResultHydrator.Hydrate(ctx, runResult)
	if err != nil {
		result.Error = fmt.Errorf("failed to hydrate run result: %w", err)
		return result, nil
	}

	// Set execution metadata
	hydratedRunResult.ExecutedAt = startTime
	hydratedRunResult.ExecutionTime = time.Since(startTime)

	result.RunResult = hydratedRunResult

	// fmt.Println("%v", result.RunResult.Deployments)

	// Stage 5: Update registry (if not dry run)
	if !params.DryRun && len(result.RunResult.Deployments) > 0 {
		uc.progress.OnProgress(ctx, ProgressEvent{
			Stage: string(StageParsing),
		})

		changeset, err := uc.registryUpdater.BuildChangesetFromRunResult(ctx, result.RunResult)
		if err != nil {
			result.Error = fmt.Errorf("failed to prepare registry updates: %w", err)
			return result, nil
		}

		if changeset.HasChanges() {
			if err := uc.registryUpdater.ApplyChangeset(ctx, changeset); err != nil {
				result.Error = fmt.Errorf("failed to update registry: %w", err)
				return result, nil
			}
			result.Changeset = changeset
		}
	}

	// Stage 6: Complete
	uc.progress.OnProgress(ctx, ProgressEvent{
		Stage: string(StageCompleted),
	})

	result.Success = true
	return result, nil
}

// takePreRunSnapshot takes an EVM snapshot and backs up registry files before a fork mode run.
func (uc *RunScript) takePreRunSnapshot(ctx context.Context, scriptRef string) error {
	networkName := uc.config.Network.Name

	// Load fork state
	state, err := uc.forkStateStore.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load fork state: %w", err)
	}

	fork := state.GetActiveFork(networkName)
	if fork == nil {
		return fmt.Errorf("no active fork for network '%s'", networkName)
	}

	// Determine next snapshot index
	nextIndex := len(fork.Snapshots)

	// Build AnvilInstance from fork entry for snapshot call
	instance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", fork.Network),
		Port:    portFromURL(fork.ForkURL),
		ChainID: fmt.Sprintf("%d", fork.ChainID),
		PidFile: fork.PidFile,
		LogFile: fork.LogFile,
	}

	// Take EVM snapshot
	snapshotID, err := uc.anvilManager.TakeSnapshot(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to take EVM snapshot: %w", err)
	}

	// Backup registry files
	if err := uc.forkFileManager.BackupFiles(ctx, networkName, nextIndex); err != nil {
		return fmt.Errorf("failed to backup registry files: %w", err)
	}

	// Record snapshot in fork state
	fork.Snapshots = append(fork.Snapshots, domain.SnapshotEntry{
		Index:      nextIndex,
		SnapshotID: snapshotID,
		Command:    scriptRef,
		Timestamp:  time.Now(),
	})

	// Save updated fork state
	if err := uc.forkStateStore.Save(ctx, state); err != nil {
		return fmt.Errorf("failed to save fork state: %w", err)
	}

	return nil
}

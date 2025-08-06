package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	configpkg "github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	netpkg "github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/display"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/executor"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parameters"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// RunConfig contains configuration for running a script
type RunConfig struct {
	// Script identification
	ScriptPath string // Can be a path or contract name

	// Network and deployment config
	Network   string
	Namespace string

	// Environment variables
	EnvVars map[string]string

	// Execution options
	DryRun    bool
	Debug     bool
	DebugJSON bool
	Verbose   bool

	// Interaction mode
	NonInteractive bool

	// Working directory
	WorkDir string
}

// RunResult contains the results of running a script
type RunResult struct {
	Success       bool
	Execution     *parser.ScriptExecution
	BroadcastPath string
	RawOutput     []byte
}

// ScriptRunner handles the execution of Foundry scripts
type ScriptRunner struct {
	workDir    string
	indexer    *contracts.Indexer
	resolver   *resolvers.ContractsResolver
	interactive bool
}

// NewScriptRunner creates a new script runner
func NewScriptRunner(workDir string, interactive bool) (*ScriptRunner, error) {
	if workDir == "" {
		workDir = "."
	}

	indexer, err := contracts.GetGlobalIndexer(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	resolver := resolvers.NewContractsResolver(indexer, interactive)

	return &ScriptRunner{
		workDir:     workDir,
		indexer:     indexer,
		resolver:    resolver,
		interactive: interactive,
	}, nil
}

// Run executes a script with the given configuration
func (r *ScriptRunner) Run(config *RunConfig) (*RunResult, error) {
	// Apply defaults
	if config.WorkDir == "" {
		config.WorkDir = r.workDir
	}
	if config.Network == "" {
		config.Network = os.Getenv("DEPLOYMENT_NETWORK")
		if config.Network == "" {
			config.Network = "local"
		}
	}
	if config.Namespace == "" {
		// Try to load from context
		configManager := configpkg.NewManager(config.WorkDir)
		if cfg, err := configManager.Load(); err == nil {
			config.Namespace = cfg.Namespace
		}
		if config.Namespace == "" {
			config.Namespace = "default"
		}
	}

	// Resolve script contract
	scriptContract, err := r.resolver.ResolveContract(config.ScriptPath, types.ScriptContractFilter())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script contract: %w", err)
	}

	// Resolve network info
	networkResolver, err := netpkg.NewResolver(config.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create network resolver: %w", err)
	}
	networkInfo, err := networkResolver.ResolveNetwork(config.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network: %w", err)
	}

	// Load treb config
	fullConfig, err := configpkg.LoadTrebConfig(config.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load treb config: %w", err)
	}

	trebConfig, err := fullConfig.GetProfileTrebConfig(config.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile config: %w", err)
	}

	// Initialize environment variables if not provided
	if config.EnvVars == nil {
		config.EnvVars = make(map[string]string)
	}

	// Parse and resolve script parameters if artifact is available
	if scriptContract.Artifact != nil {
		paramParser := parameters.NewParameterParser()
		params, err := paramParser.ParseFromArtifact(scriptContract.Artifact)
		if err != nil {
			return nil, fmt.Errorf("failed to parse script parameters: %w", err)
		}

		if len(params) > 0 {
			resolvedEnvVars, err := r.resolveParameters(params, config.EnvVars, trebConfig, config, int64(networkInfo.ChainID))
			if err != nil {
				return nil, err
			}

			// Update env vars with resolved values
			for k, v := range resolvedEnvVars {
				if v != "" {
					config.EnvVars[k] = v
				}
			}
		}
	}

	// Display deployment banner
	display.PrintDeploymentBanner(filepath.Base(scriptContract.Path), config.Network, config.Namespace, config.DryRun)

	// Create script executor
	scriptExecutor := executor.NewExecutor(config.WorkDir, networkInfo)

	// Debug mode always implies dry run to prevent Safe transaction creation
	dryRun := config.DryRun
	if config.Debug || config.DebugJSON {
		dryRun = true
	}

	// Execute the script
	opts := executor.RunOptions{
		Script:    scriptContract,
		Network:   config.Network,
		Namespace: config.Namespace,
		EnvVars:   config.EnvVars,
		DryRun:    dryRun,
		Debug:     config.Debug || config.DebugJSON,
		DebugJSON: config.DebugJSON,
	}

	result, err := scriptExecutor.Run(opts)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return &RunResult{Success: false}, nil
	}

	// Parse the script result
	scriptParser := parser.NewParser(r.indexer)
	execution, err := scriptParser.Parse(result, config.Network, networkInfo.ChainID)
	if err != nil {
		display.PrintWarningMessage(fmt.Sprintf("Failed to parse script execution: %v", err))
	}

	// Display the execution results
	if execution != nil && (len(execution.Transactions) > 0 || len(execution.Events) > 0 || len(execution.Logs) > 0) {
		// Create display handler
		displayHandler := display.NewDisplay(r.indexer, execution)
		displayHandler.SetVerbose(config.Verbose)

		// Load sender configs to improve address display
		if senderConfigs, err := configpkg.BuildSenderConfigs(trebConfig); err == nil {
			displayHandler.SetSenderConfigs(senderConfigs)
		}

		// Enable registry-based ABI resolution for better transaction decoding
		if manager, err := registry.NewManager(config.WorkDir); err == nil {
			displayHandler.SetRegistryResolver(manager, networkInfo.ChainID)
		}

		// Display the execution
		displayHandler.DisplayExecution()

		// Update registry if not dry run
		if !config.DryRun {
			if err := r.updateRegistry(execution, config); err != nil {
				display.PrintErrorMessage(fmt.Sprintf("Failed to update registry: %v", err))
			}
		}
	} else if !config.DryRun {
		display.PrintWarningMessage("No events detected")
	}

	return &RunResult{
		Success:       true,
		Execution:     execution,
		BroadcastPath: result.BroadcastPath,
		RawOutput:     result.RawOutput,
	}, nil
}

// resolveParameters resolves script parameters, prompting for missing values if interactive
func (r *ScriptRunner) resolveParameters(params []parameters.Parameter, envVars map[string]string, trebConfig *configpkg.TrebConfig, runConfig *RunConfig, chainID int64) (map[string]string, error) {
	// Create parameter resolver
	paramResolver, err := parameters.NewParameterResolver(runConfig.WorkDir, trebConfig, runConfig.Namespace, runConfig.Network, uint64(chainID), r.interactive && !runConfig.NonInteractive)
	if err != nil {
		return nil, fmt.Errorf("failed to create parameter resolver: %w", err)
	}

	// Resolve all parameters
	resolvedEnvVars, err := paramResolver.ResolveAll(params, envVars)
	if err != nil {
		if runConfig.NonInteractive {
			return nil, fmt.Errorf("parameter resolution failed: %w", err)
		}
		// In interactive mode, we'll prompt for missing values
	}

	// Ensure we have a valid map even if resolution had errors
	if resolvedEnvVars == nil {
		resolvedEnvVars = make(map[string]string)
	}

	// Check for missing required parameters
	var missingRequired []parameters.Parameter
	for _, param := range params {
		if !param.Optional && resolvedEnvVars[param.Name] == "" {
			missingRequired = append(missingRequired, param)
		}
	}

	// Handle missing parameters
	if len(missingRequired) > 0 {
		if runConfig.NonInteractive {
			var missingNames []string
			for _, p := range missingRequired {
				missingNames = append(missingNames, p.Name)
			}
			return nil, fmt.Errorf("missing required parameters: %s", strings.Join(missingNames, ", "))
		} else if r.interactive {
			// Interactive mode: prompt for missing values
			fmt.Println("The script supports the following parameters:")
			for _, p := range params {
				var status, nameColor string
				if resolvedEnvVars[p.Name] != "" {
					// Present - green
					status = color.GreenString("✓")
					nameColor = color.GreenString(p.Name)
				} else if p.Optional {
					// Optional and missing - yellow
					status = color.YellowString("○")
					nameColor = color.YellowString(p.Name)
				} else {
					// Required and missing - red
					status = color.RedString("✗")
					nameColor = color.RedString(p.Name)
				}
				fmt.Printf("  %s %s (%s): %s\n", status, nameColor, p.Type, p.Description)
			}
			fmt.Println("\nMissing one or more required parameters.")

			prompter := parameters.NewParameterPrompter(paramResolver)
			promptedVars, err := prompter.PromptForMissingParameters(params, resolvedEnvVars)
			if err != nil {
				return nil, fmt.Errorf("failed to prompt for parameters: %w", err)
			}

			// Re-resolve with prompted values
			resolvedEnvVars, err = paramResolver.ResolveAll(params, promptedVars)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve prompted parameters: %w", err)
			}
		}
	}

	return resolvedEnvVars, nil
}

// updateRegistry updates the deployment registry with execution results
func (r *ScriptRunner) updateRegistry(execution *parser.ScriptExecution, config *RunConfig) error {
	manager, err := registry.NewManager(config.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to create registry manager: %w", err)
	}

	updater := manager.NewScriptExecutionUpdater(execution, config.Namespace, config.Network, config.ScriptPath)
	if updater.HasChanges() {
		if err := updater.Write(); err != nil {
			return err
		}
		display.PrintSuccessMessage(fmt.Sprintf("Updated registry for %s network in namespace %s", config.Network, config.Namespace))
	} else {
		fmt.Printf("%s- No registry changes recorded for %s network in namespace %s%s\n",
			display.ColorYellow, config.Network, config.Namespace, display.ColorReset)
	}

	return nil
}
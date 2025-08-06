package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/display"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/executor"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parameters"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ExecutorConfig contains configuration for the orchestration executor
type ExecutorConfig struct {
	Network     string
	Namespace   string
	Profile     string
	DryRun      bool
	Debug       bool
	DebugJSON   bool
	Verbose     bool
	NonInteractive bool
}

// Executor handles the execution of orchestrated deployments
type Executor struct {
	config    *ExecutorConfig
	indexer   *contracts.Indexer
	resolver  *resolvers.ContractsResolver
}

// NewExecutor creates a new orchestration executor
func NewExecutor(config *ExecutorConfig) (*Executor, error) {
	// Initialize contract indexer
	indexer, err := contracts.GetGlobalIndexer(".")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// Create contracts resolver
	resolver := resolvers.NewContractsResolver(indexer, !config.NonInteractive)

	return &Executor{
		config:   config,
		indexer:  indexer,
		resolver: resolver,
	}, nil
}

// Execute runs the orchestration plan
func (e *Executor) Execute(plan *ExecutionPlan) error {
	display.PrintBanner(fmt.Sprintf("🎯 Orchestrating %s", plan.Group))
	fmt.Printf("📋 Execution plan: %d components\n\n", len(plan.Components))

	// Display execution plan
	e.displayExecutionPlan(plan)

	// Execute each step
	for i, step := range plan.Components {
		if err := e.executeStep(i+1, len(plan.Components), step); err != nil {
			return fmt.Errorf("failed to execute step %d (%s): %w", i+1, step.Name, err)
		}
	}

	display.PrintSuccessMessage(fmt.Sprintf("🎉 Successfully orchestrated %s deployment", plan.Group))
	return nil
}

// displayExecutionPlan shows the execution plan to the user
func (e *Executor) displayExecutionPlan(plan *ExecutionPlan) {
	fmt.Printf("%s📋 Execution Plan:%s\n", display.ColorBold, display.ColorReset)
	fmt.Printf("%s%s%s\n", display.ColorGray, display.StringRepeat("─", 50), display.ColorReset)

	for i, step := range plan.Components {
		// Show step number and component name
		fmt.Printf("%s%d.%s %s%s%s", 
			display.ColorGray, i+1, display.ColorReset,
			display.ColorCyan, step.Name, display.ColorReset)

		// Show script
		fmt.Printf(" → %s%s%s", display.ColorGreen, step.Script, display.ColorReset)

		// Show dependencies if any
		if len(step.Dependencies) > 0 {
			fmt.Printf(" %s(depends on: %v)%s", display.ColorGray, step.Dependencies, display.ColorReset)
		}

		// Show environment variables if any
		if len(step.Env) > 0 {
			fmt.Printf("\n   %sEnv: %v%s", display.ColorYellow, step.Env, display.ColorReset)
		}

		fmt.Println()
	}

	fmt.Println()
}

// executeStep executes a single step in the orchestration
func (e *Executor) executeStep(stepNum, totalSteps int, step *ExecutionStep) error {
	// Display step header
	fmt.Printf("%s[%d/%d] Executing %s%s\n", 
		display.ColorBold, stepNum, totalSteps, step.Name, display.ColorReset)
	fmt.Printf("%s%s%s\n", display.ColorGray, display.StringRepeat("─", 70), display.ColorReset)

	// Resolve the script contract using the same logic as the run command
	scriptContract, err := e.resolver.ResolveContract(step.Script, types.ScriptContractFilter())
	if err != nil {
		return fmt.Errorf("failed to resolve script contract '%s': %w", step.Script, err)
	}

	// Resolve network
	networkResolver, err := network.NewResolver(".")
	if err != nil {
		return fmt.Errorf("failed to create network resolver: %w", err)
	}
	networkInfo, err := networkResolver.ResolveNetwork(e.config.Network)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Set up environment variables
	originalEnv := make(map[string]string)
	envVars := make(map[string]string)
	
	// Copy step environment variables
	for key, value := range step.Env {
		if original := os.Getenv(key); original != "" {
			originalEnv[key] = original
		}
		os.Setenv(key, value)
		envVars[key] = value
	}

	// Restore original environment variables after execution
	defer func() {
		for key := range step.Env {
			if original, exists := originalEnv[key]; exists {
				os.Setenv(key, original)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Load treb config for parameter resolution
	fullConfig, err := config.LoadTrebConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load treb config: %w", err)
	}

	trebConfig, err := fullConfig.GetProfileTrebConfig(e.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get profile config: %w", err)
	}

	// Parse and resolve script parameters if needed
	if scriptContract.Artifact != nil {
		paramParser := parameters.NewParameterParser()
		params, err := paramParser.ParseFromArtifact(scriptContract.Artifact)
		if err != nil {
			return fmt.Errorf("failed to parse script parameters: %w", err)
		}

		if len(params) > 0 {
			// Create parameter resolver
			paramResolver, err := parameters.NewParameterResolver(".", trebConfig, e.config.Namespace, e.config.Network, networkInfo.ChainID, !e.config.NonInteractive)
			if err != nil {
				return fmt.Errorf("failed to create parameter resolver: %w", err)
			}

			// Resolve all parameters
			resolvedEnvVars, err := paramResolver.ResolveAll(params, envVars)
			if err != nil && e.config.NonInteractive {
				return fmt.Errorf("parameter resolution failed: %w", err)
			}

			// Update environment variables with resolved parameters
			if resolvedEnvVars != nil {
				for key, value := range resolvedEnvVars {
					envVars[key] = value
				}
			}
		}
	}

	// Display deployment banner for this step
	display.PrintDeploymentBanner(filepath.Base(scriptContract.Path), e.config.Network, e.config.Namespace, e.config.DryRun)

	// Create script executor
	scriptExecutor := executor.NewExecutor(".", networkInfo)

	// Execute the script
	opts := executor.RunOptions{
		Script:    scriptContract,
		Network:   e.config.Network,
		Namespace: e.config.Namespace,
		EnvVars:   envVars,
		DryRun:    e.config.DryRun,
		Debug:     e.config.Debug,
		DebugJSON: e.config.DebugJSON,
	}

	result, err := scriptExecutor.Run(opts)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("script execution was not successful")
	}

	// Parse the execution results
	scriptParser := parser.NewParser(e.indexer)
	execution, err := scriptParser.Parse(result, e.config.Network, networkInfo.ChainID)
	if err != nil {
		return fmt.Errorf("failed to parse script result: %w", err)
	}

	// Display the execution results
	displayHandler := display.NewDisplay(e.indexer, execution)
	displayHandler.SetVerbose(e.config.Verbose)
	
	displayHandler.DisplayExecution()

	// Small delay between steps for readability
	if stepNum < totalSteps {
		time.Sleep(500 * time.Millisecond)
		fmt.Println()
	}

	return nil
}
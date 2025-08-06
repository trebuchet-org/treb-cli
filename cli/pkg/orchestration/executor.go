package orchestration

import (
	"fmt"
	"os"
	"time"

	"github.com/trebuchet-org/treb-cli/cli/pkg/script/display"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/runner"
)

// ExecutorConfig contains configuration for the orchestration executor
type ExecutorConfig struct {
	Network        string
	Namespace      string
	Profile        string
	DryRun         bool
	Debug          bool
	DebugJSON      bool
	Verbose        bool
	NonInteractive bool
}

// Executor handles the execution of orchestrated deployments
type Executor struct {
	config       *ExecutorConfig
	scriptRunner *runner.ScriptRunner
}

// NewExecutor creates a new orchestration executor
func NewExecutor(config *ExecutorConfig) (*Executor, error) {
	// Create script runner with non-interactive mode for orchestration
	scriptRunner, err := runner.NewScriptRunner(".", false)
	if err != nil {
		return nil, fmt.Errorf("failed to create script runner: %w", err)
	}

	return &Executor{
		config:       config,
		scriptRunner: scriptRunner,
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

	// Set up environment variables
	originalEnv := make(map[string]string)
	for key, value := range step.Env {
		if original := os.Getenv(key); original != "" {
			originalEnv[key] = original
		}
		os.Setenv(key, value)
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

	// Create run configuration
	runConfig := &runner.RunConfig{
		ScriptPath:     step.Script,
		Network:        e.config.Network,
		Namespace:      e.config.Namespace,
		EnvVars:        step.Env,
		DryRun:         e.config.DryRun,
		Debug:          e.config.Debug,
		DebugJSON:      e.config.DebugJSON,
		Verbose:        e.config.Verbose,
		NonInteractive: true, // Orchestration is always non-interactive
		WorkDir:        ".",
	}

	// Run the script using the shared runner
	result, err := e.scriptRunner.Run(runConfig)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("script execution was not successful")
	}

	// Small delay between steps for readability
	if stepNum < totalSteps {
		time.Sleep(500 * time.Millisecond)
		fmt.Println()
	}

	return nil
}

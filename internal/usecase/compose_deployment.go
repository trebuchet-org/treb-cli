package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// ComposeDeployment handles orchestrated deployments from YAML configuration
type ComposeDeployment struct {
	runScript *RunScript
	progress  ComposeSink
}

// NewComposeDeployment creates a new orchestrate deployment use case
func NewComposeDeployment(
	runScript *RunScript,
	progress ComposeSink,
) *ComposeDeployment {
	return &ComposeDeployment{
		runScript: runScript,
		progress:  progress,
	}
}

// ComposeParams contains parameters for orchestration
type ComposeParams struct {
	ConfigPath     string
	Network        string
	Namespace      string
	Profile        string
	DryRun         bool
	Debug          bool
	DebugJSON      bool
	Verbose        bool
	NonInteractive bool
	Resume         bool // Resume from previous execution
	Slow           bool
	DumpCommand    bool // Print the underlying forge commands without executing
}

// ComposeResult contains the result of orchestration
type ComposeResult struct {
	Plan             *ExecutionPlan
	ExecutedSteps    []*StepResult
	FailedStep       *StepResult
	Success          bool
	TotalDeployments int
}

// StepResult contains the result of executing a single step
type StepResult struct {
	Step      *ExecutionStep
	RunResult *RunScriptResult
	Error     error
}

// StepStateInfo is a lightweight version of StepResult for state storage
type StepStateInfo struct {
	Step        *ExecutionStep `json:"step"`
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
	Deployments int            `json:"deployments,omitempty"`
}

// ComposeState represents the state of a compose execution
type ComposeState struct {
	StartedAt        time.Time                 `json:"started_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	ConfigPath       string                    `json:"config_path"`
	Network          string                    `json:"network"`
	Namespace        string                    `json:"namespace"`
	Plan             *ExecutionPlan            `json:"plan"`
	ExecutedSteps    map[string]*StepStateInfo `json:"executed_steps"`
	CurrentStepIndex int                       `json:"current_step_index"`
	Status           string                    `json:"status"` // "running", "failed", "completed"
	TotalDeployments int                       `json:"total_deployments"`
}

// Execute runs the orchestration
func (o *ComposeDeployment) Execute(ctx context.Context, params ComposeParams) (*ComposeResult, error) {
	var state *ComposeState
	var plan *ExecutionPlan
	startIndex := 0

	// Handle resume mode
	if params.Resume {
		// Try to load previous state
		prevState, err := o.loadState(params.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resume: %w", err)
		}

		// Validate that we're resuming the same configuration
		if prevState.ConfigPath != params.ConfigPath {
			return nil, fmt.Errorf("cannot resume: config path changed (was %s, now %s)", prevState.ConfigPath, params.ConfigPath)
		}

		if prevState.Status == "completed" {
			return nil, fmt.Errorf("previous run already completed successfully")
		}

		// Use the existing plan and start from where we left off
		plan = prevState.Plan
		state = prevState
		startIndex = prevState.CurrentStepIndex

		// Emit resume event
		o.progress.OnProgress(ctx, ProgressEvent{
			Stage: "compose_resumed",
			Metadata: map[string]interface{}{
				"from_step": startIndex,
				"total":     len(plan.Components),
			},
		})
	} else {
		// Parse orchestration file
		config, err := o.parseComposeFile(params.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse orchestration file: %w", err)
		}

		// Validate configuration
		if err := config.Validate(); err != nil {
			return nil, fmt.Errorf("invalid orchestration configuration: %w", err)
		}

		// Create execution plan
		plan, err = o.createExecutionPlan(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create execution plan: %w", err)
		}

		// Initialize new state
		state = &ComposeState{
			StartedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			ConfigPath:       params.ConfigPath,
			Network:          params.Network,
			Namespace:        params.Namespace,
			Plan:             plan,
			ExecutedSteps:    make(map[string]*StepStateInfo),
			CurrentStepIndex: 0,
			Status:           "running",
			TotalDeployments: 0,
		}

		// Save initial state
		if err := o.saveState(state); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to save initial state: %v\n", err)
		}
	}

	// Emit plan created event so it can be rendered
	o.progress.OnProgress(ctx, ProgressEvent{
		Stage:    "plan_created",
		Metadata: plan,
	})

	// Build result from state
	result := &ComposeResult{
		Plan:             plan,
		ExecutedSteps:    make([]*StepResult, 0),
		Success:          true,
		TotalDeployments: state.TotalDeployments,
	}

	// Add previously executed steps to result - we only have StepStateInfo, not full StepResult
	// This is fine since we're resuming and these steps were already completed
	for i := 0; i < startIndex; i++ {
		if stateInfo, exists := state.ExecutedSteps[plan.Components[i].Name]; exists {
			// Create a minimal StepResult for display purposes
			stepResult := &StepResult{
				Step: stateInfo.Step,
				RunResult: &RunScriptResult{
					Success: stateInfo.Success,
				},
				Error: nil,
			}
			if stateInfo.Error != "" {
				stepResult.Error = fmt.Errorf("%s", stateInfo.Error)
			}
			result.ExecutedSteps = append(result.ExecutedSteps, stepResult)

			// Add the deployment count from the saved state
			if stateInfo.Deployments > 0 {
				// We can't recreate the full changeset, but we can track the count
				result.TotalDeployments += stateInfo.Deployments
			}
		}
	}

	// Execute remaining steps
	for i := startIndex; i < len(plan.Components); i++ {
		step := plan.Components[i]

		// Update current step in state
		state.CurrentStepIndex = i
		state.Status = "running"
		if err := o.saveState(state); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
		}

		// Emit step starting event
		o.progress.OnProgress(ctx, ProgressEvent{
			Stage: "step_starting",
			Metadata: map[string]any{
				"name":    step.Name,
				"script":  step.Script,
				"current": i + 1,
				"total":   len(plan.Components),
			},
		})

		// Execute the step
		stepResult := o.executeStep(ctx, step, params)
		result.ExecutedSteps = append(result.ExecutedSteps, stepResult)

		// Convert to lightweight state info for storage
		stateInfo := &StepStateInfo{
			Step:    step,
			Success: stepResult.Error == nil && (stepResult.RunResult == nil || stepResult.RunResult.Success),
			Error:   "",
		}
		if stepResult.Error != nil {
			stateInfo.Error = stepResult.Error.Error()
		}
		if stepResult.RunResult != nil && stepResult.RunResult.Changeset != nil {
			stateInfo.Deployments = len(stepResult.RunResult.Changeset.Create.Deployments)
		}
		state.ExecutedSteps[step.Name] = stateInfo

		// Emit step completed event
		o.progress.OnProgress(ctx, ProgressEvent{
			Stage:    "step_completed",
			Metadata: stepResult,
		})

		// Check for failure - either error or unsuccessful result
		if stepResult.Error != nil || (stepResult.RunResult != nil && !stepResult.RunResult.Success) {
			result.FailedStep = stepResult
			result.Success = false
			state.Status = "failed"

			// If there's no error but success is false, create an error
			if stepResult.Error == nil {
				stepResult.Error = fmt.Errorf("step '%s' failed", step.Name)
			}

			// Save failed state
			if err := o.saveState(state); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save failed state: %v\n", err)
			}

			break
		}

		// Count deployments
		if stepResult.RunResult != nil && stepResult.RunResult.Changeset != nil {
			deploymentCount := len(stepResult.RunResult.Changeset.Create.Deployments)
			result.TotalDeployments += deploymentCount
			state.TotalDeployments += deploymentCount
		}

		// Save successful step state
		if err := o.saveState(state); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state after step: %v\n", err)
		}
	}

	// Update final state
	if result.Success {
		state.Status = "completed"
		state.CurrentStepIndex = len(plan.Components)
		if err := o.saveState(state); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save final state: %v\n", err)
		}
	}

	// Emit compose completed event
	o.progress.OnProgress(ctx, ProgressEvent{
		Stage: "compose_completed",
	})

	return result, nil
}

// getStateFilePath returns the path to the compose state file
func (o *ComposeDeployment) getStateFilePath(configPath string) (string, error) {
	// Get the working directory
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Create the out/.treb directory if it doesn't exist
	stateDir := filepath.Join(workDir, "out", ".treb")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return "", err
	}

	// Get the base name of the config file without extension
	configBase := filepath.Base(configPath)
	configName := configBase
	if ext := filepath.Ext(configBase); ext != "" {
		configName = configBase[:len(configBase)-len(ext)]
	}

	// Create state file name like "compose-<config-name>.json"
	stateFileName := fmt.Sprintf("compose-%s.json", configName)
	return filepath.Join(stateDir, stateFileName), nil
}

// saveState saves the current compose state to disk
func (o *ComposeDeployment) saveState(state *ComposeState) error {
	statePath, err := o.getStateFilePath(state.ConfigPath)
	if err != nil {
		return err
	}

	state.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadState loads a previous compose state from disk
func (o *ComposeDeployment) loadState(configPath string) (*ComposeState, error) {
	statePath, err := o.getStateFilePath(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no previous compose run found for %s", filepath.Base(configPath))
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ComposeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// parseComposeFile parses the YAML orchestration file
func (o *ComposeDeployment) parseComposeFile(path string) (*ComposeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// createExecutionPlan creates a linearized execution plan from the configuration
func (o *ComposeDeployment) createExecutionPlan(config *ComposeConfig) (*ExecutionPlan, error) {
	graph := NewDependencyGraph(config)
	steps, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	return &ExecutionPlan{
		Group:      config.Group,
		Components: steps,
	}, nil
}

// executeStep executes a single orchestration step
func (o *ComposeDeployment) executeStep(ctx context.Context, step *ExecutionStep, params ComposeParams) *StepResult {
	// Prepare parameters for run script
	scriptParams := RunScriptParams{
		ScriptRef:      step.Script,
		Parameters:     step.Env,
		DryRun:         params.DryRun,
		Debug:          params.Debug,
		DebugJSON:      params.DebugJSON,
		Verbose:        params.Verbose,
		Slow:           params.Slow,
		NonInteractive: true, // Always non-interactive for orchestration
		DumpCommand:    params.DumpCommand,
	}

	// Execute the script
	runResult, err := o.runScript.Run(ctx, scriptParams)

	return &StepResult{
		Step:      step,
		RunResult: runResult,
		Error:     err,
	}
}

// Compose configuration types

// ComposeConfig represents the top-level configuration for orchestrated deployments
type ComposeConfig struct {
	Group      string                      `yaml:"group"`
	Components map[string]*ComponentConfig `yaml:"components"`
}

// ComponentConfig represents a single component in the orchestration
type ComponentConfig struct {
	Script string            `yaml:"script"`
	Deps   []string          `yaml:"deps,omitempty"`
	Env    map[string]string `yaml:"env,omitempty"`
}

// ExecutionPlan represents the linearized execution plan
type ExecutionPlan struct {
	Group      string
	Components []*ExecutionStep
}

// ExecutionStep represents a single step in the execution plan
type ExecutionStep struct {
	Name         string
	Script       string
	Env          map[string]string
	Dependencies []string // For reference/debugging
}

// Validate checks the orchestration configuration for errors
func (config *ComposeConfig) Validate() error {
	if config.Group == "" {
		return fmt.Errorf("group name is required")
	}

	if len(config.Components) == 0 {
		return fmt.Errorf("at least one component is required")
	}

	// Check for self-dependencies and non-existent dependencies
	for name, component := range config.Components {
		if component.Script == "" {
			return fmt.Errorf("component '%s' must specify a script", name)
		}

		for _, dep := range component.Deps {
			if dep == name {
				return fmt.Errorf("component '%s' cannot depend on itself", name)
			}

			if _, exists := config.Components[dep]; !exists {
				return fmt.Errorf("component '%s' depends on non-existent component '%s'", name, dep)
			}
		}
	}

	return nil
}

// DependencyGraph represents a directed acyclic graph of components
type DependencyGraph struct {
	nodes map[string]*ComponentConfig
	edges map[string][]string // adjacency list: node -> list of dependents
}

// NewDependencyGraph creates a new dependency graph from the orchestration config
func NewDependencyGraph(config *ComposeConfig) *DependencyGraph {
	graph := &DependencyGraph{
		nodes: config.Components,
		edges: make(map[string][]string),
	}

	// Build the dependency graph
	for name, component := range config.Components {
		for _, dep := range component.Deps {
			// Check if dependency exists
			if _, exists := config.Components[dep]; !exists {
				// We'll handle this error during validation
				continue
			}

			// Add edge from dependency to this component
			graph.edges[dep] = append(graph.edges[dep], name)
		}
	}

	return graph
}

// TopologicalSort performs a topological sort on the dependency graph
// Returns the components in execution order, or an error if there's a cycle
func (g *DependencyGraph) TopologicalSort() ([]*ExecutionStep, error) {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for name := range g.nodes {
		inDegree[name] = 0
	}

	// Count dependencies (in-degree)
	for name, component := range g.nodes {
		for _, dep := range component.Deps {
			if _, exists := g.nodes[dep]; !exists {
				return nil, fmt.Errorf("component '%s' depends on non-existent component '%s'", name, dep)
			}
			inDegree[name]++
		}
	}

	// Initialize queue with nodes that have no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort the initial queue for deterministic output
	sort.Strings(queue)

	var result []*ExecutionStep

	for len(queue) > 0 {
		// Remove first element from queue
		current := queue[0]
		queue = queue[1:]

		// Add to result
		component := g.nodes[current]
		step := &ExecutionStep{
			Name:         current,
			Script:       component.Script,
			Env:          component.Env,
			Dependencies: component.Deps,
		}
		result = append(result, step)

		// Process all dependents of current node
		dependents := g.edges[current]
		sort.Strings(dependents) // For deterministic output

		for _, dependent := range dependents {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
				// Keep queue sorted for deterministic output
				sort.Strings(queue)
			}
		}
	}

	// Check for cycles
	if len(result) != len(g.nodes) {
		// Find remaining nodes with dependencies (part of cycles)
		var cycleNodes []string
		for name, degree := range inDegree {
			if degree > 0 {
				cycleNodes = append(cycleNodes, name)
			}
		}
		return nil, fmt.Errorf("circular dependency detected involving components: %v", cycleNodes)
	}

	return result, nil
}

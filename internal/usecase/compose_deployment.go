package usecase

import (
	"context"
	"fmt"
	"os"
	"sort"

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

// Execute runs the orchestration
func (o *ComposeDeployment) Execute(ctx context.Context, params ComposeParams) (*ComposeResult, error) {
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
	plan, err := o.createExecutionPlan(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	// Emit plan created event so it can be rendered
	o.progress.OnProgress(ctx, ProgressEvent{
		Stage:    "plan_created",
		Metadata: plan,
	})

	// Execute each step
	result := &ComposeResult{
		Plan:          plan,
		ExecutedSteps: make([]*StepResult, 0),
		Success:       true,
	}

	for i, step := range plan.Components {
		// Emit step starting event
		o.progress.OnProgress(ctx, ProgressEvent{
			Stage: "step_starting",
			Metadata: map[string]interface{}{
				"name":    step.Name,
				"script":  step.Script,
				"current": i + 1,
				"total":   len(plan.Components),
			},
		})

		// Execute the step
		stepResult := o.executeStep(ctx, step, params)
		result.ExecutedSteps = append(result.ExecutedSteps, stepResult)

		// Emit step completed event
		o.progress.OnProgress(ctx, ProgressEvent{
			Stage:    "step_completed",
			Metadata: stepResult,
		})

		if stepResult.Error != nil {
			result.FailedStep = stepResult
			result.Success = false
			break
		}

		// Count deployments
		if stepResult.RunResult != nil && stepResult.RunResult.Changeset != nil {
			result.TotalDeployments += len(stepResult.RunResult.Changeset.Create.Deployments)
		}
	}

	// Emit compose completed event
	o.progress.OnProgress(ctx, ProgressEvent{
		Stage: "compose_completed",
	})

	return result, nil
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
		NonInteractive: true, // Always non-interactive for orchestration
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

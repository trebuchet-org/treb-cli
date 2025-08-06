package orchestration

import (
	"fmt"
	"sort"
)

// OrchestrationConfig represents the top-level configuration for orchestrated deployments
type OrchestrationConfig struct {
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

// DependencyGraph represents a directed acyclic graph of components
type DependencyGraph struct {
	nodes map[string]*ComponentConfig
	edges map[string][]string // adjacency list: node -> list of dependents
}

// NewDependencyGraph creates a new dependency graph from the orchestration config
func NewDependencyGraph(config *OrchestrationConfig) *DependencyGraph {
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

// Validate checks the orchestration configuration for errors
func (config *OrchestrationConfig) Validate() error {
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

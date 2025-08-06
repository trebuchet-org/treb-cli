package orchestration

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing orchestration configurations
type Parser struct{}

// NewParser creates a new orchestration parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses an orchestration configuration from a YAML file
func (p *Parser) ParseFile(filePath string) (*OrchestrationConfig, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("orchestration file not found: %s", absPath)
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read orchestration file: %w", err)
	}

	return p.Parse(data)
}

// Parse parses an orchestration configuration from YAML data
func (p *Parser) Parse(data []byte) (*OrchestrationConfig, error) {
	var config OrchestrationConfig
	
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid orchestration configuration: %w", err)
	}

	return &config, nil
}

// CreateExecutionPlan creates an execution plan from the orchestration config
func (p *Parser) CreateExecutionPlan(config *OrchestrationConfig) (*ExecutionPlan, error) {
	// Build dependency graph
	graph := NewDependencyGraph(config)

	// Perform topological sort
	steps, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	return &ExecutionPlan{
		Group:      config.Group,
		Components: steps,
	}, nil
}
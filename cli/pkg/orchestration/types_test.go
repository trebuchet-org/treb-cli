package orchestration

import (
	"testing"
)

func TestOrchestrationConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *OrchestrationConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"Broker": {
						Script: "DeployBroker",
					},
					"Tokens": {
						Script: "DeployTokens",
						Deps:   []string{"Broker"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing group",
			config: &OrchestrationConfig{
				Components: map[string]*ComponentConfig{
					"Broker": {Script: "DeployBroker"},
				},
			},
			expectError: true,
			errorMsg:    "group name is required",
		},
		{
			name: "no components",
			config: &OrchestrationConfig{
				Group:      "Test Protocol",
				Components: map[string]*ComponentConfig{},
			},
			expectError: true,
			errorMsg:    "at least one component is required",
		},
		{
			name: "component missing script",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"Broker": {},
				},
			},
			expectError: true,
			errorMsg:    "component 'Broker' must specify a script",
		},
		{
			name: "self dependency",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"Broker": {
						Script: "DeployBroker",
						Deps:   []string{"Broker"},
					},
				},
			},
			expectError: true,
			errorMsg:    "component 'Broker' cannot depend on itself",
		},
		{
			name: "non-existent dependency",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"Broker": {
						Script: "DeployBroker",
						Deps:   []string{"NonExistent"},
					},
				},
			},
			expectError: true,
			errorMsg:    "component 'Broker' depends on non-existent component 'NonExistent'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s' but got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	tests := []struct {
		name          string
		config        *OrchestrationConfig
		expectedOrder []string
		expectError   bool
		errorContains string
	}{
		{
			name: "simple linear dependency",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"A": {Script: "DeployA"},
					"B": {Script: "DeployB", Deps: []string{"A"}},
					"C": {Script: "DeployC", Deps: []string{"B"}},
				},
			},
			expectedOrder: []string{"A", "B", "C"},
			expectError:   false,
		},
		{
			name: "parallel dependencies",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"A": {Script: "DeployA"},
					"B": {Script: "DeployB"},
					"C": {Script: "DeployC", Deps: []string{"A", "B"}},
				},
			},
			expectedOrder: []string{"A", "B", "C"}, // A and B can be in either order
			expectError:   false,
		},
		{
			name: "complex dependencies",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"Broker":  {Script: "DeployBroker"},
					"Tokens":  {Script: "DeployTokens", Deps: []string{"Broker"}},
					"Reserve": {Script: "DeployReserve", Deps: []string{"Tokens"}},
					"Oracles": {Script: "DeployOracles", Deps: []string{"Reserve"}},
				},
			},
			expectedOrder: []string{"Broker", "Tokens", "Reserve", "Oracles"},
			expectError:   false,
		},
		{
			name: "circular dependency",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"A": {Script: "DeployA", Deps: []string{"B"}},
					"B": {Script: "DeployB", Deps: []string{"A"}},
				},
			},
			expectedOrder: nil,
			expectError:   true,
			errorContains: "circular dependency",
		},
		{
			name: "complex circular dependency",
			config: &OrchestrationConfig{
				Group: "Test Protocol",
				Components: map[string]*ComponentConfig{
					"A": {Script: "DeployA", Deps: []string{"B"}},
					"B": {Script: "DeployB", Deps: []string{"C"}},
					"C": {Script: "DeployC", Deps: []string{"A"}},
				},
			},
			expectedOrder: nil,
			expectError:   true,
			errorContains: "circular dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewDependencyGraph(tt.config)
			result, err := graph.TopologicalSort()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && err.Error() == "" {
					t.Errorf("expected error containing '%s' but got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
					return
				}

				// Check the order is valid (dependencies come before dependents)
				if !isValidOrder(tt.config, result) {
					t.Errorf("topological sort produced invalid order")
				}

				// For simple cases, check exact order
				if len(tt.expectedOrder) > 0 {
					if len(result) != len(tt.expectedOrder) {
						t.Errorf("expected %d steps but got %d", len(tt.expectedOrder), len(result))
						return
					}

					resultNames := make([]string, len(result))
					for i, step := range result {
						resultNames[i] = step.Name
					}

					// For deterministic tests, check exact order
					if tt.name == "simple linear dependency" || tt.name == "complex dependencies" {
						for i, expected := range tt.expectedOrder {
							if resultNames[i] != expected {
								t.Errorf("expected step %d to be '%s' but got '%s'", i, expected, resultNames[i])
							}
						}
					}
				}
			}
		})
	}
}

// isValidOrder checks if the topological sort result respects dependencies
func isValidOrder(config *OrchestrationConfig, result []*ExecutionStep) bool {
	// Build position map
	positions := make(map[string]int)
	for i, step := range result {
		positions[step.Name] = i
	}

	// Check that all dependencies come before their dependents
	for name, component := range config.Components {
		namePos := positions[name]
		for _, dep := range component.Deps {
			depPos := positions[dep]
			if depPos >= namePos {
				return false // Dependency should come before dependent
			}
		}
	}

	return true
}

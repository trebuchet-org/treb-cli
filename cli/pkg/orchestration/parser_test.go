package orchestration

import (
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		yamlData    string
		expectError bool
		errorMsg    string
		validate    func(*OrchestrationConfig) bool
	}{
		{
			name: "valid minimal configuration",
			yamlData: `
group: Test Protocol
components:
  Broker:
    script: DeployBroker
`,
			expectError: false,
			validate: func(config *OrchestrationConfig) bool {
				return config.Group == "Test Protocol" &&
					len(config.Components) == 1 &&
					config.Components["Broker"].Script == "DeployBroker"
			},
		},
		{
			name: "configuration with dependencies",
			yamlData: `
group: Mento Protocol
components:
  Broker:
    script: DeployBroker
  Tokens:
    script: DeployTokens
    deps: 
      - Broker
  Reserve:
    script: DeployReserve
    deps: 
      - Tokens
    env:
      INITIAL_BALANCE: "1000000"
      RESERVE_RATIO: "0.1"
`,
			expectError: false,
			validate: func(config *OrchestrationConfig) bool {
				if config.Group != "Mento Protocol" || len(config.Components) != 3 {
					return false
				}

				tokens := config.Components["Tokens"]
				if tokens.Script != "DeployTokens" || len(tokens.Deps) != 1 || tokens.Deps[0] != "Broker" {
					return false
				}

				reserve := config.Components["Reserve"]
				if reserve.Script != "DeployReserve" ||
					len(reserve.Deps) != 1 || reserve.Deps[0] != "Tokens" ||
					len(reserve.Env) != 2 ||
					reserve.Env["INITIAL_BALANCE"] != "1000000" ||
					reserve.Env["RESERVE_RATIO"] != "0.1" {
					return false
				}

				return true
			},
		},
		{
			name: "invalid yaml",
			yamlData: `
group: Test Protocol
components:
  Broker:
    script: DeployBroker
  - invalid structure
`,
			expectError: true,
		},
		{
			name: "missing group",
			yamlData: `
components:
  Broker:
    script: DeployBroker
`,
			expectError: true,
			errorMsg:    "group name is required",
		},
		{
			name: "circular dependency - detected during execution plan",
			yamlData: `
group: Test Protocol
components:
  A:
    script: DeployA
    deps: [B]
  B:
    script: DeployB
    deps: [A]
`,
			expectError: false, // Parsing succeeds, but execution plan creation will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			config, err := parser.Parse([]byte(tt.yamlData))

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" {
					if err.Error() != tt.errorMsg && !containsString(err.Error(), tt.errorMsg) {
						t.Errorf("expected error containing '%s' but got '%s'", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
					return
				}

				if tt.validate != nil && !tt.validate(config) {
					t.Errorf("validation failed for config: %+v", config)
				}
			}
		})
	}
}

func TestParser_CreateExecutionPlan(t *testing.T) {
	yamlData := `
group: Mento Protocol
components:
  Broker:
    script: DeployBroker
  Tokens:
    script: DeployTokens
    deps: [Broker]
  Reserve:
    script: DeployReserve
    deps: [Tokens]
    env:
      INITIAL_BALANCE: "1000000"
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlData))
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	plan, err := parser.CreateExecutionPlan(config)
	if err != nil {
		t.Fatalf("failed to create execution plan: %v", err)
	}

	if plan.Group != "Mento Protocol" {
		t.Errorf("expected group 'Mento Protocol' but got '%s'", plan.Group)
	}

	if len(plan.Components) != 3 {
		t.Errorf("expected 3 components but got %d", len(plan.Components))
	}

	// Check execution order
	expectedOrder := []string{"Broker", "Tokens", "Reserve"}
	for i, expected := range expectedOrder {
		if i >= len(plan.Components) {
			t.Errorf("missing component at position %d", i)
			continue
		}
		if plan.Components[i].Name != expected {
			t.Errorf("expected component %d to be '%s' but got '%s'", i, expected, plan.Components[i].Name)
		}
	}

	// Check that Reserve has environment variables
	reserveStep := plan.Components[2]
	if reserveStep.Name != "Reserve" {
		t.Errorf("expected third step to be 'Reserve'")
	}
	if len(reserveStep.Env) != 1 || reserveStep.Env["INITIAL_BALANCE"] != "1000000" {
		t.Errorf("expected Reserve to have INITIAL_BALANCE=1000000, got: %v", reserveStep.Env)
	}
}

func TestParser_CreateExecutionPlan_CircularDependency(t *testing.T) {
	yamlData := `
group: Test Protocol
components:
  A:
    script: DeployA
    deps: [B]
  B:
    script: DeployB
    deps: [A]
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlData))
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Creating execution plan should fail due to circular dependency
	plan, err := parser.CreateExecutionPlan(config)
	if err == nil {
		t.Fatalf("expected error due to circular dependency but got none")
	}

	if plan != nil {
		t.Errorf("expected plan to be nil when there's a circular dependency")
	}

	if !containsString(err.Error(), "circular dependency") {
		t.Errorf("expected error to mention 'circular dependency' but got: %s", err.Error())
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
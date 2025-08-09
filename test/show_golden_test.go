package integration_test

import (
	"testing"
	"strings"
)

func TestShowCommandGolden(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, ctx *TrebContext)
		args       []string
		goldenFile string
		expectErr  bool
	}{
		{
			name: "show_counter",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"show", "Counter"},
			goldenFile: "commands/show/counter.golden",
		},
		{
			name: "show_with_namespace",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"show", "Counter", "--namespace", "production"},
			goldenFile: "commands/show/counter_production.golden",
		},
		{
			name: "show_proxy",
			setup: func(t *testing.T, ctx *TrebContext) {
				// Skip proxy tests for now since proxy generation isn't implemented
				t.Skip("Proxy generation not yet implemented")
			},
			args:       []string{"show", "CounterProxy"},
			goldenFile: "commands/show/counter_proxy.golden",
		},
		{
			name:       "show_not_found",
			args:       []string{"show", "NonExistent"},
			goldenFile: "commands/show/not_found.golden",
			expectErr:  true,
		},
		{
			name: "show_by_address",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"show", "PLACEHOLDER_ADDRESS"},
			goldenFile: "commands/show/by_address.golden",
		},
	}

	for _, tt := range tests {
		test := tt // capture range variable
		IsolatedTest(t, test.name, func(t *testing.T, ctx *TrebContext) {
			if test.setup != nil {
				test.setup(t, ctx)
			}
			
			// For by_address test, we need to get the actual address first
			if test.name == "show_by_address" {
				// Get the actual address from list output
				output, err := ctx.treb("list", "--contract", "Counter")
				if err != nil {
					t.Fatalf("Failed to list contracts: %v\nOutput:\n%s", err, output)
				}
				
				// Extract first address from output
				lines := strings.Split(output, "\n")
				var address string
				for _, line := range lines {
					if strings.Contains(line, "0x") {
						// Extract address (40 hex chars after 0x)
						idx := strings.Index(line, "0x")
						if idx >= 0 && len(line) >= idx+42 {
							address = line[idx:idx+42]
							break
						}
					}
				}
				
				if address != "" {
					test.args[1] = address
				}
			}
			
			if test.expectErr {
				ctx.trebGoldenWithError(test.goldenFile, test.args...)
			} else {
				ctx.trebGolden(test.goldenFile, test.args...)
			}
		})
	}
}

// hasExistingDeployments checks if there are deployments in the registry
func hasExistingDeployments(t *testing.T) bool {
	t.Helper()
	
	ctx := NewTrebContext(t)
	output, err := ctx.treb("list")
	if err != nil {
		return false
	}
	
	return !strings.Contains(output, "No deployments found")
}
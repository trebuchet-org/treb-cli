package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommandGolden(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, ctx *TrebContext)
		args       []string
		goldenFile string
	}{
		{
			name:       "empty",
			args:       []string{"list"},
			goldenFile: "commands/list/empty.golden",
		},
		{
			name: "default_with_deployments",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list"},
			goldenFile: "commands/list/default.golden",
		},
		{
			name: "with_namespace",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list", "--namespace", "production"},
			goldenFile: "commands/list/with_namespace.golden",
		},
		{
			name: "with_chain",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list", "--chain", "31337"},
			goldenFile: "commands/list/with_chain.golden",
		},
		{
			name: "with_contract",
			setup: func(t *testing.T, ctx *TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list", "--contract", "Counter"},
			goldenFile: "commands/list/with_contract.golden",
		},
	}

	for _, tt := range tests {
		IsolatedTest(t, tt.name, func(t *testing.T, ctx *TrebContext) {
			if tt.setup != nil {
				tt.setup(t, ctx)
			}
			ctx.trebGolden(tt.goldenFile, tt.args...)
		})
	}
}

// setupTestDeployments creates test deployments for golden file testing
func setupTestDeployments(t *testing.T, ctx *TrebContext) {
	t.Helper()

	// First check if deployment script exists, if not generate it
	if _, err := os.Stat(filepath.Join(fixtureDir, "script/deploy/Counter.s.sol")); os.IsNotExist(err) {
		output, err := ctx.treb("gen", "deploy", "src/Counter.sol:Counter")
		if err != nil {
			t.Fatalf("Failed to generate Counter deployment script: %v\nOutput:\n%s", err, output)
		}
		t.Logf("Generated deployment script")
	}

	// Deploy Counter in default namespace
	t.Logf("Deploying Counter in default namespace")
	output, err := ctx.treb("run", "script/deploy/DeployCounter.s.sol")
	if err != nil {
		t.Fatalf("Failed to deploy Counter: %v\nOutput:\n%s", err, output)
	}
	if !strings.Contains(output, "Deployment Summary") && !strings.Contains(output, "0x") {
		t.Fatalf("Expected deployment output to contain deployment information, got:\n%s", output)
	}
	t.Logf("Deployed Counter in default namespace")

	// Deploy another with different namespace  
	ctx2 := ctx.WithNamespace("production")
	t.Logf("Deploying Counter in production namespace")
	output, err = ctx2.treb("run", "script/deploy/DeployCounter.s.sol", "--env", "label=prod")
	if err != nil {
		t.Fatalf("Failed to deploy Counter in production namespace: %v\nOutput:\n%s", err, output)
	}
	if !strings.Contains(output, "Deployment Summary") && !strings.Contains(output, "0x") {
		t.Fatalf("Expected deployment output to contain deployment information, got:\n%s", output)
	}
	t.Logf("Deployed Counter in production namespace")
	
	// List to verify deployments
	listOut, _ := ctx.treb("list")
	t.Logf("Deployments after setup:\n%s", listOut)
}
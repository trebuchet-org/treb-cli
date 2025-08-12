package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommandGolden(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, ctx *helpers.TrebContext)
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
			setup: func(t *testing.T, ctx *helpers.TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list"},
			goldenFile: "commands/list/default.golden",
		},
		{
			name: "with_namespace",
			setup: func(t *testing.T, ctx *helpers.TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list", "--namespace", "production"},
			goldenFile: "commands/list/with_namespace.golden",
		},
		{
			name: "with_chain",
			setup: func(t *testing.T, ctx *helpers.TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list", "--chain", "31337"},
			goldenFile: "commands/list/with_chain.golden",
		},
		{
			name: "with_contract",
			setup: func(t *testing.T, ctx *helpers.TrebContext) {
				setupTestDeployments(t, ctx)
			},
			args:       []string{"list", "--contract", "Counter"},
			goldenFile: "commands/list/with_contract.golden",
		},
	}

	for _, tt := range tests {
		helpers.IsolatedTest(t, tt.name, func(t *testing.T, ctx *helpers.TrebContext) {
			if tt.setup != nil {
				tt.setup(t, ctx)
			}

			TrebGolden(t, ctx, tt.goldenFile, tt.args...)
		})
	}
}

// setupTestDeployments creates test deployments for golden file testing
func setupTestDeployments(t *testing.T, ctx *helpers.TrebContext) {
	t.Helper()

	// First check if deployment script exists, if not generate it
	if _, err := os.Stat(filepath.Join(helpers.GetFixtureDir(), "script/deploy/Counter.s.sol")); os.IsNotExist(err) {
		output, err := ctx.Treb("gen", "deploy", "src/Counter.sol:Counter")
		if err != nil {
			t.Fatalf("Failed to generate Counter deployment script: %v\nOutput:\n%s", err, output)
		}
		t.Logf("Generated deployment script")
	}

	// Deploy Counter in default namespace
	t.Logf("Deploying Counter in default namespace")
	output, err := ctx.Treb("run", "script/deploy/DeployCounter.s.sol")
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
	output, err = ctx2.Treb("run", "script/deploy/DeployCounter.s.sol", "--env", "label=prod")
	if err != nil {
		t.Fatalf("Failed to deploy Counter in production namespace: %v\nOutput:\n%s", err, output)
	}
	if !strings.Contains(output, "Deployment Summary") && !strings.Contains(output, "0x") {
		t.Fatalf("Expected deployment output to contain deployment information, got:\n%s", output)
	}
	t.Logf("Deployed Counter in production namespace")

	// List to verify deployments
	listOut0, _ := ctx.Treb("list")
	listOut1, _ := ctx2.Treb("list")
	t.Logf("Deployments after setup:\n%s\n%s", listOut0, listOut1)

	// Also check registry files
	if registryBytes, err := os.ReadFile(filepath.Join(helpers.GetFixtureDir(), ".treb", "registry.json")); err == nil {
		t.Logf("Registry file content: %s", string(registryBytes))
	} else {
		t.Logf("Registry file error: %v", err)
	}
	if deploymentsBytes, err := os.ReadFile(filepath.Join(helpers.GetFixtureDir(), ".treb", "deployments.json")); err == nil {
		t.Logf("Deployments file size: %d bytes", len(deploymentsBytes))
	} else {
		t.Logf("Deployments file error: %v", err)
	}
}

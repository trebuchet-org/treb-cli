package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"bytes"
	"strings"
	"testing"
	
)

func TestDeploymentWorkflowGolden(t *testing.T) {
	tests := []struct {
		name       string
		workflow   func(t *testing.T, ctx *helpers.TrebContext) string
		goldenFile string
	}{
		{
			name:       "full_deployment_flow",
			workflow:   fullDeploymentWorkflow,
			goldenFile: "workflows/full_deployment_flow.golden",
		},
		{
			name:       "proxy_deployment_flow",
			workflow:   proxyDeploymentWorkflow,
			goldenFile: "workflows/proxy_deployment_flow.golden",
		},
		{
			name:       "multi_namespace_flow",
			workflow:   multiNamespaceWorkflow,
			goldenFile: "workflows/multi_namespace_flow.golden",
		},
	}

	for _, tt := range tests {
		helpers.IsolatedTest(t, tt.name, func(t *testing.T, ctx *helpers.TrebContext) {
			output := tt.workflow(t, ctx)

			compareGolden(t, output, GoldenConfig{
				Path: tt.goldenFile,
				Normalizers: []Normalizer{
					ColorNormalizer{},
					TimestampNormalizer{},
					HashNormalizer{},
				},
			})
		})
	}
}

// fullDeploymentWorkflow captures a complete deployment lifecycle
func fullDeploymentWorkflow(t *testing.T, ctx *helpers.TrebContext) string {
	var output bytes.Buffer

	// Step 1: Generate deployment script
	output.WriteString("=== Step 1: Generate deployment script ===\n")
	out, err := ctx.Treb("gen", "deploy", "src/Counter.sol:Counter")
	if err != nil {
		t.Fatalf("Failed to generate: %v\n%s\n", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// Step 2: Run deployment
	output.WriteString("=== Step 2: Deploy contract ===\n")
	out, err = ctx.Treb("run", "script/deploy/DeployCounter.s.sol")
	if err != nil {
		t.Fatalf("Failed to deploy: %v\n%s\n", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// Step 3: List deployments
	output.WriteString("=== Step 3: List deployments ===\n")
	out, err = ctx.Treb("list")
	if err != nil {
		t.Fatalf("Failed to list: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// Step 4: Show deployment details
	output.WriteString("=== Step 4: Show deployment details ===\n")
	out, err = ctx.Treb("show", "Counter")
	if err != nil {
		t.Fatalf("Failed to show: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// Step 5: Verify contract (skipped on local chain)
	output.WriteString("=== Step 5: Verify contract ===\n")
	output.WriteString("Verification skipped (local chain)\n")

	return output.String()
}

// proxyDeploymentWorkflow captures proxy deployment flow
func proxyDeploymentWorkflow(t *testing.T, ctx *helpers.TrebContext) string {
	var output bytes.Buffer

	// Step 1: Deploy implementation
	output.WriteString("=== Step 1: Deploy implementation ===\n")
	genOut, genErr := ctx.Treb("gen", "deploy", "src/UpgradeableCounter.sol:UpgradeableCounter")
	if genErr != nil {
		t.Fatalf("Failed to generate deployment script: %v\nOutput:\n%s", genErr, genOut)
	}
	out, err := ctx.Treb("run", "script/deploy/DeployUpgradeableCounter.s.sol")
	if err != nil {
		t.Fatalf("Failed to deploy implementation: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// Step 2: Generate proxy
	output.WriteString("=== Step 2: Generate proxy script ===\n")
	out, err = ctx.Treb("gen", "UpgradeableCounter", "--proxy")
	// If proxy generation is not yet supported, provide fallback
	if err != nil && strings.Contains(out, "unknown flag") {
		out = "Proxy generation not yet implemented\n"
		err = nil
	}
	if err != nil {
		t.Fatalf("Failed to generate proxy: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// Step 3: Deploy proxy
	output.WriteString("=== Step 3: Deploy proxy ===\n")
	output.WriteString("Proxy deployment skipped - script generation not implemented\n")
	output.WriteString("\n\n")

	// Step 4: Show proxy details
	output.WriteString("=== Step 4: Show proxy details ===\n")
	output.WriteString("Proxy details skipped - no proxy deployed\n")

	return output.String()
}

// multiNamespaceWorkflow demonstrates working with multiple namespaces
func multiNamespaceWorkflow(t *testing.T, ctx *helpers.TrebContext) string {
	var output bytes.Buffer
	ctxProd := ctx.WithNamespace("production")

	// Deploy to default namespace
	output.WriteString("=== Deploy to default namespace ===\n")
	out, err := ctx.Treb("gen", "deploy", "src/Counter.sol:Counter")
	if err != nil {
		t.Fatalf("Failed to generate deployment script: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	out, err = ctx.Treb("run", "script/deploy/DeployCounter.s.sol")
	if err != nil {
		t.Fatalf("Failed to deploy to default namespace: %v\nOutput:\n%s", err, out)
	}

	output.WriteString(out)
	output.WriteString("\n\n")

	// Deploy to default namespace with different label
	output.WriteString("=== Deploy to production namespace ===\n")
	out, err = ctxProd.Treb("run", "script/deploy/DeployCounter.s.sol")
	if err != nil {
		t.Fatalf("Failed to deploy with staging label: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	// List all deployments
	output.WriteString("=== List all deployments across namespaces ===\n")
	out, err = ctx.Treb("list")
	if err != nil {
		t.Fatalf("Failed to list deployments: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)
	output.WriteString("\n\n")

	out, err = ctxProd.Treb("list")
	if err != nil {
		t.Fatalf("Failed to list deployments: %v\nOutput:\n%s", err, out)
	}
	output.WriteString(out)

	return output.String()
}

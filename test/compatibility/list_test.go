package compatibility

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"strings"
	"testing"
)

func TestListCommandCompatibility(t *testing.T) {
	// Test that list command works the same in v1 and v2
	t.Run("list_empty", func(t *testing.T) {
		CompareOutputs(t, []string{"list"}, nil)
	})

	// Test list with specific flags
	t.Run("list_json_empty", func(t *testing.T) {
		CompareOutputs(t, []string{"list", "--json"}, nil)
	})
}

// TestListCommandV2Only tests v2-specific functionality
func TestListCommandV2Only(t *testing.T) {
	if !helpers.V2BinaryExists() {
		t.Skip("v2 binary not found")
	}

	helpers.IsolatedTestWithVersion(t, "list_empty", helpers.BinaryV2, func(t *testing.T, ctx *helpers.TrebContext) {
		output, err := ctx.Treb("list")
		if err != nil {
			t.Fatalf("list command failed: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(output, "No deployments found") {
			t.Errorf("Expected 'No deployments found', got: %s", output)
		}
	})
}

// TestListWithDeployments tests list functionality after deployments
// This test only runs with v1 since v2 doesn't have deploy commands yet
func TestListWithDeployments(t *testing.T) {
	helpers.IsolatedTestWithVersion(t, "list_with_counter", helpers.BinaryV1, func(t *testing.T, ctx *helpers.TrebContext) {
		// Generate and deploy a contract
		output, err := ctx.Treb("gen", "deploy", "src/Counter.sol:Counter", "--non-interactive")
		if err != nil {
			t.Fatalf("gen deploy failed: %v\nOutput: %s", err, output)
		}

		output, err = ctx.Treb("run", "script/deploy/DeployCounter.s.sol")
		if err != nil {
			t.Fatalf("deploy failed: %v\nOutput: %s", err, output)
		}

		// Now list should show the deployment
		output, err = ctx.Treb("list")
		if err != nil {
			t.Fatalf("list command failed: %v\nOutput: %s", err, output)
		}

		// Check for expected content
		if !strings.Contains(output, "Counter") {
			t.Errorf("Expected 'Counter' in output, got: %s", output)
		}
		if !strings.Contains(output, "anvil-31337") {
			t.Errorf("Expected 'anvil-31337' in output, got: %s", output)
		}
	})
}


package compatibility

import (
	"testing"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

// CompatibilityTest runs a test against both v1 and v2 binaries
type CompatibilityTest struct {
	Name      string
	TestFunc  func(t *testing.T, v1 *helpers.TrebContext, v2 *helpers.TrebContext)
	SkipV2    bool // Skip v2 if commands not yet migrated
}

// RunCompatibilityTest runs a test with both v1 and v2 contexts
func RunCompatibilityTest(t *testing.T, test CompatibilityTest) {
	helpers.IsolatedTest(t, test.Name, func(t *testing.T, _ *helpers.TrebContext) {
		// Create v1 context
		v1Ctx := helpers.NewTrebContext(t, helpers.BinaryV1)
		
		// Create v2 context if not skipped
		var v2Ctx *helpers.TrebContext
		if !test.SkipV2 {
			if !helpers.V2BinaryExists() {
				t.Skip("v2 binary not found, run 'make build-v2' first")
			}
			v2Ctx = helpers.NewTrebContext(t, helpers.BinaryV2)
		}
		
		// Run the test with both contexts
		test.TestFunc(t, v1Ctx, v2Ctx)
	})
}

// RunCompatibilityTests runs multiple compatibility tests
func RunCompatibilityTests(t *testing.T, tests []CompatibilityTest) {
	for _, test := range tests {
		RunCompatibilityTest(t, test)
	}
}

// CompareOutputs runs a command on both v1 and v2 and compares outputs
func CompareOutputs(t *testing.T, args []string, normalizer func(string) string) {
	var v1Output, v2Output string
	
	// Run with v1
	helpers.IsolatedTestWithVersion(t, "v1", helpers.BinaryV1, func(t *testing.T, ctx *helpers.TrebContext) {
		output, err := ctx.Treb(args...)
		if err != nil {
			t.Fatalf("v1 command failed: %v\nOutput: %s", err, output)
		}
		v1Output = output
	})
	
	// Run with v2
	helpers.IsolatedTestWithVersion(t, "v2", helpers.BinaryV2, func(t *testing.T, ctx *helpers.TrebContext) {
		output, err := ctx.Treb(args...)
		if err != nil {
			t.Fatalf("v2 command failed: %v\nOutput: %s", err, output)
		}
		v2Output = output
	})
	
	// Normalize if needed
	if normalizer != nil {
		v1Output = normalizer(v1Output)
		v2Output = normalizer(v2Output)
	}
	
	// Compare
	if v1Output != v2Output {
		t.Errorf("Output mismatch between v1 and v2:\nv1:\n%s\nv2:\n%s", v1Output, v2Output)
	}
}
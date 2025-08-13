package compatibility

import (
	"bytes"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"testing"
)

// CompatibilityTest runs a test against both v1 and v2 binaries
type CompatibilityTest struct {
	Name        string
	PreSetup    func(t *testing.T, ctx *helpers.TrebContext)
	SetupCmds   [][]string
	PostSetup   func(t *testing.T, ctx *helpers.TrebContext)
	TestCmds    [][]string
	ExpectErr   bool
	Normalizers []helpers.Normalizer
	SetupCtx    *helpers.TrebContext
}

// RunCompatibilityTest runs a test with both v1 and v2 contexts
func RunCompatibilityTest(t *testing.T, test CompatibilityTest) {
	helpers.IsolatedTest(t, test.Name, func(t *testing.T, _ *helpers.TrebContext) {
		v1Output := runTest(t, helpers.BinaryV1, test)
		v2Output := runTest(t, helpers.BinaryV2, test)

		if test.Normalizers == nil {
			test.Normalizers = helpers.GetDefaultNormalizers()
		}

		v1OutputNorm := helpers.Normalize(v1Output, helpers.GetDefaultNormalizers())
		v2OutputNorm := helpers.Normalize(v2Output, helpers.GetDefaultNormalizers())

		if diff := cmp.Diff(v1OutputNorm, v2OutputNorm); diff != "" {
			t.Errorf("Output mismatch (-want +got):\n%s", diff)
		}
	})
}

// RunCompatibilityTests runs multiple compatibility tests
func RunCompatibilityTests(t *testing.T, tests []CompatibilityTest) {
	for _, test := range tests {
		RunCompatibilityTest(t, test)
	}
}

func runTest(t *testing.T, version helpers.BinaryVersion, test CompatibilityTest) string {
	var output bytes.Buffer
	helpers.IsolatedTestWithVersion(t, string(version), version, func(t *testing.T, ctx *helpers.TrebContext) {
		// Run setup if provided
		if test.PreSetup != nil {
			t.Logf("Running pre-setup function")
			test.PreSetup(t, ctx)
		}

		setupCtx := ctx
		if test.SetupCtx != nil {
			setupCtx = test.SetupCtx
		}

		// Run setup commands if provided
		if test.SetupCmds != nil {
			t.Logf("Running setup commands")
			for _, cmd := range test.SetupCmds {
				if output, err := setupCtx.Treb(cmd...); err != nil {
					t.Fatalf("Failed setup command %v: %v\nOutput:\n%s", cmd, err, output)
				}
			}
		}

		if test.PostSetup != nil {
			t.Logf("Running post-setup function")
			test.PostSetup(t, ctx)
		}

		// Run the command
		if test.TestCmds == nil {
			t.Skip("No test commands provided")
		}

		var err error
		for i, cmd := range test.TestCmds {
			cmdOut, cmdErr := ctx.Treb(cmd...)
			output.WriteString(fmt.Sprintf("=== cmd %d: %v ===\n", i, cmd))
			output.WriteString(cmdOut)
			output.WriteString("\n\n")
			if cmdErr != nil {
				err = cmdErr
				break
			}
		}

		if test.ExpectErr {
			assert.Error(t, err, "Expected command to fail")
		} else {
			assert.NoError(t, err, "Command failed: %v\nOutput: %s", err, output)
		}
	})
	return output.String()
}

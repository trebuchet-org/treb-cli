package compatibility

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
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

type testOutput struct {
	testCmdsOutput   string
	deploymentsJSON  string
	transactionsJSON string
	safeTxsJSON      string
	registryJSON     string
}

// RunCompatibilityTest runs a test with both v1 and v2 contexts
func RunCompatibilityTest(t *testing.T, test CompatibilityTest) {
	if test.Normalizers == nil {
		test.Normalizers = helpers.GetDefaultNormalizers()
	}

	helpers.IsolatedTest(t, test.Name, func(t *testing.T, _ *helpers.TrebContext) {
		v1Output := runTest(t, helpers.BinaryV1, test)
		v2Output := runTest(t, helpers.BinaryV2, test)

		if diff := cmp.Diff(v1Output.testCmdsOutput, v2Output.testCmdsOutput); diff != "" {
			t.Errorf("Output mismatch on Test Commands output (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(v1Output.deploymentsJSON, v2Output.deploymentsJSON); diff != "" {
			t.Errorf("deployments.json mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(v1Output.transactionsJSON, v2Output.transactionsJSON); diff != "" {
			t.Errorf("transactions.json mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(v1Output.safeTxsJSON, v2Output.safeTxsJSON); diff != "" {
			t.Errorf("safe-txs.json mismatch (-want +got):\n%s", diff)
		}
	})
}

// RunCompatibilityTests runs multiple compatibility tests
func RunCompatibilityTests(t *testing.T, tests []CompatibilityTest) {
	for _, test := range tests {
		RunCompatibilityTest(t, test)
	}
}

func runTest(t *testing.T, version helpers.BinaryVersion, test CompatibilityTest) (testOutput testOutput) {
	helpers.IsolatedTestWithVersion(t, string(version), version, func(t *testing.T, ctx *helpers.TrebContext) {
		var output bytes.Buffer
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

		testOutput = createTestOutput(output.String(), test)
	})
	return
}

func createTestOutput(testCmdsOutput string, test CompatibilityTest) (output testOutput) {
	var err error
	var data []byte

	output.testCmdsOutput = helpers.Normalize(testCmdsOutput, test.Normalizers)
	if data, err = os.ReadFile(path.Join(helpers.GetFixtureDir(), ".treb", "deployments.json")); err != nil {
		panic(err)
	}
	output.deploymentsJSON = helpers.Normalize(string(data), test.Normalizers)
	if data, err = os.ReadFile(path.Join(helpers.GetFixtureDir(), ".treb", "transactions.json")); err != nil {
		panic(err)
	}
	output.transactionsJSON = helpers.Normalize(string(data), test.Normalizers)
	if data, err = os.ReadFile(path.Join(helpers.GetFixtureDir(), ".treb", "safe-txs.json")); err != nil {
		panic(err)
	}
	output.safeTxsJSON = helpers.Normalize(string(data), test.Normalizers)
	if data, err = os.ReadFile(path.Join(helpers.GetFixtureDir(), ".treb", "registry.json")); err != nil {
		panic(err)
	}
	output.registryJSON = helpers.Normalize(string(data), test.Normalizers)
	return
}

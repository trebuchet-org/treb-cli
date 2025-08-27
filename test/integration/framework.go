package integration

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

type IntegrationTest struct {
	Name            string
	PreSetup        func(t *testing.T, ctx *helpers.TestContext)
	SetupCmds       [][]string
	PostSetup       func(t *testing.T, ctx *helpers.TestContext)
	TestCmds        [][]string
	PostTest        func(t *testing.T, ctx *helpers.TestContext, output string)
	ExpectErr       bool
	Normalizers     []helpers.Normalizer
	OutputArtifacts []string
	SkipGolden      bool
}

var DefaultOutputArtifacs = []string{
	".treb/deployments.json",
	".treb/registry.json",
	".treb/safe-txs.json",
}

type testOutput struct {
	testCmdsStdout string
	artifacts      map[string]string
}

// RunIntegrationTests runs multiple integration tests
func RunIntegrationTests(t *testing.T, tests []IntegrationTest) {
	for _, test := range tests {
		RunIntegrationTest(t, test)
	}
}

// RunIntegrationTest runs a test with both v1 and v2 contexts
func RunIntegrationTest(t *testing.T, test IntegrationTest) {
	if test.Normalizers == nil {
		test.Normalizers = helpers.GetDefaultNormalizers()
	}
	if test.OutputArtifacts == nil {
		test.OutputArtifacts = DefaultOutputArtifacs
	}

	helpers.IsolatedTest(t, test.Name, func(t *testing.T, ctx *helpers.TestContext) {
		output := runTest(t, test, ctx)

		if !test.SkipGolden {
			compareOutput(t, test, output.testCmdsStdout, "Command Output", "commands.golden")
			for path, artifact := range output.artifacts {
				compareOutput(t, test, artifact, path, filepath.Base(path)+".golden")
			}
		}

		if test.PostTest != nil {
			test.PostTest(t, ctx, output.testCmdsStdout)
		}
	})
}

func runTest(t *testing.T, test IntegrationTest, ctx *helpers.TestContext) testOutput {
	var output bytes.Buffer
	// Run setup if provided
	if test.PreSetup != nil {
		t.Logf("Running pre-setup function")
		test.PreSetup(t, ctx)
	}

	// Run setup commands if provided
	if test.SetupCmds != nil {
		t.Logf("Running setup commands")
		for _, cmd := range test.SetupCmds {
			if cmdOutput, err := ctx.TrebContext.Treb(cmd...); err != nil {
				t.Fatalf("Failed setup command %v: %v\nOutput:\n%s", cmd, err, cmdOutput)
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
		cmdOut, cmdErr := ctx.TrebContext.Treb(cmd...)
		output.WriteString(fmt.Sprintf("=== cmd %d: %v ===\n", i, cmd))
		output.WriteString(cmdOut)
		output.WriteString("\n\n")
		if cmdErr != nil {
			err = cmdErr
			break
		}
	}

	// Check if error is expected for this version
	if test.ExpectErr {
		assert.Error(t, err, "Expected command to fail")
	} else {
		assert.NoError(t, err, "Command failed: %v\nOutput: %s", err, output)
	}

	return createTestOutput(t, output.String(), test, ctx.TrebContext)
}

func createTestOutput(t *testing.T, testCmdsStdout string, test IntegrationTest, ctx *helpers.TrebContext) (output testOutput) {
	output.testCmdsStdout = helpers.Normalize(testCmdsStdout, test.Normalizers)
	output.artifacts = make(map[string]string)

	var err error
	var data []byte

	// Use the test context's working directory instead of GetFixtureDir()
	workDir := ctx.GetWorkDir()

	for _, path := range test.OutputArtifacts {
		if data, err = os.ReadFile(filepath.Join(workDir, path)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				data = []byte{}
			} else {
				panic(err)
			}
		}
		output.artifacts[path] = helpers.Normalize(string(data), test.Normalizers)
		if helpers.IsDebugEnabled() {
			t.Logf("Reading artifact at %s:\n---\n%s\n---\n", path, data)
		}
	}
	return
}

func compareOutput(t *testing.T, test IntegrationTest, output, displayName, goldenFile string) {
	goldenPath := helpers.GoldenPath(filepath.Join("integration", t.Name(), goldenFile))
	if helpers.ShouldUpdateGolden() {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("Failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(output), 0644); err != nil {
			t.Fatal(err)
		}
	} else {
		// Lets compare golden files
		var golden []byte
		var err error
		golden, err = os.ReadFile(goldenPath)
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Golden file missing, run with -treb.updategolden")
			golden = []byte{}
		}
		// normalize := cmp.Transformer("normalize", func(s string) string { return norm(s) })
		if diff := cmp.Diff(string(golden), output); diff != "" {
			t.Errorf("Diff on %s (v1) (-golden +output):\n%s", displayName, diff)
		}
	}
}

func s(cmd string) []string {
	return strings.Split(cmd, " ")
}

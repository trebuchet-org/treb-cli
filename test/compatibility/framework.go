package compatibility

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

// ErrorExpectation specifies which versions should error
type ErrorExpectation int

const (
	NoError ErrorExpectation = iota
	ErrorOnlyV1
	ErrorOnlyV2
	ErrorBoth
)

// CompatibilityTest runs a test against both v1 and v2 binaries
type CompatibilityTest struct {
	Name            string
	PreSetup        func(t *testing.T, ctx *helpers.TrebContext)
	SetupCmds       [][]string
	PostSetup       func(t *testing.T, ctx *helpers.TrebContext)
	TestCmds        [][]string
	ExpectErr       ErrorExpectation
	ExpectDiff      bool
	Normalizers     []helpers.Normalizer
	OutputArtifacts []string
}

var DefaultOutputArtifacs = []string{
	".treb/deployments.json",
	".treb/registry.json",
	".treb/safe-txs.json",
}

// ExpectsError returns true if the test expects an error for the given context
func (test CompatibilityTest) ExpectsError(ctx *helpers.TrebContext) bool {
	switch test.ExpectErr {
	case NoError:
		return false
	case ErrorOnlyV1:
		return ctx.BinaryVersion == helpers.BinaryV1
	case ErrorOnlyV2:
		return ctx.BinaryVersion == helpers.BinaryV2
	case ErrorBoth:
		return true
	default:
		return false
	}
}

type testOutput struct {
	testCmdsStdout string
	artifacts      map[string]string
}

// RunCompatibilityTest runs a test with both v1 and v2 contexts
func RunCompatibilityTest(t *testing.T, test CompatibilityTest) {
	if test.Normalizers == nil {
		test.Normalizers = helpers.GetDefaultNormalizers()
	}
	if test.OutputArtifacts == nil {
		test.OutputArtifacts = DefaultOutputArtifacs
	}

	// Don't use IsolatedTest here because it calls t.Parallel() which would
	// cause this test to return before subtests complete
	t.Run(test.Name, func(t *testing.T) {
		t.Parallel()
		// Store outputs from both versions
		var v1Output, v2Output testOutput

		// Run v1 test
		v1Output = runTest(t, helpers.BinaryV1, test)
		v2Output = runTest(t, helpers.BinaryV2, test)

		// Run v2 test
		// At this point, Go's testing framework ensures both subtests have completed
		// before continuing with the parent test
		compareOutput(t, test, v1Output.testCmdsStdout, v2Output.testCmdsStdout, "Test Commands", "commands.golden")
		for path, artifact := range v1Output.artifacts {
			compareOutput(t, test, artifact, v2Output.artifacts[path], path, filepath.Base(path)+".golden")
		}
	})
}

// RunCompatibilityTests runs multiple compatibility tests
func RunCompatibilityTests(t *testing.T, tests []CompatibilityTest) {
	for _, test := range tests {
		RunCompatibilityTest(t, test)
	}
}

func runTest(t *testing.T, version helpers.BinaryVersion, test CompatibilityTest) testOutput {
	var tOutput testOutput
	helpers.IsolatedTestWithVersion(t, string(version), version, func(t *testing.T, ctx *helpers.TrebContext) {
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
				if cmdOutput, err := ctx.Treb(cmd...); err != nil {
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
			cmdOut, cmdErr := ctx.Treb(cmd...)
			output.WriteString(fmt.Sprintf("=== cmd %d: %v ===\n", i, cmd))
			output.WriteString(cmdOut)
			output.WriteString("\n\n")
			if cmdErr != nil {
				err = cmdErr
				break
			}
		}

		// Check if error is expected for this version
		if test.ExpectsError(ctx) {
			assert.Error(t, err, "Expected command to fail")
		} else {
			assert.NoError(t, err, "Command failed: %v\nOutput: %s", err, output)
		}

		tOutput = createTestOutput(t, output.String(), test, ctx)

		// Log work directory if skip cleanup is enabled
		if helpers.ShouldSkipCleanup() {
			t.Logf("🔍 Test work directory for %s: %s", version, ctx.GetWorkDir())
		}
	})
	return tOutput
}

func createTestOutput(t *testing.T, testCmdsStdout string, test CompatibilityTest, ctx *helpers.TrebContext) (output testOutput) {
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

func compareOutput(t *testing.T, test CompatibilityTest, v1Output, v2Output, displayName, goldenFile string) {
	if diff := cmp.Diff(v1Output, v2Output); diff != "" {
		goldenPath := helpers.GoldenPath(filepath.Join("compatibility", t.Name(), goldenFile))
		if test.ExpectDiff {
			if helpers.ShouldUpdateGolden() {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
					t.Fatalf("Failed to create golden directory: %v", err)
				}
				if err := os.WriteFile(goldenPath, []byte(diff), 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				expected, err := os.ReadFile(goldenPath)
				if err == os.ErrNotExist {
					expected = []byte{}
				}
				normalize := cmp.Transformer("normalize", func(s string) string { return norm(s) })
				if diffDiff := cmp.Diff(string(expected), diff, normalize); diffDiff != "" {
					t.Errorf("Diff on %s (-v1 +v2):\n%s\nExpected Diff (-v1 +v2):\n%s\nDiff Diff (-golden +test):\n%s\n", displayName, diff, string(expected), diffDiff)
				}
			}
		} else {
			t.Errorf("Diff on %s (-v1 +v2):\n%s", displayName, diff)
		}
	}
}

func norm(s string) string {
	// unify line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")

	// map NBSP and friends to regular space
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u00A0', '\u2007', '\u202F': // NBSP, figure space, narrow NBSP
			return ' '
		default:
			return r
		}
	}, s)

	// trim trailing spaces/tabs on each line
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")

}

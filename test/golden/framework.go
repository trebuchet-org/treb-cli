package golden

import (
	"bytes"
	"fmt"
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// GoldenTest represents a golden file test
type GoldenTest struct {
	Name        string
	SetupCmds   [][]string
	Setup       func(t *testing.T, ctx *helpers.TrebContext)
	TestCmds    [][]string
	GoldenFile  string
	ExpectErr   bool
	Normalizers []helpers.Normalizer
}

// GoldenConfig configures golden file testing
type GoldenConfig struct {
	Path        string               // Path to golden file relative to testdata/golden/
	Normalizers []helpers.Normalizer // Output normalizers to apply
	Update      bool                 // Whether to update the golden file
}

// RunGoldenTest runs a golden test with the binary version from environment
func RunGoldenTest(t *testing.T, test GoldenTest) {
	version := helpers.GetBinaryVersionFromEnv()
	goldenFile := test.GoldenFile

	helpers.IsolatedTestWithVersion(t, test.Name, version, func(t *testing.T, ctx *helpers.TrebContext) {
		// Run setup if provided
		if test.Setup != nil {
			t.Logf("Running setup function")
			test.Setup(t, ctx)
		}

		// Run setup commands if provided
		if test.SetupCmds != nil {
			t.Logf("Running setup commands")
			for _, cmd := range test.SetupCmds {
				if output, err := ctx.Treb(cmd...); err != nil {
					t.Fatalf("Failed setup command %v: %v\nOutput:\n%s", cmd, err, output)
				}
			}
		}

		// Run the command
		if test.TestCmds == nil {
			t.Skip("No test commands provided")
		}

		var err error
		var output bytes.Buffer

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

		// Check error expectation
		if test.ExpectErr {
			assert.Error(t, err, "Expected command to fail")
		} else {
			assert.NoError(t, err, "Command failed: %v\nOutput: %s", err, output)
		}

		// Compare with golden file
		compareGolden(t, output.String(), GoldenConfig{
			Path:        goldenFile,
			Normalizers: helpers.GetDefaultNormalizers(),
		})
	})
}

func RunGoldenTests(t *testing.T, tests []GoldenTest) {
	for _, test := range tests {
		RunGoldenTest(t, test)
	}
}

// compareGolden compares output with golden file
func compareGolden(t *testing.T, output string, config GoldenConfig) {
	t.Helper()

	// Apply normalizers
	normalized := helpers.Normalize(output, config.Normalizers)

	// Get the test directory (parent of parent of current fixture directory)
	wd, _ := os.Getwd()
	testDir := filepath.Dir(filepath.Dir(wd))
	goldenPath := filepath.Join(testDir, "testdata", "golden", config.Path)

	// Update mode
	if os.Getenv("UPDATE_GOLDEN") == "true" || config.Update {
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("Failed to create golden directory: %v", err)
		}
		// Write golden file
		if err := os.WriteFile(goldenPath, []byte(normalized), 0644); err != nil {
			t.Fatalf("Failed to write golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read expected output
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Golden file does not exist: %s\nRun with UPDATE_GOLDEN=true to create it", goldenPath)
		}
		t.Fatalf("Failed to read golden file: %v", err)
	}

	expectedStr := strings.TrimSpace(string(expected))

	// Always ensure a trailing newline for consistency on expected too
	if expectedStr != "" {
		expectedStr = expectedStr + "\n"
	}

	// Compare
	if diff := cmp.Diff(expectedStr, normalized); diff != "" {
		t.Errorf("Output mismatch (-want +got):\n%s", diff)

		// Save actual output for debugging
		actualPath := goldenPath + ".actual"
		if err := os.WriteFile(actualPath, []byte(normalized), 0644); err == nil {
			t.Logf("Actual output saved to: %s", actualPath)
		}
	}
}

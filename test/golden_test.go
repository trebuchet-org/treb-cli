package integration_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

// GoldenConfig configures golden file testing
type GoldenConfig struct {
	Path        string       // Path to golden file relative to testdata/golden/
	Normalizers []Normalizer // Output normalizers to apply
	Update      bool         // Whether to update the golden file
}

// Normalizer processes output to remove dynamic content
type Normalizer interface {
	Normalize(output string) string
}

// TimestampNormalizer replaces timestamps with placeholders
type TimestampNormalizer struct{}

func (n TimestampNormalizer) Normalize(output string) string {
	// ISO timestamps: 2024-08-09T14:30:45Z
	output = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z?`).ReplaceAllString(output, "<TIMESTAMP>")

	// Standard timestamps: 2024-08-09 14:30:45
	output = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`).ReplaceAllString(output, "<TIMESTAMP>")

	// Relative times: "2 minutes ago", "1 hour ago"
	output = regexp.MustCompile(`\d+ \w+ ago`).ReplaceAllString(output, "<TIME_AGO>")

	// Unix timestamps (10-13 digits)
	output = regexp.MustCompile(`\b\d{10,13}\b`).ReplaceAllString(output, "<UNIX_TIME>")

	return output
}

// AddressNormalizer replaces Ethereum addresses
type AddressNormalizer struct{}

func (n AddressNormalizer) Normalize(output string) string {
	return regexp.MustCompile(`0x[a-fA-F0-9]{40}`).ReplaceAllString(output, "0x<ADDRESS>")
}

// HashNormalizer replaces transaction hashes
type HashNormalizer struct{}

func (n HashNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Tx: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Tx: 0x<HASH>")
	output = regexp.MustCompile(`Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Hash: 0x<HASH>")
	return output
}

// BlockNumberNormalizer replaces block numbers
type BlockNumberNormalizer struct{}

func (n BlockNumberNormalizer) Normalize(output string) string {
	// Block numbers in various formats
	output = regexp.MustCompile(`block:\s*\d+`).ReplaceAllString(output, "block: <BLOCK>")
	output = regexp.MustCompile(`"blockNumber":\s*\d+`).ReplaceAllString(output, `"blockNumber": <BLOCK>`)
	output = regexp.MustCompile(`#\d+`).ReplaceAllString(output, "#<BLOCK>")

	return output
}

// GasNormalizer replaces gas values
type GasNormalizer struct{}

func (n GasNormalizer) Normalize(output string) string {
	// Gas values in various formats
	output = regexp.MustCompile(`gas:\s*\d+`).ReplaceAllString(output, "gas: <GAS>")
	output = regexp.MustCompile(`"gas":\s*\d+`).ReplaceAllString(output, `"gas": <GAS>`)
	output = regexp.MustCompile(`"gasUsed":\s*\d+`).ReplaceAllString(output, `"gasUsed": <GAS>`)

	return output
}

// ColorNormalizer removes ANSI color codes
type ColorNormalizer struct{}

func (n ColorNormalizer) Normalize(output string) string {
	// Remove ANSI escape codes
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(output, "")
}

// PathNormalizer makes paths relative to project root
type PathNormalizer struct{}

func (n PathNormalizer) Normalize(output string) string {
	// Get absolute path to fixture dir
	fixtureAbs, _ := filepath.Abs(fixtureDir)

	// Replace absolute paths with relative ones
	output = strings.ReplaceAll(output, fixtureAbs, ".")

	// Normalize path separators
	output = strings.ReplaceAll(output, "\\", "/")

	return output
}

// VersionNormalizer replaces version-related strings
type VersionNormalizer struct{}

func (n VersionNormalizer) Normalize(output string) string {
	// Treb version: v1.0.0-beta.1-95-g6a2e70e
	output = regexp.MustCompile(`v\d+\.\d+\.\d+(-[a-zA-Z0-9\.\-]+)?`).ReplaceAllString(output, "v<VERSION>")

	// Git commit hashes (7-40 chars)
	output = regexp.MustCompile(`commit:\s*[a-f0-9]{7,40}`).ReplaceAllString(output, "commit: <COMMIT>")

	// Build timestamps: built: 2025-08-11 09:10:03 UTC
	output = regexp.MustCompile(`built:\s*\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} UTC`).ReplaceAllString(output, "built: <BUILD_TIME>")

	return output
}

// GitCommitNormalizer handles git commit hashes in deployment output
type GitCommitNormalizer struct{}

func (n GitCommitNormalizer) Normalize(output string) string {
	// Git Commit: 6a2e70eb854f6aaeaccfd4de3b81556cd372a124
	output = regexp.MustCompile(`Git Commit:\s*[a-f0-9]{40}`).ReplaceAllString(output, "Git Commit: <GIT_COMMIT>")
	// Also handle short hashes
	output = regexp.MustCompile(`Git Commit:\s*[a-f0-9]{7,39}`).ReplaceAllString(output, "Git Commit: <GIT_COMMIT>")
	return output
}

// compareGolden compares output with golden file
func compareGolden(t *testing.T, output string, config GoldenConfig) {
	t.Helper()

	// Apply normalizers
	normalizedOutput := output
	for _, normalizer := range config.Normalizers {
		normalizedOutput = normalizer.Normalize(normalizedOutput)
	}

	// Construct golden file path relative to the test directory, not fixture directory
	// Since tests run from testdata/project, we need to go up two levels
	goldenPath := filepath.Join("..", "..", "testdata", "golden", config.Path)

	// Update golden file if requested
	if config.Update || os.Getenv("UPDATE_GOLDEN") == "true" {
		// Create directory if needed
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create golden directory: %v", err)
		}

		// Write normalized output
		if err := os.WriteFile(goldenPath, []byte(normalizedOutput), 0644); err != nil {
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

	// Compare
	if diff := cmp.Diff(string(expected), normalizedOutput); diff != "" {
		t.Errorf("Output mismatch (-want +got):\n%s", diff)

		// Save actual output for debugging
		actualPath := goldenPath + ".actual"
		if err := os.WriteFile(actualPath, []byte(normalizedOutput), 0644); err == nil {
			t.Logf("Actual output saved to: %s", actualPath)
		}
	}
}

// TrebContextGolden extends TrebContext with golden file support
func (tc *TrebContext) trebGolden(goldenPath string, args ...string) {
	tc.t.Helper()

	output, err := tc.treb(args...)
	if err != nil {
		tc.t.Fatalf("Command failed unexpectedly: %v\nArgs: %v\nOutput:\n%s", err, args, output)
	}

	compareGolden(tc.t, output, GoldenConfig{
		Path: goldenPath,
		Normalizers: []Normalizer{
			ColorNormalizer{},
			TimestampNormalizer{},
			VersionNormalizer{},
			GitCommitNormalizer{},
			HashNormalizer{},
			// AddressNormalizer{},
			// PathNormalizer{},
			// BlockNumberNormalizer{},
			// GasNormalizer{},
		},
	})
}

// trebGoldenWithError runs a command expecting an error and compares output
func (tc *TrebContext) trebGoldenWithError(goldenPath string, args ...string) {
	tc.t.Helper()

	output, err := tc.treb(args...)
	require.Error(tc.t, err)

	compareGolden(tc.t, output, GoldenConfig{
		Path: goldenPath,
		Normalizers: []Normalizer{
			ColorNormalizer{},
			TimestampNormalizer{},
			VersionNormalizer{},
			GitCommitNormalizer{},
			HashNormalizer{},
			// AddressNormalizer{},
			// HashNormalizer{},
			// PathNormalizer{},
		},
	})
}

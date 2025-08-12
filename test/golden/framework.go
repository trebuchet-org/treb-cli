package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GoldenTest represents a golden file test
type GoldenTest struct {
	Name       string
	Setup      func(t *testing.T)
	Args       []string
	GoldenFile string
	ExpectErr  bool
	Normalizer func(string) string
}

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
	output = regexp.MustCompile(`tx 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "tx 0x<HASH>")
	output = regexp.MustCompile(`transaction 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "transaction 0x<HASH>")
	return output
}

// ColorNormalizer removes ANSI color codes
type ColorNormalizer struct{}

func (n ColorNormalizer) Normalize(output string) string {
	// Remove ANSI escape sequences (color codes)
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(output, "")
}

// GitCommitNormalizer replaces git commit hashes
type GitCommitNormalizer struct{}

func (n GitCommitNormalizer) Normalize(output string) string {
	return regexp.MustCompile(`[a-f0-9]{7,40}`).ReplaceAllString(output, "<COMMIT>")
}

// TargetedGitCommitNormalizer only replaces git commits in specific contexts
type TargetedGitCommitNormalizer struct{}

func (n TargetedGitCommitNormalizer) Normalize(output string) string {
	// Only replace git commits in specific contexts
	// Git Commit field in show output (full 40 char hash)
	output = regexp.MustCompile(`Git Commit: [a-f0-9]{40}`).ReplaceAllString(output, "Git Commit: <GIT_COMMIT>")
	// Commit in version output (7 char short hash)
	output = regexp.MustCompile(`commit: [a-f0-9]{7}`).ReplaceAllString(output, "commit: <COMMIT>")
	return output
}

// TargetedHashNormalizer only replaces hashes in specific contexts
type TargetedHashNormalizer struct{}

func (n TargetedHashNormalizer) Normalize(output string) string {
	// Transaction hashes in specific contexts
	output = regexp.MustCompile(`Tx: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Tx: 0x<HASH>")
	output = regexp.MustCompile(`Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Hash: 0x<HASH>")
	// Init code hash and bytecode hash in show output
	output = regexp.MustCompile(`Init Code Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Init Code Hash: 0x<HASH>")
	output = regexp.MustCompile(`Bytecode Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Bytecode Hash: 0x<HASH>")
	return output
}

// BlockNumberNormalizer replaces block numbers
type BlockNumberNormalizer struct{}

func (n BlockNumberNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Block: \d+`).ReplaceAllString(output, "Block: <BLOCK>")
	output = regexp.MustCompile(`block \d+`).ReplaceAllString(output, "block <BLOCK>")
	return output
}

// GasNormalizer replaces gas values
type GasNormalizer struct{}

func (n GasNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Gas: \d+`).ReplaceAllString(output, "Gas: <GAS>")
	output = regexp.MustCompile(`gas_used: \d+`).ReplaceAllString(output, "gas_used: <GAS>")
	return output
}

// PathNormalizer replaces absolute paths with relative ones
type PathNormalizer struct{}

func (n PathNormalizer) Normalize(output string) string {
	// Get absolute path to fixture dir
	fixtureAbs, _ := filepath.Abs(helpers.GetFixtureDir())

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

	// Git commit in version strings
	output = regexp.MustCompile(`-g[a-f0-9]{7,}`).ReplaceAllString(output, "-g<COMMIT>")

	return output
}

// RunGoldenTest runs a golden test with the binary version from environment
func RunGoldenTest(t *testing.T, test GoldenTest) {
	// Determine binary version from environment
	version := helpers.GetBinaryVersionFromEnv()

	// Adjust golden file path based on version if needed
	goldenFile := test.GoldenFile
	if version == helpers.BinaryV2 && os.Getenv("TREB_TEST_V2_GOLDEN") == "true" {
		// Use v2-specific golden files if they exist
		v2Golden := strings.Replace(goldenFile, ".golden", ".v2.golden", 1)
		if _, err := os.Stat(filepath.Join("testdata/golden", v2Golden)); err == nil {
			goldenFile = v2Golden
		}
	}

	helpers.IsolatedTestWithVersion(t, test.Name, version, func(t *testing.T, ctx *helpers.TrebContext) {
		// Run setup if provided
		if test.Setup != nil {
			test.Setup(t)
		}

		// Run the command
		output, err := ctx.Treb(test.Args...)

		// Check error expectation
		if test.ExpectErr {
			assert.Error(t, err, "Expected command to fail")
		} else {
			assert.NoError(t, err, "Command failed: %v\nOutput: %s", err, output)
		}

		// Normalize output if normalizer provided
		if test.Normalizer != nil {
			output = test.Normalizer(output)
		} else {
			output = normalizeOutput(output)
		}

		// Compare with golden file
		compareGolden(t, output, GoldenConfig{
			Path:        goldenFile,
			Normalizers: getDefaultNormalizers(),
		})
	})
}

// normalizeOutput applies standard normalizations
func normalizeOutput(output string) string {
	// Remove trailing whitespace from each line
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// getDefaultNormalizers returns the default set of normalizers
func getDefaultNormalizers() []Normalizer {
	return []Normalizer{
		ColorNormalizer{},
		TimestampNormalizer{},
		VersionNormalizer{},
		TargetedGitCommitNormalizer{},
		TargetedHashNormalizer{},
		// AddressNormalizer{}, // We don't normalize addresses as they should be deterministic
		// PathNormalizer{},    // Often we want to see actual paths
		// BlockNumberNormalizer{},
		// GasNormalizer{},
	}
}

// compareGolden compares output with golden file
func compareGolden(t *testing.T, output string, config GoldenConfig) {
	t.Helper()

	// Apply normalizers
	normalized := output
	for _, n := range config.Normalizers {
		normalized = n.Normalize(normalized)
	}

	// Ensure consistent line endings
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	
	// Always ensure a trailing newline for consistency
	if normalized != "" {
		normalized = normalized + "\n"
	}

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

// TrebGolden runs a treb command and compares output with golden file
func TrebGolden(t *testing.T, ctx *helpers.TrebContext, goldenPath string, args ...string) {
	t.Helper()

	output, err := ctx.Treb(args...)
	if err != nil {
		t.Fatalf("Command failed unexpectedly: %v\nArgs: %v\nOutput:\n%s", err, args, output)
	}

	compareGolden(t, output, GoldenConfig{
		Path:        goldenPath,
		Normalizers: getDefaultNormalizers(),
	})
}

// TrebGoldenWithError runs a command expecting an error and compares output
func TrebGoldenWithError(t *testing.T, ctx *helpers.TrebContext, goldenPath string, args ...string) {
	t.Helper()

	output, err := ctx.Treb(args...)
	require.Error(t, err)

	compareGolden(t, output, GoldenConfig{
		Path:        goldenPath,
		Normalizers: getDefaultNormalizers(),
	})
}


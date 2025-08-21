package helpers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// BinaryVersion represents which treb binary to use
type BinaryVersion string

const (
	BinaryV1 BinaryVersion = "v1"
	BinaryV2 BinaryVersion = "v2"
)

// TrebContext holds configuration for running treb commands in tests
type TrebContext struct {
	t             *testing.T
	Network       string
	Namespace     string
	BinaryVersion BinaryVersion
	workDir       string // Working directory for parallel tests
}

// NewTrebContext creates a new TrebContext with default settings
func NewTrebContext(t *testing.T, version BinaryVersion) *TrebContext {
	tc := &TrebContext{
		t:             t,
		Network:       "anvil-31337",
		Namespace:     "default",
		BinaryVersion: version,
	}

	return tc
}

func (tc *TrebContext) GetBinaryPath() string {
	switch tc.BinaryVersion {
	case BinaryV1:
		return filepath.Join(bin, "treb")
	case BinaryV2:
		return filepath.Join(bin, "treb-v2")
	default:
		panic(fmt.Errorf("Unexpected BineryVersion: %v", tc.BinaryVersion))
	}
}

// WithNetwork sets the network for this context
func (tc *TrebContext) WithNetwork(network string) *TrebContext {
	// Create a new context to avoid modifying the original
	newCtx := *tc
	newCtx.Network = network
	return &newCtx
}

// WithNamespace sets the namespace for this context
func (tc *TrebContext) WithNamespace(namespace string) *TrebContext {
	// Create a new context to avoid modifying the original
	newCtx := *tc
	newCtx.Namespace = namespace
	return &newCtx
}

// Treb runs a treb command with the context settings automatically applied
func (tc *TrebContext) Treb(args ...string) (string, error) {
	tc.t.Helper()

	// Build the full command with context flags
	allArgs := []string{"--non-interactive"}

	// Only add deployment context flags for commands that support them,
	// and only if not already explicitly provided in args
	if len(args) > 0 {
		cmd := args[0]

		// Determine if user already passed network/namespace
		hasNetwork := false
		hasNamespace := false
		for i := 0; i < len(args); i++ {
			if args[i] == "--network" && i+1 < len(args) {
				hasNetwork = true
			}
			if args[i] == "--namespace" && i+1 < len(args) {
				hasNamespace = true
			}
		}

		// Add network flag for commands that support it
		if tc.Network != "" && supportsNetworkFlag(cmd) && !hasNetwork {
			allArgs = append(allArgs, "--network", tc.Network)
		}

		// Add namespace flag for commands that support it
		if tc.Namespace != "" && supportsNamespaceFlag(cmd) && !hasNamespace {
			allArgs = append(allArgs, "--namespace", tc.Namespace)
		}
	}

	// Add the command arguments
	allArgs = append(allArgs, args...)

	// Check if debug mode is enabled
	debugMode := os.Getenv("TREB_TEST_DEBUG") != ""

	if debugMode {
		tc.t.Logf("=== TREB COMMAND DEBUG (%s) ===", tc.BinaryVersion)
		tc.t.Logf("Binary: %s", tc.GetBinaryPath())
		tc.t.Logf("Command: %s %s", tc.GetBinaryPath(), strings.Join(allArgs, " "))
		cwd, _ := os.Getwd()
		tc.t.Logf("Current Working Dir: %s", cwd)
		if tc.workDir != "" {
			tc.t.Logf("Test WorkDir: %s", tc.workDir)
			tc.t.Logf("Command Dir: %s", tc.workDir)
		} else {
			tc.t.Logf("Command Dir: %s", GetFixtureDir())
		}
		// Check if config file exists
		configPath := ".treb/config.local.json"
		if tc.workDir != "" {
			configPath = tc.workDir + "/" + configPath
		} else {
			configPath = GetFixtureDir() + "/" + configPath
		}
		if _, err := os.Stat(configPath); err == nil {
			tc.t.Logf("Config file exists at: %s", configPath)
		} else {
			tc.t.Logf("Config file NOT found at: %s (error: %v)", configPath, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, tc.GetBinaryPath(), allArgs...)
	// Use work directory if set (for parallel tests), otherwise use fixture dir
	if tc.workDir != "" {
		cmd.Dir = tc.workDir
	} else {
		cmd.Dir = GetFixtureDir()
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if debugMode {
		tc.t.Logf("Exit Code: %v", err)
		tc.t.Logf("=== STDOUT ===")
		tc.t.Logf("%s", stdout.String())
		tc.t.Logf("=== STDERR ===")
		tc.t.Logf("%s", stderr.String())
		tc.t.Logf("=== END DEBUG ===")
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after 30 seconds")
	}

	return output, err
}

// GetVersion returns the binary version being used
func (tc *TrebContext) GetVersion() BinaryVersion {
	return tc.BinaryVersion
}

// SetVersion sets the binary version and updates the binary path
func (tc *TrebContext) SetVersion(version BinaryVersion) {
	tc.BinaryVersion = version
}

// GetWorkDir returns the working directory for the test
func (tc *TrebContext) GetWorkDir() string {
	if tc.workDir != "" {
		return tc.workDir
	}
	return GetFixtureDir()
}

// supportsNetworkFlag returns true if the command supports --network flag
func supportsNetworkFlag(command string) bool {
	networkCommands := []string{"run", "show", "orchestrate", "prune"}
	return slices.Contains(networkCommands, command)
}

// supportsNamespaceFlag returns true if the command supports --namespace flag
func supportsNamespaceFlag(command string) bool {
	namespaceCommands := []string{"run", "show", "verify", "list", "tag"}
	return slices.Contains(namespaceCommands, command)
}

// Helper to determine which binary version to use based on environment
func GetBinaryVersionFromEnv() BinaryVersion {
	version := os.Getenv("TREB_TEST_BINARY")
	switch version {
	case "v2":
		return BinaryV2
	case "v1", "":
		return BinaryV1
	default:
		panic(fmt.Sprintf("Invalid TREB_TEST_BINARY value: %s", version))
	}
}

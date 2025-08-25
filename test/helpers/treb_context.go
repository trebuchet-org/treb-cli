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

// TrebContext holds configuration for running treb commands in tests
type TrebContext struct {
	t         *testing.T
	Network   string
	Namespace string
	workDir   string // Working directory for parallel tests
}

// NewTrebContext creates a new TrebContext with default settings
func NewTrebContext(t *testing.T) *TrebContext {
	tc := &TrebContext{
		t:         t,
		Network:   "anvil-31337",
		Namespace: "default",
	}

	return tc
}

func (tc *TrebContext) GetBinaryPath() string {
	return filepath.Join(bin, "treb")
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
	debugMode := IsDebugEnabled()

	if debugMode {
		tc.t.Logf("=== TREB COMMAND DEBUG ===")
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

	// Pass environment variables
	cmd.Env = os.Environ()
	if debugMode {
		// Pass TREB_TEST_DEBUG to the treb command if debug is enabled
		cmd.Env = append(cmd.Env, "TREB_TEST_DEBUG=1")
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

// Debug logs a message only if debug mode is enabled
func (tc *TrebContext) Debug(format string, args ...interface{}) {
	if IsDebugEnabled() {
		tc.t.Logf("[DEBUG] "+format, args...)
	}
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

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TrebContext holds configuration for running treb commands in tests
type TrebContext struct {
	t         *testing.T
	network   string
	namespace string
}

// NewTrebContext creates a new TrebContext with default settings
func NewTrebContext(t *testing.T) *TrebContext {
	return &TrebContext{
		t:         t,
        network:   "anvil0",
		namespace: "default",
	}
}

// WithNetwork sets the network for this context
func (tc *TrebContext) WithNetwork(network string) *TrebContext {
	// Create a new context to avoid modifying the original
	return &TrebContext{
		t:         tc.t,
		network:   network,
		namespace: tc.namespace,
	}
}

// WithNamespace sets the namespace for this context
func (tc *TrebContext) WithNamespace(namespace string) *TrebContext {
	// Create a new context to avoid modifying the original
	return &TrebContext{
		t:         tc.t,
		network:   tc.network,
		namespace: namespace,
	}
}

// treb runs a treb command with the context settings automatically applied
func (tc *TrebContext) treb(args ...string) (string, error) {
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
        if tc.network != "" && tc.supportsNetworkFlag(cmd) && !hasNetwork {
            allArgs = append(allArgs, "--network", tc.network)
        }

        // Add namespace flag for commands that support it
        if tc.namespace != "" && tc.supportsNamespaceFlag(cmd) && !hasNamespace {
            allArgs = append(allArgs, "--namespace", tc.namespace)
        }
    }

	// Add the command arguments
	allArgs = append(allArgs, args...)

	// Check if debug mode is enabled
	debugMode := os.Getenv("TREB_TEST_DEBUG") != ""

	if debugMode {
		tc.t.Logf("=== TREB COMMAND DEBUG ===")
		tc.t.Logf("Command: %s %s", trebBin, strings.Join(allArgs, " "))
		tc.t.Logf("Directory: %s", fixtureDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, trebBin, allArgs...)
	cmd.Dir = fixtureDir

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

// supportsNetworkFlag returns true if the command supports --network flag
func (tc *TrebContext) supportsNetworkFlag(command string) bool {
	networkCommands := []string{"run", "show", "orchestrate", "prune"}
	for _, cmd := range networkCommands {
		if command == cmd {
			return true
		}
	}
	return false
}

// supportsNamespaceFlag returns true if the command supports --namespace flag
func (tc *TrebContext) supportsNamespaceFlag(command string) bool {
	namespaceCommands := []string{"run", "show", "verify", "list", "tag", "prune"}
	for _, cmd := range namespaceCommands {
		if command == cmd {
			return true
		}
	}
	return false
}

// Helper to run treb commands with timeout (automatically adds --non-interactive)
// Deprecated: Use TrebContext.treb() instead
func runTreb(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Automatically add --non-interactive flag
	allArgs := append([]string{"--non-interactive"}, args...)

	// Check if debug mode is enabled
	debugMode := os.Getenv("TREB_TEST_DEBUG") != ""

	if debugMode {
		t.Logf("=== TREB COMMAND DEBUG (runTreb) ===")
		t.Logf("Command: %s %s", trebBin, strings.Join(allArgs, " "))
		t.Logf("Directory: %s", fixtureDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, trebBin, allArgs...)
	cmd.Dir = fixtureDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if debugMode {
		t.Logf("Exit Code: %v", err)
		t.Logf("=== STDOUT ===")
		t.Logf("%s", stdout.String())
		t.Logf("=== STDERR ===")
		t.Logf("%s", stderr.String())
		t.Logf("=== END DEBUG ===")
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after 30 seconds")
	}

	return output, err
}

// Helper functions
func cleanupGeneratedFiles(t *testing.T) {
	t.Helper()

	scriptDir := filepath.Join(fixtureDir, "script", "deploy")
	entries, _ := os.ReadDir(scriptDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "Deploy") && strings.HasSuffix(entry.Name(), ".s.sol") {
			os.Remove(filepath.Join(scriptDir, entry.Name()))
		}
	}

	// Clean .treb directory (v2 registry structure)
	trebDir := filepath.Join(fixtureDir, ".treb")
	os.RemoveAll(trebDir)

	// Clean forge cache to ensure fresh deployments
	cacheDir := filepath.Join(fixtureDir, "cache")
	os.RemoveAll(cacheDir)

	// Clean forge out directory
	outDir := filepath.Join(fixtureDir, "out")
	os.RemoveAll(outDir)
}

// runTrebDebug runs treb command and prints output on failure for debugging
func runTrebDebug(t *testing.T, args ...string) (string, error) {
	t.Helper()

	output, err := runTreb(t, args...)
	if err != nil {
		t.Logf("Command failed: treb %s", strings.Join(args, " "))
		t.Logf("Error: %v", err)
		t.Logf("Output:\n%s", output)
	}
	return output, err
}

// runScript executes a script with treb run command using environment variables
func runScript(t *testing.T, scriptPath string, envVars ...string) (string, error) {
	t.Helper()

    args := []string{"run", scriptPath, "--network", "anvil0"}

	// Add environment variables
	for _, envVar := range envVars {
		args = append(args, "--env", envVar)
	}

	return runTreb(t, args...)
}

// runScriptDebug executes a script with debug output on failure
func runScriptDebug(t *testing.T, scriptPath string, envVars ...string) (string, error) {
	t.Helper()

	output, err := runScript(t, scriptPath, envVars...)
	if err != nil {
		t.Logf("Script failed: %s", scriptPath)
		t.Logf("Env vars: %v", envVars)
		t.Logf("Error: %v", err)
		t.Logf("Output:\n%s", output)
	}
	return output, err
}

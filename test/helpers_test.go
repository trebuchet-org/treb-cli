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
	sender    string
}

// NewTrebContext creates a new TrebContext with default settings
func NewTrebContext(t *testing.T) *TrebContext {
	return &TrebContext{
		t:         t,
		network:   "anvil",
		namespace: "default",
		sender:    "anvil",
	}
}

// WithNetwork sets the network for this context
func (tc *TrebContext) WithNetwork(network string) *TrebContext {
	tc.network = network
	return tc
}

// WithNamespace sets the namespace for this context
func (tc *TrebContext) WithNamespace(namespace string) *TrebContext {
	tc.namespace = namespace
	return tc
}

// WithSender sets the sender for this context
func (tc *TrebContext) WithSender(sender string) *TrebContext {
	tc.sender = sender
	return tc
}

// treb runs a treb command with the context settings automatically applied
func (tc *TrebContext) treb(args ...string) (string, error) {
	tc.t.Helper()

	// Build the full command with context flags
	allArgs := []string{"--non-interactive"}

	// Only add deployment context flags for commands that support them
	if len(args) > 0 && tc.supportsDeploymentFlags(args[0]) {
		// Add context flags if they're set
		if tc.network != "" {
			allArgs = append(allArgs, "--network", tc.network)
		}
		if tc.namespace != "" {
			allArgs = append(allArgs, "--namespace", tc.namespace)
		}
	}

	// Add the command arguments
	allArgs = append(allArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, trebBin, allArgs...)
	cmd.Dir = fixtureDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after 30 seconds")
	}

	return output, err
}

// supportsDeploymentFlags returns true if the command supports network/namespace/sender flags
func (tc *TrebContext) supportsDeploymentFlags(command string) bool {
	deploymentCommands := []string{"run", "show", "verify", "list"}
	for _, cmd := range deploymentCommands {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, trebBin, allArgs...)
	cmd.Dir = fixtureDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

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

	args := []string{"run", scriptPath, "--network", "anvil"}

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

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// cleanTestArtifacts removes all test-generated files
func cleanTestArtifacts(t *testing.T) {
	t.Helper()

	// Clean .treb directory
	trebDir := filepath.Join(fixtureDir, ".treb")
	if err := os.RemoveAll(trebDir); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove .treb directory: %v", err)
	}

	// Clean broadcast directory
	broadcastDir := filepath.Join(fixtureDir, "broadcast")
	if err := os.RemoveAll(broadcastDir); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove broadcast directory: %v", err)
	}

	// Clean generated scripts
	scriptDir := filepath.Join(fixtureDir, "script", "deploy")
	entries, _ := os.ReadDir(scriptDir)
	for _, entry := range entries {
		if entry.Name() != ".gitkeep" {
			os.Remove(filepath.Join(scriptDir, entry.Name()))
		}
	}

	// Clean forge cache
	cacheDir := filepath.Join(fixtureDir, "cache")
	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove cache directory: %v", err)
	}

	// Clean forge out directory
	outDir := filepath.Join(fixtureDir, "out")
	if err := os.RemoveAll(outDir); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove out directory: %v", err)
	}
}

// TestCleanup manages test isolation
type TestCleanup struct {
	t        *testing.T
	snapshot *AnvilSnapshot
}

// NewTestCleanup creates a new test cleanup handler
func NewTestCleanup(t *testing.T) *TestCleanup {
	t.Helper()
	
	// Create a snapshot at the beginning of the test
	snapshot, err := globalAnvilManager.Snapshot()
	if err != nil {
		t.Fatalf("Failed to create test snapshot: %v", err)
	}
	
	return &TestCleanup{
		t:        t,
		snapshot: snapshot,
	}
}

// Cleanup reverts the snapshot and cleans artifacts
func (tc *TestCleanup) Cleanup() {
	tc.t.Helper()
	
	// Revert to the test snapshot
	if err := globalAnvilManager.Revert(tc.snapshot); err != nil {
		tc.t.Logf("Warning: failed to revert snapshot: %v", err)
	}
	
	// Clean test artifacts
	cleanTestArtifacts(tc.t)
}

// IsolatedTest runs a test with full isolation
func IsolatedTest(t *testing.T, name string, fn func(t *testing.T, ctx *TrebContext)) {
	t.Run(name, func(t *testing.T) {
		// Clean artifacts first to ensure clean state
		cleanTestArtifacts(t)
		
		// Create cleanup handler with snapshot
		cleanup := NewTestCleanup(t)
		defer cleanup.Cleanup()
		
		// Create test context and run test
		ctx := NewTrebContext(t)
		fn(t, ctx)
	})
}

// runCommand is a helper to run shell commands
func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}


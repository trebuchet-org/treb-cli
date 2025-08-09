package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SnapshotManager handles EVM snapshot/revert for test isolation
type SnapshotManager struct {
	t              *testing.T
	baseSnapshotID string
}

// baseSnapshot is the initial clean state snapshot ID (always "0x0")
const baseSnapshot = "0x0"

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(t *testing.T) *SnapshotManager {
	t.Helper()
	
	sm := &SnapshotManager{
		t:              t,
		baseSnapshotID: baseSnapshot,
	}
	return sm
}

// Revert rolls back to the base snapshot to ensure clean state
func (sm *SnapshotManager) Revert() {
	sm.t.Helper()
	
	// Always revert to the base snapshot for deterministic state
	output, err := runCommand("cast", "rpc", "evm_revert", sm.baseSnapshotID, "--rpc-url", "http://localhost:8545")
	if err != nil {
		sm.t.Fatalf("Failed to revert to base snapshot: %v\nOutput: %s", err, output)
	}
	
	sm.t.Logf("Reverted to base snapshot: %s", sm.baseSnapshotID)
}

// TestCleanup ensures clean state for each test
type TestCleanup struct {
	t        *testing.T
	snapshot *SnapshotManager
	cleanups []func()
}

// NewTestCleanup creates a new test cleanup manager
func NewTestCleanup(t *testing.T) *TestCleanup {
	t.Helper()
	
	return &TestCleanup{
		t:        t,
		snapshot: NewSnapshotManager(t),
		cleanups: []func(){},
	}
}

// AddCleanup registers a cleanup function to run at test end
func (tc *TestCleanup) AddCleanup(fn func()) {
	tc.cleanups = append(tc.cleanups, fn)
}

// Cleanup runs all cleanup functions
func (tc *TestCleanup) Cleanup() {
	tc.t.Helper()
	
	// Run cleanup functions in reverse order
	for i := len(tc.cleanups) - 1; i >= 0; i-- {
		tc.cleanups[i]()
	}
	
	// Note: Revert and artifact cleanup happen at the start of the next test
}

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

// IsolatedTest runs a test with full isolation
func IsolatedTest(t *testing.T, name string, fn func(t *testing.T, ctx *TrebContext)) {
	t.Run(name, func(t *testing.T) {
		cleanup := NewTestCleanup(t)
		defer cleanup.Cleanup()
		
		// Revert to clean state first
		cleanup.snapshot.Revert()
		
		// Then clean artifacts
		cleanTestArtifacts(t)
		
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
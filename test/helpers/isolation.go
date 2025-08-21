package helpers

import (
	"os"
	"path/filepath"
	"testing"
)

// cleanTestArtifacts removes all test-generated files
func cleanTestArtifacts(t *testing.T) {
	t.Helper()

	if ShouldSkipCleanup() {
		t.Logf("🔍 Skipping cleanup, test artifacts preserved at: %s", GetFixtureDir())
		return
	}

	fixtureDir := GetFixtureDir()

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
func NewTestCleanup(t *testing.T, manager *AnvilManager) *TestCleanup {
	t.Helper()

	// Create a snapshot at the beginning of the test
	snapshot, err := manager.Snapshot()
	if err != nil {
		t.Fatalf("Failed to create test snapshot: %v", err)
	}

	return &TestCleanup{
		t:        t,
		snapshot: snapshot,
	}
}

// Cleanup reverts the snapshot and cleans artifacts
func (tc *TestCleanup) Cleanup(manager *AnvilManager) {
	tc.t.Helper()

	// Revert to the test snapshot
	if err := manager.Revert(tc.snapshot); err != nil {
		tc.t.Logf("Warning: failed to revert snapshot: %v", err)
	}

	// Clean test artifacts
	cleanTestArtifacts(tc.t)
}

// IsolatedTest runs a test with full isolation
func IsolatedTest(t *testing.T, name string, fn func(t *testing.T, ctx *TrebContext)) {
	t.Run(name, func(t *testing.T) {
		// Always use pool-based isolation for consistency
		pool := GetGlobalPool()
		if pool == nil {
			t.Fatal("Test pool not initialized")
		}
		testCtx := pool.Acquire(t)

		// Only release if not skipping cleanup
		if !ShouldSkipCleanup() {
			defer pool.Release(testCtx)
		} else {
			defer func() {
				t.Logf("🔍 Test context not released due to skip cleanup flag: %s", testCtx.WorkDir)
			}()
		}

		// Determine binary version from environment or default
		version := GetBinaryVersionFromEnv()
		testCtx.TrebContext.SetVersion(version)

		// Run the test
		fn(t, testCtx.TrebContext)
	})
}

// IsolatedTestWithVersion runs a test with a specific binary version
func IsolatedTestWithVersion(t *testing.T, name string, version BinaryVersion, fn func(t *testing.T, ctx *TrebContext)) {
	t.Run(name, func(t *testing.T) {
		// Always use pool-based isolation for consistency
		pool := GetGlobalPool()
		if pool == nil {
			t.Fatal("Test pool not initialized")
		}
		testCtx := pool.Acquire(t)

		// Only release if not skipping cleanup
		if !ShouldSkipCleanup() {
			defer func() {
				if err := pool.Release(testCtx); err != nil {
					t.Fatal(err)
				}
			}()
		} else {
			defer func() {
				t.Logf("🔍 Test context not released due to skip cleanup flag: %s", testCtx.WorkDir)
			}()
		}

		// Set the specific version
		testCtx.TrebContext.SetVersion(version)

		// Run the test
		fn(t, testCtx.TrebContext)
	})
}

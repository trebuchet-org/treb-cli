package submodule

import (
	"os"
	"path/filepath"
	"testing"
	
	"github.com/trebuchet-org/treb-cli/cli/pkg/version"
)

func TestTrebSolManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "treb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	manager := NewTrebSolManager(tempDir)

	t.Run("IsTrebSolInstalled_NotInstalled", func(t *testing.T) {
		if manager.IsTrebSolInstalled() {
			t.Error("Expected IsTrebSolInstalled to return false for empty directory")
		}
	})

	t.Run("IsTrebSolInstalled_MockInstalled", func(t *testing.T) {
		// Create mock treb-sol directory with .git
		trebSolPath := filepath.Join(tempDir, "lib", "treb-sol")
		if err := os.MkdirAll(trebSolPath, 0755); err != nil {
			t.Fatalf("Failed to create treb-sol dir: %v", err)
		}
		
		// Create .git file (submodule indicator)
		gitPath := filepath.Join(trebSolPath, ".git")
		if err := os.WriteFile(gitPath, []byte("gitdir: ../../.git/modules/lib/treb-sol"), 0644); err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		if !manager.IsTrebSolInstalled() {
			t.Error("Expected IsTrebSolInstalled to return true for valid submodule")
		}
	})

	t.Run("GetExpectedCommit", func(t *testing.T) {
		// Test that GetExpectedCommit returns the version.TrebSolCommit value
		expected := manager.GetExpectedCommit()
		if expected != version.TrebSolCommit {
			t.Errorf("Expected GetExpectedCommit to return %s, got %s", version.TrebSolCommit, expected)
		}
	})

	t.Run("NeedsUpdate_UnknownExpected", func(t *testing.T) {
		// Save original value
		original := version.TrebSolCommit
		version.TrebSolCommit = "unknown"
		defer func() {
			version.TrebSolCommit = original
		}()

		needsUpdate, _, _, err := manager.NeedsUpdate()
		if err == nil {
			// If there's no error (likely no git repo), needsUpdate should be false
			if needsUpdate {
				t.Error("Expected NeedsUpdate to return false when expected commit is 'unknown'")
			}
		}
		// If there's an error (expected since we don't have a real git repo), that's fine
	})
}
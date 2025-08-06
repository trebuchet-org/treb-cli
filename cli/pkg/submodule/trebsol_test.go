package submodule

import (
	"os"
	"path/filepath"
	"testing"
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

}
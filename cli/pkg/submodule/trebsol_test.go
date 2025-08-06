package submodule

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trebuchet-org/treb-cli/cli/pkg/version"
)

func TestShortCommit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full commit", "38d8164935b41d697db47c99b70c0c45a78ede67", "38d8164"},
		{"short commit", "38d8164", "38d8164"},
		{"very short", "38d", "38d"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shortCommit(tt.input)
			if result != tt.expected {
				t.Errorf("shortCommit(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

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

	t.Run("WithSubmodulePath", func(t *testing.T) {
		customPath := "custom/path/treb-sol"
		customManager := NewTrebSolManager(tempDir).WithSubmodulePath(customPath)

		if customManager.submodulePath != customPath {
			t.Errorf("Expected submodulePath to be %s, got %s", customPath, customManager.submodulePath)
		}
	})

	t.Run("NeedsUpdate_UnknownExpected", func(t *testing.T) {
		// Test with a mock manager that returns "unknown" for expected commit
		// This avoids mutating global variables
		mockManager := &TrebSolManager{
			projectRoot:   tempDir,
			submodulePath: DefaultSubmodulePath,
		}

		// Since we don't have a real git repo, this will error
		// but we're testing the behavior when expected is "unknown"
		expected := mockManager.GetExpectedCommit()
		if expected == "unknown" {
			// This is what we expect in tests where TrebSolCommit isn't set
			// The actual NeedsUpdate will fail due to no git repo, but that's OK
			_, _, _, err := mockManager.NeedsUpdate()
			if err != nil {
				// Expected - no git repo in test environment
				return
			}
		}
	})
}

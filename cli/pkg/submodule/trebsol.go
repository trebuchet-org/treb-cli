package submodule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/version"
)


// TrebSolManager manages the treb-sol submodule version checking and updating
type TrebSolManager struct {
	projectRoot string
}

// NewTrebSolManager creates a new TrebSolManager
func NewTrebSolManager(projectRoot string) *TrebSolManager {
	return &TrebSolManager{
		projectRoot: projectRoot,
	}
}

// GetCurrentCommit returns the current commit hash of the treb-sol submodule
func (m *TrebSolManager) GetCurrentCommit() (string, error) {
	cmd := exec.Command("git", "submodule", "status", "lib/treb-sol")
	cmd.Dir = m.projectRoot
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get submodule status: %w", err)
	}
	
	// Parse output: " 38d8164935b41d697db47c99b70c0c45a78ede67 lib/treb-sol (remotes/origin/ignore-collisions)"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "lib/treb-sol") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Remove leading + or - if present
				commit := strings.TrimPrefix(parts[0], "+")
				commit = strings.TrimPrefix(commit, "-")
				return commit, nil
			}
		}
	}
	
	return "", fmt.Errorf("could not find lib/treb-sol in submodule status")
}

// GetExpectedCommit returns the expected treb-sol commit hash from build time
func (m *TrebSolManager) GetExpectedCommit() string {
	return version.TrebSolCommit
}

// CheckIfCommitExists checks if a commit exists in the local repository
func (m *TrebSolManager) CheckIfCommitExists(commit string) (bool, error) {
	cmd := exec.Command("git", "-C", "lib/treb-sol", "cat-file", "-e", commit)
	cmd.Dir = m.projectRoot
	
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means the object doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if commit exists: %w", err)
	}
	
	return true, nil
}

// CheckoutCommit checks out a specific commit in the treb-sol submodule
func (m *TrebSolManager) CheckoutCommit(commit string) error {
	// First fetch to ensure we have the commit
	fetchCmd := exec.Command("git", "-C", "lib/treb-sol", "fetch", "origin")
	fetchCmd.Dir = m.projectRoot
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	
	// Then checkout the specific commit
	checkoutCmd := exec.Command("git", "-C", "lib/treb-sol", "checkout", commit)
	checkoutCmd.Dir = m.projectRoot
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
	}
	
	return nil
}

// NeedsUpdate checks if the current commit matches the expected commit
func (m *TrebSolManager) NeedsUpdate() (bool, string, string, error) {
	current, err := m.GetCurrentCommit()
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get current commit: %w", err)
	}
	
	expected := m.GetExpectedCommit()
	
	// If expected is "unknown", skip the check
	if expected == "unknown" {
		return false, current, expected, nil
	}
	
	return current != expected, current, expected, nil
}

// Update updates the treb-sol submodule to the expected commit
func (m *TrebSolManager) Update(commit string) error {
	return m.CheckoutCommit(commit)
}

// CheckAndUpdate checks if the current commit matches the expected one and updates if needed
func (m *TrebSolManager) CheckAndUpdate(silent bool) error {
	needsUpdate, current, expected, err := m.NeedsUpdate()
	if err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "Warning: Could not check treb-sol version: %v\n", err)
		}
		return nil // Don't fail the command
	}
	
	if !needsUpdate {
		return nil
	}
	
	// Check if the expected commit exists locally
	exists, err := m.CheckIfCommitExists(expected)
	if err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "Warning: Could not check if commit exists: %v\n", err)
		}
		return nil
	}
	
	if exists {
		// Commit exists locally but is different - just print a warning
		if !silent {
			fmt.Fprintf(os.Stderr, "Warning: treb-sol is at commit %s but treb expects %s\n", current[:7], expected[:7])
			fmt.Fprintf(os.Stderr, "Continuing with current version...\n")
		}
		return nil
	}
	
	// Commit doesn't exist locally - need to update
	if !silent {
		fmt.Printf("Updating treb-sol from %s to %s...\n", current[:7], expected[:7])
	}
	
	if err := m.Update(expected); err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update treb-sol to expected version: %v\n", err)
			fmt.Fprintf(os.Stderr, "You may need to manually update treb-sol\n")
		}
		return nil // Don't fail the command
	}
	
	if !silent {
		fmt.Println("Successfully updated treb-sol to expected version")
	}
	
	return nil
}

// IsTrebSolInstalled checks if treb-sol is properly installed as a submodule
func (m *TrebSolManager) IsTrebSolInstalled() bool {
	trebSolPath := filepath.Join(m.projectRoot, "lib", "treb-sol")
	info, err := os.Stat(trebSolPath)
	if err != nil {
		return false
	}
	
	// Check if it's a directory and has .git file/folder (submodule indicator)
	if !info.IsDir() {
		return false
	}
	
	gitPath := filepath.Join(trebSolPath, ".git")
	_, err = os.Stat(gitPath)
	return err == nil
}
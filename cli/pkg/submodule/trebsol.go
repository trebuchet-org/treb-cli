package submodule

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	trebSolRepo = "https://github.com/trebuchet-org/treb-sol"
	trebSolAPI  = "https://api.github.com/repos/trebuchet-org/treb-sol/commits/main"
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

// GetLatestCommit fetches the latest commit hash from the treb-sol repository
func (m *TrebSolManager) GetLatestCommit() (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// Create request
	req, err := http.NewRequest("GET", trebSolAPI, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "treb-cli")
	
	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest commit: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var result struct {
		SHA string `json:"sha"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	return result.SHA, nil
}

// IsUpdateAvailable checks if there's a newer version of treb-sol available
func (m *TrebSolManager) IsUpdateAvailable() (bool, string, string, error) {
	current, err := m.GetCurrentCommit()
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get current commit: %w", err)
	}
	
	latest, err := m.GetLatestCommit()
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get latest commit: %w", err)
	}
	
	return current != latest, current, latest, nil
}

// Update updates the treb-sol submodule to the latest version
func (m *TrebSolManager) Update() error {
	// First, fetch the latest changes
	cmd := exec.Command("git", "submodule", "update", "--init", "--remote", "lib/treb-sol")
	cmd.Dir = m.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update submodule: %w", err)
	}
	
	return nil
}

// CheckAndUpdate checks for updates and updates if available
func (m *TrebSolManager) CheckAndUpdate(silent bool) error {
	hasUpdate, current, latest, err := m.IsUpdateAvailable()
	if err != nil {
		// In case of network error or API issues, just print warning
		if !silent {
			fmt.Fprintf(os.Stderr, "Warning: Could not check for treb-sol updates: %v\n", err)
		}
		return nil // Don't fail the command
	}
	
	if !hasUpdate {
		return nil
	}
	
	if !silent {
		fmt.Printf("Updating treb-sol from %s to %s...\n", current[:7], latest[:7])
	}
	
	if err := m.Update(); err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update treb-sol to latest version: %v\n", err)
			fmt.Fprintf(os.Stderr, "You may need to manually update treb-sol by running: git submodule update --init --remote treb-sol\n")
		}
		return nil // Don't fail the command
	}
	
	if !silent {
		fmt.Println("Successfully updated treb-sol to latest version")
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
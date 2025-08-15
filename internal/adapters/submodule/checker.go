package submodule

import (
	"fmt"
	"os"
	"path/filepath"
)

// Checker checks for the presence of required submodules
type Checker struct {
	projectRoot string
}

// NewChecker creates a new submodule checker
func NewChecker(projectRoot string) *Checker {
	return &Checker{
		projectRoot: projectRoot,
	}
}

// CheckTrebSol checks if the treb-sol submodule is installed
func (c *Checker) CheckTrebSol() error {
	// Check if treb-sol directory exists in lib
	trebSolPath := filepath.Join(c.projectRoot, "lib", "treb-sol")
	if _, err := os.Stat(trebSolPath); os.IsNotExist(err) {
		return fmt.Errorf("treb-sol not found. Please run 'forge install trebuchet-org/treb-sol' to install it")
	}

	// Check if it's a valid directory
	info, err := os.Stat(trebSolPath)
	if err != nil {
		return fmt.Errorf("failed to check treb-sol: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("treb-sol exists but is not a directory")
	}

	// Check for key files to ensure it's properly installed
	requiredFiles := []string{
		"src/TrebuchetDeployment.sol",
		"src/interfaces/ITrebEvents.sol",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(trebSolPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("treb-sol is incomplete. Missing %s. Please reinstall with 'forge install trebuchet-org/treb-sol'", file)
		}
	}

	return nil
}
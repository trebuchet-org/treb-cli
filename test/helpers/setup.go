package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Setup() error {
	// First, ensure we build the binaries before tests
	fmt.Println("🔨 Building treb binaries...")
	if err := buildBinaries(); err != nil {
		return fmt.Errorf("failed to build binaries: %w", err)
	}
	// Change to fixture directory and build contracts
	if err := os.Chdir(fixtureDir); err != nil {
		return fmt.Errorf("failed to change to fixture directory: %w", err)
	}

	// Clean up previous test artifacts
	fmt.Println("🧹 Cleaning previous test artifacts...")
	os.RemoveAll(".treb")
	os.RemoveAll("broadcast")

	fmt.Println("🔨 Building contracts...")
	cmd := exec.Command("forge", "build")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contracts: %w", err)
	}

	fmt.Println("🔗 Starting anvil nodes with CreateX factory...")
	if err := anvilManager.StartAll(); err != nil {
		return fmt.Errorf("failed to start anvil nodes: %w", err)
	}
	fmt.Println("✅ Anvil nodes with CreateX ready")

	return nil
}

func buildBinaries() error {
	// Get project root (parent of test directory)
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(filepath.Dir(wd))

	// Build v1 binary
	cmd := exec.Command("make", "build")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build v1 binary: %w\nOutput: %s", err, output)
	}

	// Build v2 binary if it exists
	cmd = exec.Command("make", "build-v2")
	cmd.Dir = projectRoot
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build v2 binary: %w\nOutput: %s", err, output)
	}

	return nil
}

func Teardown() {
	fmt.Println("🧹 Cleaning up...")
	anvilManager.StopAll()
}

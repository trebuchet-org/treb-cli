package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/trebuchet-org/treb-cli/cli/pkg/dev"
)

// Global test variables
var (
	trebBin    string
	fixtureDir string
)

// TestMain handles setup/teardown for all tests
func TestMain(m *testing.M) {
	// Setup
	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	teardown()

	os.Exit(code)
}

func setup() error {
	// Build treb with dev tag for anvil management
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	trebBin = filepath.Join(projectRoot, "treb")
	fixtureDir = filepath.Join(wd, "fixture")

	fmt.Println("🔨 Building treb binary with dev tools...")
	cmd := exec.Command("go", "build", "-tags", "dev", "-o", "treb", "./cli")
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build treb: %w", err)
	}

	// Change to fixture directory and build contracts
	if err := os.Chdir(fixtureDir); err != nil {
		return fmt.Errorf("failed to change to fixture directory: %w", err)
	}

	fmt.Println("🔨 Building contracts...")
	cmd = exec.Command("forge", "build")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build contracts: %w", err)
	}

	// Start anvil with CreateX using our management tool
	fmt.Println("🔗 Starting anvil node with CreateX factory...")
	if err := dev.StartAnvil(); err != nil {
		return fmt.Errorf("failed to start anvil: %w", err)
	}

	fmt.Println("✅ Anvil node with CreateX ready")
	return nil
}

func teardown() {
	fmt.Println("🧹 Cleaning up...")
	// Stop anvil using our management tool
	if err := dev.StopAnvil(); err != nil {
		fmt.Printf("Warning: failed to stop anvil: %v\n", err)
	}
}
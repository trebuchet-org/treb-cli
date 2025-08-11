package integration_test

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

)

// Global test variables
var (
	trebBin            string
	fixtureDir         string
	globalAnvilManager *AnvilManager
)

// TestMain handles setup/teardown for all tests
func TestMain(m *testing.M) {
	// Force sequential test execution by setting parallel to 1
	// This prevents nonce issues when multiple tests try to deploy simultaneously
	testing.Init()
	flag.Parse()
	if !flag.Parsed() {
		flag.Set("test.parallel", "1")
	}

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
	trebBin = filepath.Join(projectRoot, "bin", "treb")
	fixtureDir = filepath.Join(wd, "testdata/project")

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

	// Create anvil manager and start nodes
	t := &testing.T{} // Dummy testing.T for setup
	globalAnvilManager = NewAnvilManager(t)

	fmt.Println("🔗 Starting anvil nodes with CreateX factory...")
	if err := globalAnvilManager.StartAll(); err != nil {
		return fmt.Errorf("failed to start anvil nodes: %w", err)
	}
	fmt.Println("✅ Anvil nodes with CreateX ready")

	return nil
}

func teardown() {
	fmt.Println("🧹 Cleaning up...")
	// Stop all anvil nodes
	if globalAnvilManager != nil {
		globalAnvilManager.StopAll()
	}
}

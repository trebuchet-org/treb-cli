package integration_test

import (
	"flag"
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

	// Restart anvil with CreateX using our management tool (ensures clean state)
	fmt.Println("🔗 Restarting anvil node with CreateX factory...")
	if err := dev.RestartAnvil(); err != nil {
		return fmt.Errorf("failed to restart anvil: %w", err)
	}

	fmt.Println("✅ Anvil node with CreateX ready")
	
	// Create initial snapshot for deterministic test isolation
	fmt.Println("📸 Creating base snapshot...")
	output, err := exec.Command("cast", "rpc", "evm_snapshot", "--rpc-url", "http://localhost:8545").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create base snapshot: %w\nOutput: %s", err, output)
	}
	fmt.Println("✅ Base snapshot created")
	
	return nil
}

func teardown() {
	fmt.Println("🧹 Cleaning up...")
	// Stop anvil using our management tool
	if err := dev.StopAnvil(); err != nil {
		fmt.Printf("Warning: failed to stop anvil: %v\n", err)
	}
}

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

    // Restart two anvil nodes with CreateX for multichain tests
    fmt.Println("🔗 Restarting anvil nodes with CreateX factory...")
    if err := dev.RestartAnvilInstance("anvil0", "8545", "31337"); err != nil {
        return fmt.Errorf("failed to restart anvil0: %w", err)
    }
    if err := dev.RestartAnvilInstance("anvil1", "9545", "31338"); err != nil {
        return fmt.Errorf("failed to restart anvil1: %w", err)
    }

    fmt.Println("✅ Anvil nodes with CreateX ready")
    
    // Create initial snapshots for deterministic test isolation
    fmt.Println("📸 Creating base snapshots...")
    if output, err := exec.Command("cast", "rpc", "evm_snapshot", "--rpc-url", "http://localhost:8545").CombinedOutput(); err != nil {
        return fmt.Errorf("failed to create base snapshot on anvil0: %w\nOutput: %s", err, output)
    }
    if output, err := exec.Command("cast", "rpc", "evm_snapshot", "--rpc-url", "http://localhost:9545").CombinedOutput(); err != nil {
        return fmt.Errorf("failed to create base snapshot on anvil1: %w\nOutput: %s", err, output)
    }
    fmt.Println("✅ Base snapshots created")
	
	return nil
}

func teardown() {
	fmt.Println("🧹 Cleaning up...")
    // Stop anvils using our management tool
    if err := dev.StopAnvilInstance("anvil0", "8545"); err != nil {
        fmt.Printf("Warning: failed to stop anvil0: %v\n", err)
    }
    if err := dev.StopAnvilInstance("anvil1", "9545"); err != nil {
        fmt.Printf("Warning: failed to stop anvil1: %v\n", err)
    }
}

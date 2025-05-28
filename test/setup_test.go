package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Global test variables
var (
	trebBin    string
	fixtureDir string
	anvilCmd   *exec.Cmd
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
	// Build treb
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	trebBin = filepath.Join(projectRoot, "treb")
	fixtureDir = filepath.Join(wd, "fixture")

	fmt.Println("🔨 Building treb binary...")
	cmd := exec.Command("go", "build", "-o", "treb", "./cli")
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

	// Start anvil
	fmt.Println("🔗 Starting anvil node...")
	anvilCmd = exec.Command("anvil", "--host", "0.0.0.0", "--port", "8545")
	if err := anvilCmd.Start(); err != nil {
		return fmt.Errorf("failed to start anvil: %w", err)
	}

	// Wait for anvil
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("anvil did not start in time")
		default:
			if resp, err := http.Get("http://localhost:8545"); err == nil {
				resp.Body.Close()
				fmt.Println("✅ Anvil node ready")
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func teardown() {
	fmt.Println("🧹 Cleaning up...")
	if anvilCmd != nil && anvilCmd.Process != nil {
		anvilCmd.Process.Kill()
		anvilCmd.Wait()
	}
}
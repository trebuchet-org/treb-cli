package integration

import (
	"flag"
	"fmt"
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"testing"
)

// TestMain handles setup/teardown for all tests
func TestMain(m *testing.M) {
	// Initialize test flags (this will also parse flags)
	helpers.InitTestFlags()

	// Always use pool-based isolation - default to 1 context for sequential tests
	poolSize := 1
	
	// Check environment variable override
	if envSize := os.Getenv("TREB_TEST_POOL_SIZE"); envSize != "" {
		fmt.Sscanf(envSize, "%d", &poolSize)
	}

	fmt.Printf("🚀 Initializing test pool with %d contexts...\n", poolSize)
	
	if err := helpers.InitializeTestPool(poolSize); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize test pool: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	fmt.Println("🧹 Cleaning up test pool...")
	if pool := helpers.GetGlobalPool(); pool != nil {
		pool.Shutdown()
	}

	os.Exit(code)
}

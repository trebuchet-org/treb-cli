package compatibility

import (
	"flag"
	"fmt"
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"testing"
)

// parseParallelFlag extracts the value of -test.parallel flag
func parseParallelFlag() int {
	parallelFlag := 0
	flag.VisitAll(func(f *flag.Flag) {
		if f.Name == "test.parallel" {
			fmt.Sscanf(f.Value.String(), "%d", &parallelFlag)
		}
	})
	return parallelFlag
}

// TestMain handles setup/teardown for all tests
func TestMain(m *testing.M) {
	// Initialize test flags (this will also parse flags)
	helpers.InitTestFlags()

	// Get the value of -test.parallel flag
	parallelFlag := parseParallelFlag()
	
	var code int
	
	// Use parallel mode if Go's parallel flag is > 1 or TREB_TEST_PARALLEL env var is set
	useParallel := parallelFlag > 1 || os.Getenv("TREB_TEST_PARALLEL") == "true"
	
	if useParallel {
		// Parallel mode - use context pool
		// Use the parallel flag value as pool size, or default to 4
		poolSize := parallelFlag
		if poolSize <= 1 {
			poolSize = 4
			// Check environment variable as fallback
			if envSize := os.Getenv("TREB_TEST_POOL_SIZE"); envSize != "" {
				fmt.Sscanf(envSize, "%d", &poolSize)
			}
		}

		fmt.Printf("🚀 Initializing parallel test pool with %d contexts...\n", poolSize)
		
		if err := helpers.InitializeTestPool(poolSize); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize test pool: %v\n", err)
			os.Exit(1)
		}

		// Run tests
		code = m.Run()

		// Cleanup
		fmt.Println("🧹 Cleaning up test pool...")
		if pool := helpers.GetGlobalPool(); pool != nil {
			pool.Shutdown()
		}
	} else {
		// Sequential mode - use existing setup
		// Force sequential test execution by setting parallel to 1
		// This prevents nonce issues when multiple tests try to deploy simultaneously
		testing.Init()
		flag.Set("test.parallel", "1")

		// Setup
		if err := helpers.Setup(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to setup: %v\n", err)
			os.Exit(1)
		}

		// Run tests
		code = m.Run()
		defer func() {
			if r := recover(); r != nil {
				helpers.Teardown()
			}
		}()

		// Teardown
		helpers.Teardown()
	}

	os.Exit(code)
}

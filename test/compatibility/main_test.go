package compatibility

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
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

	// Always use pool-based isolation for consistency
	// Use the parallel flag value as pool size, or default based on parallel flag
	poolSize := parallelFlag
	if poolSize <= 0 {
		poolSize = 1 // Minimum pool size for sequential mode
	}

	runFlag := flag.Lookup("test.run")
	if runFlag != nil && strings.Contains(runFlag.Value.String(), "/") {
		poolSize = 2
	}

	// Check environment variable override
	if envSize := os.Getenv("TREB_TEST_POOL_SIZE"); envSize != "" {
		fmt.Sscanf(envSize, "%d", &poolSize)
	}

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

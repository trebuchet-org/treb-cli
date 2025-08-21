package compatibility

import (
	"testing"
	"time"
)

func TestParallelExecution(t *testing.T) {
	// Simple test to verify parallel execution works
	tests := []CompatibilityTest{
		{
			Name: "parallel_test_1",
			TestCmds: [][]string{
				{"version"},
			},
			ExpectDiff: true, // Different version output format
		},
		{
			Name: "parallel_test_2",
			TestCmds: [][]string{
				{"--help"},
			},
			ExpectDiff: true, // Different help text between versions
		},
		{
			Name: "parallel_test_3",
			TestCmds: [][]string{
				{"networks"},
			},
		},
		{
			Name: "parallel_test_4",
			TestCmds: [][]string{
				{"config", "show"},
			},
		},
		{
			Name: "parallel_test_5",
			TestCmds: [][]string{
				{"list"},
			},
		},
	}

	// Record start time
	start := time.Now()
	
	// Run tests (they should run in parallel if TREB_TEST_PARALLEL=true)
	RunCompatibilityTests(t, tests)
	
	// Log execution time
	t.Logf("Tests completed in %v", time.Since(start))
}
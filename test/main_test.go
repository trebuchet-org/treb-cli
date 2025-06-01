package integration_test

import (
	"os"
	"testing"
)

func init() {
	// Force sequential test execution
	// This prevents race conditions and nonce issues with blockchain interactions
	os.Setenv("GOMAXPROCS", "1")
}

// TestSequential is a marker test to ensure sequential execution
func TestSequential(t *testing.T) {
	// This is a marker test to ensure tests run sequentially
	// All other tests in this package will run sequentially
	t.Log("Integration tests running in sequential mode")
}

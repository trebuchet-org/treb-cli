package compatibility

import (
	"flag"
	"fmt"
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"testing"
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
	if err := helpers.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()
	defer func() {
		if r := recover(); r != nil {
			helpers.Teardown()
		}
	}()

	// Teardown
	helpers.Teardown()

	os.Exit(code)
}

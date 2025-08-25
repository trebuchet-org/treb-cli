package compatibility

import (
	"os"
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

// TestMain handles setup/teardown for all tests
func TestMain(m *testing.M) {
	helpers.Setup()
	defer helpers.Cleanup()
	// Run tests
	code := m.Run()
	os.Exit(code)
}

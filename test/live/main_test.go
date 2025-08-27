package live

import (
	"os"
	"runtime"
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

// TestMain handles setup/teardown for all tests
func TestMain(m *testing.M) {
	runtime.GOMAXPROCS(1)

	if err := helpers.Setup(); err != nil {
		panic(err)
	}
	// Run tests
	code := m.Run()
	helpers.Cleanup()
	os.Exit(code)
}

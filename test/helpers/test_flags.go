package helpers

import (
	"flag"
	"os"
	"testing"
)

// Test flags
var (
	debugFlag        = flag.Bool("treb.debug", false, "Enable debug output for treb tests (equivalent to TREB_TEST_DEBUG=1)")
	updateGoldenFlag = flag.Bool("treb.updategolden", false, "Update golden files (equivalent to UPDATE_GOLDEN=true)")
)

// InitTestFlags initializes test flags and syncs them with environment variables
func InitTestFlags() {
	// Parse flags if not already parsed
	if !flag.Parsed() {
		flag.Parse()
	}

	// Sync flags with environment variables
	// Flag takes precedence over env var
	if *debugFlag {
		os.Setenv("TREB_TEST_DEBUG", "1")
	} else if os.Getenv("TREB_TEST_DEBUG") != "" {
		*debugFlag = true
	}

	if *updateGoldenFlag {
		os.Setenv("UPDATE_GOLDEN", "true")
	} else if os.Getenv("UPDATE_GOLDEN") == "true" {
		*updateGoldenFlag = true
	}
}

// IsDebugEnabled returns true if debug mode is enabled
func IsDebugEnabled() bool {
	return *debugFlag || os.Getenv("TREB_TEST_DEBUG") != ""
}

// ShouldUpdateGolden returns true if golden files should be updated
func ShouldUpdateGolden() bool {
	return *updateGoldenFlag || os.Getenv("UPDATE_GOLDEN") == "true"
}

// Debugf logs a debug message if debug mode is enabled
func Debugf(t testing.TB, format string, args ...interface{}) {
	if IsDebugEnabled() {
		t.Logf("[DEBUG] "+format, args...)
	}
}
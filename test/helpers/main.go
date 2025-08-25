package helpers

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
)

// Test flags
var (
	debugFlag        = flag.Bool("treb.debug", false, "Enable debug output for treb tests (equivalent to TREB_TEST_DEBUG=1)")
	updateGoldenFlag = flag.Bool("treb.updategolden", false, "Update golden files (equivalent to UPDATE_GOLDEN=true)")
	skipCleanupFlag  = flag.Bool("treb.skipcleanup", false, "Skip cleanup of test artifacts and log work directories (equivalent to TREB_TEST_SKIP_CLEANUP=1)")
)
var parallel = 1
var setupSpinner *spinner.Spinner
var (
	bin             string
	fixtureDir      string
	testProjectRoot string
)

func Setup() error {
	parseFlags()

	setupSpinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	defer setupSpinner.Stop()
	setupSpinner.Start()

	setupSpinner.Suffix = "Setting up testing env"

	if err := buildBinaries(); err != nil {
		return err
	}
	if err := initializeTestPool(parallel); err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println("Caught signal:", sig)

		// Perform cleanup here
		Cleanup()

		// Exit immediately after cleanup
		os.Exit(1)
	}()

	return nil
}

func Cleanup() {
	globalPool.Shutdown()
}

func buildBinaries() error {
	setupSpinner.Suffix = "Building binaries..."
	// Get project root (parent of test directory)
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(filepath.Dir(wd))

	// Build v1 binary
	cmd := exec.Command("make", "build")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build binary: %w\nOutput: %s", err, output)
	}

	return nil
}

// InitTestFlags initializes test flags and syncs them with environment variables
func parseFlags() {
	// Parse flags if not already parsed
	if !flag.Parsed() {
		flag.Parse()
	}

	parallelF := flag.Lookup("test.parallel")
	fmt.Sscanf(parallelF.Value.String(), "%d", &parallel)
}

// IsDebugEnabled returns true if debug mode is enabled
func IsDebugEnabled() bool {
	return *debugFlag
}

// ShouldUpdateGolden returns true if golden files should be updated
func ShouldUpdateGolden() bool {
	return *updateGoldenFlag
}

// ShouldSkipCleanup returns true if test cleanup should be skipped
func ShouldSkipCleanup() bool {
	return *skipCleanupFlag
}

func Parallel() int {
	return parallel
}

// InitGlobals initializes global test variables
func init() {
	// Get working directory
	trebProjectRoot, _ := os.Getwd()
	for !isTrebRoot(trebProjectRoot) && filepath.Dir(trebProjectRoot) != trebProjectRoot {
		trebProjectRoot = filepath.Dir(trebProjectRoot)
	}

	testProjectRoot = filepath.Join(trebProjectRoot, "test")
	fixtureDir = filepath.Join(testProjectRoot, "testdata", "project")

	// Binary is at project root level
	bin = filepath.Join(trebProjectRoot, "bin")
}

func isTrebRoot(wd string) bool {
	goModFile := filepath.Join(wd, "go.mod")
	if _, err := os.Stat(goModFile); err != nil {
		return false
	}
	if output, err := os.ReadFile(goModFile); err == nil {
		return strings.Contains(string(output), "module github.com/trebuchet-org/treb-cli\n")
	} else {
		panic(err)
	}
}

// GetTrebBin returns the path to the treb binary
func GetTrebBin() string {
	return filepath.Join(bin, "treb")
}

// GetFixtureDir returns the path to the test fixture directory
func GetFixtureDir() string {
	return fixtureDir
}

func GoldenPath(test string) string {
	return filepath.Join(testProjectRoot, "testdata", "golden", test)
}

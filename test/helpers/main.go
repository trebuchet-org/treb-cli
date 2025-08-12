package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Global test variables
var (
	bin             string
	fixtureDir      string
	testProjectRoot string
	anvilManager    *AnvilManager
)

// InitGlobals initializes global test variables
func init() {
	// Get working directory
	trebProjectRoot, _ := os.Getwd()
	for !isTrebRoot(trebProjectRoot) && filepath.Dir(trebProjectRoot) != trebProjectRoot {
		trebProjectRoot = filepath.Dir(trebProjectRoot)
	}

	fmt.Println(trebProjectRoot)

	testProjectRoot = filepath.Join(trebProjectRoot, "test")
	fixtureDir = filepath.Join(testProjectRoot, "testdata", "project")

	// Binary is at project root level
	bin = filepath.Join(trebProjectRoot, "bin")

	var err error
	anvilManager, err = NewAnvilManager()
	if err != nil {
		panic(err)
	}

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

// GetV2Binary returns the path to the v2 binary
func GetV2Binary() string {
	return filepath.Join(bin, "treb-v2")
}

// V2BinaryExists checks if v2 binary exists
func V2BinaryExists() bool {
	_, err := os.Stat(GetV2Binary())
	return err == nil
}

// GetGlobalAnvilManager returns the global anvil manager
func GetAnvilManager() *AnvilManager {
	return anvilManager
}

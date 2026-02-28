package helpers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TrebContext holds configuration for running treb commands in tests
type TrebContext struct {
	t       *testing.T
	workDir string // Working directory for parallel tests
	Bin     string
}

// NewTrebContext creates a new TrebContext with default settings
func NewTrebContext(t *testing.T, tctx *TestContext) *TrebContext {
	tc := &TrebContext{
		t:       t,
		Bin:     filepath.Join(bin, "treb"),
		workDir: tctx.WorkDir,
	}

	return tc
}

// Treb runs a treb command with the context settings automatically applied
func (tc *TrebContext) Treb(args ...string) (string, error) {
	tc.t.Helper()

	// Build the full command with context flags
	additionalArgs := []string{"--non-interactive"}
	// Add the command arguments
	allArgs := append(slices.Clone(args), additionalArgs...)

	// Check if debug mode is enabled
	debugMode := IsDebugEnabled()

	if debugMode {
		tc.t.Logf("=== TREB COMMAND DEBUG ===")
		tc.t.Logf("Binary: %s", tc.Bin)
		tc.t.Logf("Command: %s %s", tc.Bin, strings.Join(allArgs, " "))
		cwd, _ := os.Getwd()
		tc.t.Logf("Current Working Dir: %s", cwd)
		if tc.workDir != "" {
			tc.t.Logf("Test WorkDir: %s", tc.workDir)
			tc.t.Logf("Command Dir: %s", tc.workDir)
		} else {
			tc.t.Logf("Command Dir: %s", GetFixtureDir())
		}
		// Check if config file exists
		configPath := ".treb/config.local.json"
		if tc.workDir != "" {
			configPath = tc.workDir + "/" + configPath
		} else {
			configPath = GetFixtureDir() + "/" + configPath
		}
		if _, err := os.Stat(configPath); err == nil {
			tc.t.Logf("Config file exists at: %s", configPath)
		} else {
			tc.t.Logf("Config file NOT found at: %s (error: %v)", configPath, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, tc.Bin, allArgs...)
	// Use work directory if set (for parallel tests), otherwise use fixture dir
	if tc.workDir != "" {
		cmd.Dir = tc.workDir
	} else {
		cmd.Dir = GetFixtureDir()
	}

	// Pass environment variables
	cmd.Env = os.Environ()
	if debugMode {
		// Pass TREB_TEST_DEBUG to the treb command if debug is enabled
		cmd.Env = append(cmd.Env, "TREB_LOG_LEVEL=debug")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if debugMode {
		tc.t.Logf("Exit Code: %v", err)
		tc.t.Logf("=== STDOUT ===")
		tc.t.Logf("%s", stdout.String())
		tc.t.Logf("=== STDERR ===")
		tc.t.Logf("%s", stderr.String())
		tc.t.Logf("=== END DEBUG ===")
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out")
	}

	return output, err
}

// Debug logs a message only if debug mode is enabled
func (tc *TrebContext) Debug(format string, args ...interface{}) {
	if IsDebugEnabled() {
		tc.t.Logf("[DEBUG] "+format, args...)
	}
}

// GetWorkDir returns the working directory for the test
func (tc *TrebContext) GetWorkDir() string {
	if tc.workDir != "" {
		return tc.workDir
	}
	return GetFixtureDir()
}

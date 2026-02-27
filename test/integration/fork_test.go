package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

// setupForkEnvVars converts hardcoded RPC URLs in foundry.toml to env var references
// so that fork enter can work (it requires env var references for RPC override).
func setupForkEnvVars(t *testing.T, ctx *helpers.TestContext) {
	t.Helper()
	workDir := ctx.TrebContext.GetWorkDir()
	foundryPath := filepath.Join(workDir, "foundry.toml")
	envPath := filepath.Join(workDir, ".env")

	// Read current foundry.toml
	data, err := os.ReadFile(foundryPath)
	require.NoError(t, err)
	content := string(data)

	// Find the anvil-31337 URL - the go-toml marshal may use different quoting
	// Look for the URL pattern after the key
	for _, network := range []string{"anvil-31337", "anvil-31338"} {
		node := ctx.AnvilNodes[network]
		if node == nil {
			continue
		}

		envVarName := strings.ToUpper(strings.NewReplacer("-", "_").Replace(network)) + "_RPC_URL"

		// Replace the URL with env var reference - handle both single and double quotes
		// The test framework rewrites foundry.toml with go-toml which may use single quotes
		oldPatterns := []string{
			fmt.Sprintf(`'%s'`, node.URL),
			fmt.Sprintf(`"%s"`, node.URL),
		}
		replacement := fmt.Sprintf(`"${%s}"`, envVarName)

		replaced := false
		for _, old := range oldPatterns {
			if strings.Contains(content, old) {
				content = strings.Replace(content, old, replacement, 1)
				replaced = true
				break
			}
		}
		if !replaced {
			t.Logf("Warning: could not find URL %s in foundry.toml for env var migration", node.URL)
			continue
		}

		// Append env var to .env
		envLine := fmt.Sprintf("%s=%s\n", envVarName, node.URL)
		f, err := os.OpenFile(envPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec
		require.NoError(t, err)
		_, err = f.WriteString(envLine)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	// Write updated foundry.toml
	require.NoError(t, os.WriteFile(foundryPath, []byte(content), 0644))
}

func TestForkCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_enter_success",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"fork", "enter", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify output contains expected info
				assert.Contains(t, output, "Fork mode entered")
				assert.Contains(t, output, "anvil-31337")

				// Verify fork state file exists
				statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
				data, err := os.ReadFile(statePath)
				require.NoError(t, err, "fork-state.json should exist")

				var state domain.ForkState
				require.NoError(t, json.Unmarshal(data, &state))
				require.NotNil(t, state.Forks["anvil-31337"], "fork entry should exist for anvil-31337")

				entry := state.Forks["anvil-31337"]
				assert.Equal(t, "anvil-31337", entry.Network)
				assert.Equal(t, uint64(31337), entry.ChainID)
				assert.NotEmpty(t, entry.ForkURL)
				assert.Equal(t, "ANVIL_31337_RPC_URL", entry.EnvVarName)
				assert.Greater(t, entry.AnvilPID, 0)
				assert.Len(t, entry.Snapshots, 1, "should have initial snapshot")
				assert.Equal(t, 0, entry.Snapshots[0].Index)
				assert.Equal(t, "fork enter", entry.Snapshots[0].Command)

				// Verify registry files backed up to snapshot 0
				snapshotDir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337", "snapshots", "0")
				_, err = os.Stat(snapshotDir)
				assert.NoError(t, err, "snapshot 0 directory should exist")

				// Verify anvil process is running
				assert.True(t, isProcessAlive(entry.AnvilPID), "fork anvil process should be running")
			},
		},
		{
			Name: "fork_enter_already_active",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"fork", "enter", "anvil-31337"},
			},
			ExpectErr:  true,
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				assert.Contains(t, output, "fork already active")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkExitCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_exit_restores_state",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"fork", "exit", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				workDir := ctx.TrebContext.GetWorkDir()

				// Verify output
				assert.Contains(t, output, "Fork mode exited")
				assert.Contains(t, output, "anvil-31337")

				// Verify fork state file is gone (no more active forks)
				statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
				_, err := os.Stat(statePath)
				assert.True(t, os.IsNotExist(err), "fork-state.json should be deleted when no forks remain")

				// Verify fork directory is cleaned up
				forkDir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337")
				_, err = os.Stat(forkDir)
				assert.True(t, os.IsNotExist(err), "fork directory should be cleaned up")
			},
		},
		{
			Name: "fork_exit_after_deploy_restores_registry",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"fork", "exit", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				workDir := ctx.TrebContext.GetWorkDir()

				// Verify fork state is cleaned up
				statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
				_, err := os.Stat(statePath)
				assert.True(t, os.IsNotExist(err), "fork-state.json should be deleted")

				// Verify deployments.json is restored to pre-fork state
				// Before fork, we had no deployments (only gen, no run)
				// After fork exit, either deployments.json doesn't exist (it was created during fork
				// and cleaned up) or it exists but doesn't contain Counter
				deploymentsPath := filepath.Join(workDir, ".treb", "deployments.json")
				data, err := os.ReadFile(deploymentsPath)
				if err == nil {
					// If file exists, Counter should not be present
					assert.NotContains(t, string(data), "Counter",
						"Counter deployment should be reverted after fork exit")
				} else {
					// File doesn't exist - that's valid, it was created during fork and removed
					assert.True(t, os.IsNotExist(err), "unexpected error reading deployments.json: %v", err)
				}
			},
		},
		{
			Name: "fork_exit_no_active_fork",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"fork", "exit", "anvil-31337"},
			},
			ExpectErr:  true,
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				assert.Contains(t, output, "no active fork")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

// cleanupForkAnvil stops the fork anvil process and removes state files
func cleanupForkAnvil(t *testing.T, ctx *helpers.TestContext, network string) {
	t.Helper()
	workDir := ctx.TrebContext.GetWorkDir()

	// Read fork state to get PID
	statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return // No state file, nothing to clean up
	}

	var state domain.ForkState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	entry := state.Forks[network]
	if entry == nil {
		return
	}

	// Kill the fork anvil process
	if entry.AnvilPID > 0 {
		if proc, err := os.FindProcess(entry.AnvilPID); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}

	// Clean up PID file
	if entry.PidFile != "" {
		os.Remove(entry.PidFile)
	}

	// Clean up log file
	if entry.LogFile != "" {
		os.Remove(entry.LogFile)
	}
}

// isProcessAlive checks if a process with the given PID is running
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// readForkState reads and parses the fork state file from a test context
func readForkState(t *testing.T, ctx *helpers.TestContext) *domain.ForkState {
	t.Helper()
	workDir := ctx.TrebContext.GetWorkDir()
	statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")

	data, err := os.ReadFile(statePath)
	require.NoError(t, err, "should be able to read fork-state.json")

	var state domain.ForkState
	require.NoError(t, json.Unmarshal(data, &state))

	if state.Forks == nil {
		state.Forks = make(map[string]*domain.ForkEntry)
	}

	return &state
}

// readPidFromFile reads a PID from a file
func readPidFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

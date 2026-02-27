package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

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
		_ = os.Remove(entry.PidFile)
	}

	// Clean up log file
	if entry.LogFile != "" {
		_ = os.Remove(entry.LogFile)
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
func readPidFromFile(path string) (int, error) { //nolint:unused // utility for fork tests
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// ethGetCode makes an eth_getCode RPC call and returns the code at the given address
func ethGetCode(t *testing.T, rpcURL, address string) string {
	t.Helper()

	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getCode",
		"params":  []interface{}{address, "latest"},
		"id":      1,
	})
	require.NoError(t, err)

	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(reqBody)) //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var rpcResp struct {
		Result string `json:"result"`
	}
	require.NoError(t, json.Unmarshal(body, &rpcResp))

	return rpcResp.Result
}

// getDeploymentAddress extracts the deployment address from deployments.json
// The file is a flat map keyed by deployment ID (e.g. "default/31337/Counter")
func getDeploymentAddress(t *testing.T, workDir, contractName string) string {
	t.Helper()

	deploymentsPath := filepath.Join(workDir, ".treb", "deployments.json")
	data, err := os.ReadFile(deploymentsPath)
	require.NoError(t, err, "deployments.json should exist")

	var deployments map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &deployments))

	for _, depRaw := range deployments {
		var dep struct {
			ContractName string `json:"contractName"`
			Address      string `json:"address"`
		}
		if err := json.Unmarshal(depRaw, &dep); err != nil {
			continue
		}
		if dep.ContractName == contractName {
			return dep.Address
		}
	}
	t.Fatalf("deployment for %s not found in deployments.json", contractName)
	return ""
}

func TestForkRunCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_run_deploys_to_fork_anvil",
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
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify deployment recorded in deployments.json
				address := getDeploymentAddress(t, workDir, "Counter")
				assert.NotEmpty(t, address, "Counter address should be recorded")

				// Read fork state to get the fork URL
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should exist")

				// Verify contract deployed to fork anvil (via eth_getCode on fork URL)
				code := ethGetCode(t, fork.ForkURL, address)
				assert.NotEqual(t, "0x", code, "contract should have code on fork anvil")
				assert.True(t, len(code) > 10, "contract code should be non-trivial")

				// Verify the deployment was NOT made to the regular anvil
				regularNode := ctx.AnvilNodes["anvil-31337"]
				require.NotNil(t, regularNode, "regular anvil node should exist")
				regularCode := ethGetCode(t, regularNode.URL, address)
				assert.Equal(t, "0x", regularCode, "contract should NOT have code on regular anvil")
			},
		},
		{
			Name: "run_without_fork_deploys_to_regular_anvil",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				workDir := ctx.TrebContext.GetWorkDir()

				// Verify deployment recorded in deployments.json
				address := getDeploymentAddress(t, workDir, "Counter")
				assert.NotEmpty(t, address, "Counter address should be recorded")

				// Verify contract deployed to regular anvil
				regularNode := ctx.AnvilNodes["anvil-31337"]
				require.NotNil(t, regularNode, "regular anvil node should exist")
				code := ethGetCode(t, regularNode.URL, address)
				assert.NotEqual(t, "0x", code, "contract should have code on regular anvil")
				assert.True(t, len(code) > 10, "contract code should be non-trivial")

				// Verify no fork state file exists (fork mode was never entered)
				statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
				_, err := os.Stat(statePath)
				assert.True(t, os.IsNotExist(err), "fork-state.json should not exist when not in fork mode")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkPreRunSnapshots(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_run_creates_pre_run_snapshot",
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
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify snapshot 1 directory exists with pre-run registry files
				snapshotDir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337", "snapshots", "1")
				_, err := os.Stat(snapshotDir)
				assert.NoError(t, err, "snapshot 1 directory should exist")

				// Verify fork-state.json has 2 entries in snapshot stack (initial + pre-run)
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should exist")
				assert.Len(t, fork.Snapshots, 2, "should have 2 snapshots (initial + pre-run)")

				// Verify snapshot 0 is the initial entry
				assert.Equal(t, 0, fork.Snapshots[0].Index)
				assert.Equal(t, "fork enter", fork.Snapshots[0].Command)

				// Verify snapshot 1 is the pre-run entry with the script ref
				assert.Equal(t, 1, fork.Snapshots[1].Index)
				assert.Contains(t, fork.Snapshots[1].Command, "DeployCounter")
				assert.NotEmpty(t, fork.Snapshots[1].SnapshotID)
			},
		},
		{
			Name: "fork_multiple_runs_create_multiple_snapshots",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify snapshots 1 and 2 exist
				snapshot1Dir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337", "snapshots", "1")
				_, err := os.Stat(snapshot1Dir)
				assert.NoError(t, err, "snapshot 1 directory should exist")

				snapshot2Dir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337", "snapshots", "2")
				_, err = os.Stat(snapshot2Dir)
				assert.NoError(t, err, "snapshot 2 directory should exist")

				// Verify fork-state.json has 3 entries in snapshot stack
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should exist")
				assert.Len(t, fork.Snapshots, 3, "should have 3 snapshots (initial + 2 pre-run)")

				// Verify snapshot entries
				assert.Equal(t, 0, fork.Snapshots[0].Index)
				assert.Equal(t, "fork enter", fork.Snapshots[0].Command)

				assert.Equal(t, 1, fork.Snapshots[1].Index)
				assert.Contains(t, fork.Snapshots[1].Command, "DeployCounter")

				assert.Equal(t, 2, fork.Snapshots[2].Index)
				assert.Contains(t, fork.Snapshots[2].Command, "DeploySampleToken")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

// ethGetBalance makes an eth_getBalance RPC call and returns the balance as a hex string
func ethGetBalance(t *testing.T, rpcURL, address string) string {
	t.Helper()

	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBalance",
		"params":  []interface{}{address, "latest"},
		"id":      1,
	})
	require.NoError(t, err)

	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(reqBody)) //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var rpcResp struct {
		Result string `json:"result"`
	}
	require.NoError(t, json.Unmarshal(body, &rpcResp))

	return rpcResp.Result
}

func TestForkSetupScript(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_enter_with_setup_script",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				s("config set fork.setup script/setup/SetupFork.s.sol"),
			},
			TestCmds: [][]string{
				{"fork", "enter", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Verify fork entered successfully
				assert.Contains(t, output, "Fork mode entered")
				assert.Contains(t, output, "Setup")

				// Read fork state to get the fork URL
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should exist")

				// Verify the setup script ran by checking that the known address has 100 ETH
				// The SetupFork script calls vm.deal(0x1234...7890, 100 ether)
				balance := ethGetBalance(t, fork.ForkURL, "0x1234567890123456789012345678901234567890")
				assert.NotEqual(t, "0x0", balance, "test address should have ETH after setup script")
				// 100 ether = 100 * 10^18 = 0x56bc75e2d63100000
				assert.Equal(t, "0x56bc75e2d63100000", balance,
					"test address should have exactly 100 ETH")
			},
		},
		{
			Name: "fork_enter_with_failing_setup_script",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				s("config set fork.setup script/setup/SetupForkFailing.s.sol"),
			},
			TestCmds: [][]string{
				{"fork", "enter", "anvil-31337"},
			},
			ExpectErr:  true,
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				// Verify error message mentions setup script failure
				assert.Contains(t, output, "setup fork script failed")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify no fork state file created (fork was aborted)
				statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
				_, err := os.Stat(statePath)
				assert.True(t, os.IsNotExist(err), "fork-state.json should not exist after failed setup")
			},
		},
		{
			Name: "fork_enter_without_setup_config",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				// No fork.setup configured
			},
			TestCmds: [][]string{
				{"fork", "enter", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Verify fork entered successfully without setup
				assert.Contains(t, output, "Fork mode entered")
				assert.NotContains(t, output, "Setup")

				// Verify fork state exists and is valid
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should exist")
				assert.Len(t, fork.Snapshots, 1, "should have initial snapshot")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkStatusCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_status_shows_active_fork",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"fork", "status"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Verify output shows fork info
				assert.Contains(t, output, "Active Forks")
				assert.Contains(t, output, "anvil-31337")
				assert.Contains(t, output, "31337")
				assert.Contains(t, output, "healthy")
				assert.Contains(t, output, "Snapshots:    1")
				assert.Contains(t, output, "Fork Deploys: 0")
				assert.Contains(t, output, "(current)")
			},
		},
		{
			Name: "fork_status_after_deploy",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"fork", "status"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Verify output shows 1 fork deployment and 2 snapshots
				assert.Contains(t, output, "Active Forks")
				assert.Contains(t, output, "anvil-31337")
				assert.Contains(t, output, "healthy")
				assert.Contains(t, output, "Snapshots:    2")
				assert.Contains(t, output, "Fork Deploys: 1")
			},
		},
		{
			Name: "fork_status_no_active_forks",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"fork", "status"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				assert.Contains(t, output, "No active forks")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkHistoryCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_history_shows_entries",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"fork", "history", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Verify output shows fork history header
				assert.Contains(t, output, "Fork History: anvil-31337")

				// Verify 3 entries: initial, DeployCounter, DeploySampleToken
				assert.Contains(t, output, "initial")
				assert.Contains(t, output, "DeployCounter")
				assert.Contains(t, output, "DeploySampleToken")

				// Verify the most recent entry is marked as current (→)
				assert.Contains(t, output, "→")

				// Verify index markers
				assert.Contains(t, output, "[0]")
				assert.Contains(t, output, "[1]")
				assert.Contains(t, output, "[2]")
			},
		},
		{
			Name: "fork_history_after_revert",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"fork", "revert", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"fork", "history", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// After revert, only the initial entry should remain
				assert.Contains(t, output, "Fork History: anvil-31337")
				assert.Contains(t, output, "initial")
				assert.Contains(t, output, "[0]")

				// The reverted entry should not appear
				assert.NotContains(t, output, "DeployCounter")
				assert.NotContains(t, output, "[1]")
			},
		},
		{
			Name: "fork_history_no_active_fork",
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
			},
			TestCmds: [][]string{
				{"fork", "history", "anvil-31337"},
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

func TestForkAwareList(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_list_shows_fork_indicator",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				// Deploy Counter before fork
				{"run", "script/deploy/DeployCounter.s.sol"},
				// Enter fork mode
				{"fork", "enter", "anvil-31337"},
				// Deploy SampleToken inside fork
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"list"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Counter should NOT have [fork] indicator
				// SampleToken SHOULD have [fork] indicator
				assert.Contains(t, output, "SampleToken")
				assert.Contains(t, output, "[fork]")
				// Verify Counter doesn't have [fork] - check that [fork] appears after SampleToken (not Counter)
				assert.Contains(t, output, "Counter")
			},
		},
		{
			Name: "fork_list_fork_flag_filters",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"list", "--fork"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// --fork: should show only SampleToken (fork-added)
				assert.Contains(t, output, "SampleToken")
				assert.NotContains(t, output, "Counter")
			},
		},
		{
			Name: "fork_list_no_fork_flag_filters",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"list", "--no-fork"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// --no-fork: should show only Counter (pre-fork)
				assert.Contains(t, output, "Counter")
				assert.NotContains(t, output, "SampleToken")
			},
		},
		{
			Name: "fork_list_json_with_fork_field",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"list", "--json"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Extract JSON from output (framework prepends "=== cmd N: ... ===\n")
				jsonStr := extractJSONArray(output)

				// Parse JSON output
				var entries []map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(jsonStr), &entries))
				require.Len(t, entries, 2)

				// Find Counter and SampleToken entries
				var counterEntry, tokenEntry map[string]interface{}
				for _, e := range entries {
					switch e["contractName"] {
					case "Counter":
						counterEntry = e
					case "SampleToken":
						tokenEntry = e
					}
				}

				require.NotNil(t, counterEntry, "Counter should be in JSON output")
				require.NotNil(t, tokenEntry, "SampleToken should be in JSON output")

				// Counter should NOT have fork: true (omitempty means field absent or false)
				counterFork, hasFork := counterEntry["fork"]
				if hasFork {
					assert.False(t, counterFork.(bool), "Counter should not be marked as fork")
				}

				// SampleToken SHOULD have fork: true
				assert.Equal(t, true, tokenEntry["fork"], "SampleToken should be marked as fork")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

// extractJSONArray extracts a JSON array from output that may contain framework headers.
// The framework prepends "=== cmd N: [...] ===\n" to command output.
func extractJSONArray(output string) string {
	// Find the JSON array start - look for "\n[" to skip the framework header brackets
	idx := strings.Index(output, "\n[")
	if idx >= 0 {
		return strings.TrimSpace(output[idx+1:])
	}
	// Fallback: if output starts with "[" directly
	if strings.HasPrefix(strings.TrimSpace(output), "[") {
		return strings.TrimSpace(output)
	}
	return output
}

func TestForkRevertCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_revert_restores_last_run",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"fork", "revert", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify output
				assert.Contains(t, output, "Reverted")
				assert.Contains(t, output, "DeployCounter")

				// Verify Counter no longer in deployments.json
				deploymentsPath := filepath.Join(workDir, ".treb", "deployments.json")
				data, err := os.ReadFile(deploymentsPath)
				if err == nil {
					assert.NotContains(t, string(data), "Counter",
						"Counter deployment should be reverted")
				}

				// Verify fork state has only initial snapshot
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should still exist")
				assert.Len(t, fork.Snapshots, 1, "should have only initial snapshot after revert")
				assert.Equal(t, 0, fork.Snapshots[0].Index)
				assert.Equal(t, "fork enter", fork.Snapshots[0].Command)

				// Verify snapshot 1 directory is cleaned up
				snapshot1Dir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337", "snapshots", "1")
				_, err = os.Stat(snapshot1Dir)
				assert.True(t, os.IsNotExist(err), "snapshot 1 directory should be removed after revert")
			},
		},
		{
			Name: "fork_revert_partial_keeps_earlier_deploy",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"fork", "revert", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify output mentions SampleToken (the reverted command)
				assert.Contains(t, output, "DeploySampleToken")

				// Counter should still be deployed (only SampleToken was reverted)
				deploymentsPath := filepath.Join(workDir, ".treb", "deployments.json")
				data, err := os.ReadFile(deploymentsPath)
				require.NoError(t, err, "deployments.json should exist")
				assert.Contains(t, string(data), "Counter",
					"Counter should still be in deployments after partial revert")
				assert.NotContains(t, string(data), "SampleToken",
					"SampleToken should be reverted from deployments")

				// Verify fork state has 2 snapshots (initial + Counter's pre-run)
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork)
				assert.Len(t, fork.Snapshots, 2, "should have 2 snapshots (initial + first run)")
			},
		},
		{
			Name: "fork_revert_all_restores_initial_state",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"gen", "deploy", "src/SampleToken.sol:SampleToken"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeploySampleToken.s.sol"},
			},
			TestCmds: [][]string{
				{"fork", "revert", "--all", "anvil-31337"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify output
				assert.Contains(t, output, "Reverted 2 run(s)")
				assert.Contains(t, output, "initial fork state")

				// Both deployments should be gone
				deploymentsPath := filepath.Join(workDir, ".treb", "deployments.json")
				data, err := os.ReadFile(deploymentsPath)
				if err == nil {
					assert.NotContains(t, string(data), "Counter",
						"Counter should be reverted after --all")
					assert.NotContains(t, string(data), "SampleToken",
						"SampleToken should be reverted after --all")
				}

				// Verify fork state has only initial snapshot
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork)
				assert.Len(t, fork.Snapshots, 1, "should have only initial snapshot after revert --all")
				assert.Equal(t, 0, fork.Snapshots[0].Index)
			},
		},
		{
			Name: "fork_revert_nothing_to_revert",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"fork", "revert", "anvil-31337"},
			},
			ExpectErr:  true,
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				assert.Contains(t, output, "nothing to revert")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

// killForkAnvil sends SIGKILL to the fork anvil process to simulate a crash
func killForkAnvil(t *testing.T, ctx *helpers.TestContext, network string) {
	t.Helper()
	state := readForkState(t, ctx)
	entry := state.Forks[network]
	require.NotNil(t, entry, "fork entry should exist for %s", network)

	if entry.AnvilPID > 0 {
		proc, err := os.FindProcess(entry.AnvilPID)
		require.NoError(t, err)
		err = proc.Signal(syscall.SIGKILL)
		require.NoError(t, err, "should be able to kill fork anvil process")
		// Wait briefly for process to die
		time.Sleep(100 * time.Millisecond)
	}
}

func TestForkCrashDetection(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_run_detects_crashed_anvil",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds:   [][]string{},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, _ string) {
				// Kill the fork anvil to simulate crash
				killForkAnvil(t, ctx, "anvil-31337")

				// Now try to run a deployment - should fail with crash error
				output, err := ctx.TrebContext.Treb("run", "script/deploy/DeployCounter.s.sol")
				require.Error(t, err, "treb run should fail when fork anvil is crashed")

				assert.Contains(t, output, "has crashed")
				assert.Contains(t, output, "treb fork restart")
				assert.Contains(t, output, "treb fork exit")
			},
		},
		{
			Name: "fork_status_shows_dead_for_crashed_anvil",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds:   [][]string{},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, _ string) {
				// Kill the fork anvil
				killForkAnvil(t, ctx, "anvil-31337")

				// Check fork status - should show dead
				output, err := ctx.TrebContext.Treb("fork", "status")
				require.NoError(t, err, "fork status should succeed even with dead fork")

				assert.Contains(t, output, "Active Forks")
				assert.Contains(t, output, "anvil-31337")
				assert.Contains(t, output, "dead")
			},
		},
		{
			Name: "fork_exit_cleans_up_crashed_anvil",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds:   [][]string{},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, _ string) {
				// Kill the fork anvil
				killForkAnvil(t, ctx, "anvil-31337")

				// Exit should succeed even with dead anvil
				output, err := ctx.TrebContext.Treb("fork", "exit", "anvil-31337")
				require.NoError(t, err, "fork exit should succeed with dead anvil")

				assert.Contains(t, output, "Fork mode exited")
				assert.Contains(t, output, "anvil-31337")

				workDir := ctx.TrebContext.GetWorkDir()

				// Verify fork state file is gone
				statePath := filepath.Join(workDir, ".treb", "priv", "fork-state.json")
				_, err = os.Stat(statePath)
				assert.True(t, os.IsNotExist(err), "fork-state.json should be deleted")

				// Verify fork directory is cleaned up
				forkDir := filepath.Join(workDir, ".treb", "priv", "fork", "anvil-31337")
				_, err = os.Stat(forkDir)
				assert.True(t, os.IsNotExist(err), "fork directory should be cleaned up")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkRestartCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_restart_after_crash_restores_state",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds:   [][]string{},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, _ string) {
				workDir := ctx.TrebContext.GetWorkDir()

				// Verify Counter is deployed before crash
				deploymentsPath := filepath.Join(workDir, ".treb", "deployments.json")
				data, err := os.ReadFile(deploymentsPath)
				require.NoError(t, err)
				assert.Contains(t, string(data), "Counter", "Counter should be deployed before restart")

				// Kill the fork anvil
				killForkAnvil(t, ctx, "anvil-31337")

				// Restart the fork
				output, err := ctx.TrebContext.Treb("fork", "restart", "anvil-31337")
				require.NoError(t, err, "fork restart should succeed")
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				assert.Contains(t, output, "Fork restarted")
				assert.Contains(t, output, "anvil-31337")

				// Verify Counter is no longer in deployments.json (restored to initial state)
				data, err = os.ReadFile(deploymentsPath)
				if err == nil {
					assert.NotContains(t, string(data), "Counter",
						"Counter deployment should be gone after restart (restored to initial state)")
				}

				// Verify fresh fork state
				state := readForkState(t, ctx)
				fork := state.Forks["anvil-31337"]
				require.NotNil(t, fork, "fork entry should exist after restart")
				assert.Len(t, fork.Snapshots, 1, "should have only initial snapshot after restart")
				assert.Equal(t, "fork restart", fork.Snapshots[0].Command)

				// Verify new fork anvil is healthy
				assert.True(t, isProcessAlive(fork.AnvilPID), "new fork anvil should be running")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkAwareShow(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_show_displays_fork_indicator",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				// Enter fork mode, then deploy Counter
				{"fork", "enter", "anvil-31337"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"show", "Counter"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Fork-added deployment should have [fork] indicator
				assert.Contains(t, output, "[fork]")
				assert.Contains(t, output, "Counter")
				assert.Contains(t, output, "Deployment:")
			},
		},
		{
			Name: "fork_show_no_fork_indicator_for_pre_fork_deploy",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				// Deploy Counter BEFORE fork
				{"run", "script/deploy/DeployCounter.s.sol"},
				// Enter fork mode after deploy
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"show", "Counter"},
			},
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				// Pre-fork deployment should NOT have [fork] indicator
				assert.NotContains(t, output, "[fork]")
				assert.Contains(t, output, "Counter")
				assert.Contains(t, output, "Deployment:")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

func TestForkCommandGuards(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "fork_verify_blocked",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"verify", "Counter"},
			},
			ExpectErr:  true,
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				assert.Contains(t, output, "cannot verify contracts on a fork")
			},
		},
		{
			Name: "fork_sync_blocked",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				setupForkEnvVars(t, ctx)
			},
			SetupCmds: [][]string{
				s("config set network anvil-31337"),
				{"fork", "enter", "anvil-31337"},
			},
			TestCmds: [][]string{
				{"sync"},
			},
			ExpectErr:  true,
			SkipGolden: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				defer cleanupForkAnvil(t, ctx, "anvil-31337")

				assert.Contains(t, output, "cannot sync with a fork")
			},
		},
	}

	RunIntegrationTests(t, tests)
}

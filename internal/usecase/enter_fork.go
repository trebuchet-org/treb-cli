package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// EnterFork handles entering fork mode for a network
type EnterFork struct {
	cfg          *config.RuntimeConfig
	forkState    ForkStateStore
	forkFiles    ForkFileManager
	anvilManager AnvilManager
	forgeRunner  ForgeScriptRunner
}

// NewEnterFork creates a new EnterFork use case
func NewEnterFork(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
	forkFiles ForkFileManager,
	anvilManager AnvilManager,
	forgeRunner ForgeScriptRunner,
) *EnterFork {
	return &EnterFork{
		cfg:          cfg,
		forkState:    forkState,
		forkFiles:    forkFiles,
		anvilManager: anvilManager,
		forgeRunner:  forgeRunner,
	}
}

// EnterForkParams contains parameters for entering fork mode
type EnterForkParams struct {
	Network         string // network name from foundry.toml
	RPCURL          string // resolved RPC URL (after env var expansion)
	ChainID         uint64 // chain ID
	EnvVarName      string // env var name that foundry.toml uses for the RPC endpoint
	ExternalURL     string // optional external Anvil endpoint URL (skips local Anvil startup)
	ForkBlockNumber uint64 // optional block number to fork at (0 = latest)
}

// EnterForkResult contains the result of entering fork mode
type EnterForkResult struct {
	ForkEntry      *domain.ForkEntry
	Message        string
	SetupScriptRan bool
}

// Execute enters fork mode for the specified network
func (uc *EnterFork) Execute(ctx context.Context, params EnterForkParams) (*EnterForkResult, error) {
	// Load current fork state
	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load fork state: %w", err)
	}

	// Check if fork is already active for this network
	if state.IsForkActive(params.Network) {
		return nil, fmt.Errorf("fork already active for network '%s'. Run 'treb fork exit %s' first", params.Network, params.Network)
	}

	if params.ExternalURL != "" {
		return uc.executeExternal(ctx, state, params)
	}
	return uc.executeLocal(ctx, state, params)
}

// executeExternal enters fork mode using an already-running external Anvil endpoint
func (uc *EnterFork) executeExternal(ctx context.Context, state *domain.ForkState, params EnterForkParams) (*EnterForkResult, error) {
	forkURL := params.ExternalURL

	// Validate endpoint by calling evm_snapshot — this doubles as taking the initial snapshot
	snapshotID, err := evmSnapshot(forkURL)
	if err != nil {
		return nil, fmt.Errorf("external endpoint validation failed — evm_snapshot not supported at %s: %w", forkURL, err)
	}

	// Backup registry files to snapshot 0
	if err := uc.forkFiles.BackupFiles(ctx, params.Network, 0); err != nil {
		return nil, fmt.Errorf("failed to backup registry files: %w", err)
	}

	// Build fork entry for external fork
	entry := &domain.ForkEntry{
		Network:     params.Network,
		ChainID:     params.ChainID,
		EnvVarName:  params.EnvVarName,
		OriginalRPC: params.RPCURL,
		ForkURL:     forkURL,
		AnvilPID:    0,
		PidFile:     "",
		LogFile:     "",
		EnteredAt:   time.Now(),
		External:    true,
		Snapshots: []domain.SnapshotEntry{
			{
				Index:      0,
				SnapshotID: snapshotID,
				Command:    "fork enter",
				Timestamp:  time.Now(),
			},
		},
	}

	// Save fork state
	state.Forks[params.Network] = entry
	if err := uc.forkState.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save fork state: %w", err)
	}

	return &EnterForkResult{
		ForkEntry:      entry,
		Message:        fmt.Sprintf("Fork mode entered for network '%s' (external)", params.Network),
		SetupScriptRan: false,
	}, nil
}

// executeLocal enters fork mode by starting a local Anvil instance
func (uc *EnterFork) executeLocal(ctx context.Context, state *domain.ForkState, params EnterForkParams) (*EnterForkResult, error) {
	// Find available port
	port, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Build anvil instance for fork - PID/log files under .treb/priv/ for project scoping
	privDir := filepath.Join(uc.cfg.DataDir, "priv")
	if err := os.MkdirAll(privDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create priv directory: %w", err)
	}

	instance := &domain.AnvilInstance{
		Name:            fmt.Sprintf("fork-%s", params.Network),
		Port:            fmt.Sprintf("%d", port),
		ChainID:         fmt.Sprintf("%d", params.ChainID),
		ForkURL:         params.RPCURL,
		ForkBlockNumber: params.ForkBlockNumber,
		PidFile:         filepath.Join(privDir, fmt.Sprintf("fork-%s.pid", params.Network)),
		LogFile:         filepath.Join(privDir, fmt.Sprintf("fork-%s.log", params.Network)),
	}

	// Start anvil (includes CreateX deployment)
	if err := uc.anvilManager.Start(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to start fork anvil: %w", err)
	}

	// Verify anvil is healthy
	status, err := uc.anvilManager.GetStatus(ctx, instance)
	if err != nil || !status.Running || !status.RPCHealthy {
		// Clean up on failure
		_ = uc.anvilManager.Stop(ctx, instance)
		if err != nil {
			return nil, fmt.Errorf("failed to verify fork anvil health: %w", err)
		}
		return nil, fmt.Errorf("fork anvil started but is not healthy")
	}

	// Build fork env overrides for setup script execution
	forkURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	forkEnvOverrides := map[string]string{
		params.EnvVarName: forkURL,
	}

	// Execute SetupFork script if configured
	setupScriptRan, err := uc.executeSetupFork(ctx, params, forkEnvOverrides)
	if err != nil {
		_ = uc.anvilManager.Stop(ctx, instance)
		return nil, fmt.Errorf("setup fork script failed: %w", err)
	}

	// Backup registry files to snapshot 0 (AFTER SetupFork so snapshot captures setup state)
	if err := uc.forkFiles.BackupFiles(ctx, params.Network, 0); err != nil {
		_ = uc.anvilManager.Stop(ctx, instance)
		return nil, fmt.Errorf("failed to backup registry files: %w", err)
	}

	// Take initial EVM snapshot (AFTER SetupFork so snapshot captures setup state)
	snapshotID, err := uc.anvilManager.TakeSnapshot(ctx, instance)
	if err != nil {
		_ = uc.anvilManager.Stop(ctx, instance)
		return nil, fmt.Errorf("failed to take initial EVM snapshot: %w", err)
	}

	// Build fork entry
	entry := &domain.ForkEntry{
		Network:     params.Network,
		ChainID:     params.ChainID,
		EnvVarName:  params.EnvVarName,
		OriginalRPC: params.RPCURL,
		ForkURL:     forkURL,
		AnvilPID:    status.PID,
		PidFile:     instance.PidFile,
		LogFile:     instance.LogFile,
		EnteredAt:   time.Now(),
		Snapshots: []domain.SnapshotEntry{
			{
				Index:      0,
				SnapshotID: snapshotID,
				Command:    "fork enter",
				Timestamp:  time.Now(),
			},
		},
	}

	// Save fork state
	state.Forks[params.Network] = entry
	if err := uc.forkState.Save(ctx, state); err != nil {
		_ = uc.anvilManager.Stop(ctx, instance)
		return nil, fmt.Errorf("failed to save fork state: %w", err)
	}

	// Add .treb/priv/ to .gitignore
	if err := ensureGitignoreEntry(uc.cfg.ProjectRoot, ".treb/priv/"); err != nil {
		// Non-fatal - just warn
		fmt.Fprintf(os.Stderr, "Warning: failed to update .gitignore: %v\n", err)
	}

	return &EnterForkResult{
		ForkEntry:      entry,
		Message:        fmt.Sprintf("Fork mode entered for network '%s'", params.Network),
		SetupScriptRan: setupScriptRan,
	}, nil
}

// executeSetupFork runs the configured fork setup script if it exists.
// Returns (true, nil) if a script was executed successfully,
// (false, nil) if no script is configured or file doesn't exist,
// (false, error) if execution failed.
func (uc *EnterFork) executeSetupFork(ctx context.Context, params EnterForkParams, forkEnvOverrides map[string]string) (bool, error) {
	if uc.cfg.ForkSetup == "" {
		return false, nil
	}

	// Check if the script file exists
	scriptPath := uc.cfg.ForkSetup
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(uc.cfg.ProjectRoot, scriptPath)
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// Script file doesn't exist - skip silently
		return false, nil
	}

	// Build minimal network config for the script
	network := &config.Network{
		Name:    params.Network,
		ChainID: params.ChainID,
		RPCURL:  params.RPCURL,
	}

	// Build minimal contract for the script (just needs the path)
	script := &models.Contract{
		Name: filepath.Base(uc.cfg.ForkSetup),
		Path: uc.cfg.ForkSetup,
	}

	// Execute the setup script with fork env overrides
	runConfig := RunScriptConfig{
		Script:           script,
		Network:          network,
		Namespace:        uc.cfg.Namespace,
		Parameters:       map[string]string{},
		DryRun:           false,
		Debug:            false,
		ForkEnvOverrides: forkEnvOverrides,
	}

	result, err := uc.forgeRunner.RunScript(ctx, runConfig)
	if err != nil {
		return false, fmt.Errorf("failed to execute setup script '%s': %w", uc.cfg.ForkSetup, err)
	}

	if !result.Success {
		return false, fmt.Errorf("setup script '%s' failed", uc.cfg.ForkSetup)
	}

	return true, nil
}

// getAvailablePort finds an available TCP port
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port, nil
}

// evmSnapshot calls evm_snapshot on an arbitrary RPC endpoint and returns the snapshot ID.
// Used for external fork validation — if the endpoint doesn't support evm_snapshot, it's not a valid Anvil fork.
func evmSnapshot(rpcURL string) (string, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "evm_snapshot",
		"params":  []interface{}{},
		"id":      1,
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(reqBody)) //nolint:gosec // user-provided URL for fork endpoint
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return "", fmt.Errorf("invalid JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	snapshotID, ok := rpcResp.Result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected evm_snapshot response type: %T", rpcResp.Result)
	}

	return snapshotID, nil
}

// ensureGitignoreEntry adds an entry to .gitignore if not already present
func ensureGitignoreEntry(projectRoot, entry string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	data, err := os.ReadFile(gitignorePath) //nolint:gosec // internal path
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	content := string(data)
	// Check if entry already present
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return nil // Already present
		}
	}

	// Append entry
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // internal path
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer f.Close()

	prefix := ""
	if len(data) > 0 && data[len(data)-1] != '\n' {
		prefix = "\n"
	}

	if _, err := fmt.Fprintf(f, "%s%s\n", prefix, entry); err != nil {
		return fmt.Errorf("failed to write to .gitignore: %w", err)
	}

	return nil
}

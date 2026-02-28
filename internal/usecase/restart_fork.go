package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// RestartFork handles restarting a crashed fork
type RestartFork struct {
	cfg          *config.RuntimeConfig
	forkState    ForkStateStore
	forkFiles    ForkFileManager
	anvilManager AnvilManager
	forgeRunner  ForgeScriptRunner
}

// NewRestartFork creates a new RestartFork use case
func NewRestartFork(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
	forkFiles ForkFileManager,
	anvilManager AnvilManager,
	forgeRunner ForgeScriptRunner,
) *RestartFork {
	return &RestartFork{
		cfg:          cfg,
		forkState:    forkState,
		forkFiles:    forkFiles,
		anvilManager: anvilManager,
		forgeRunner:  forgeRunner,
	}
}

// RestartForkParams contains parameters for restarting a fork
type RestartForkParams struct {
	Network string // network name (empty = use current configured network)
}

// RestartForkResult contains the result of restarting a fork
type RestartForkResult struct {
	ForkEntry      *domain.ForkEntry
	SetupScriptRan bool
	Message        string
}

// Execute restarts a fork: stops dead process, restores files from initial backup,
// starts fresh fork, re-runs SetupFork if configured, takes new initial snapshot.
func (uc *RestartFork) Execute(ctx context.Context, params RestartForkParams) (*RestartForkResult, error) {
	if params.Network == "" {
		return nil, fmt.Errorf("no network specified. Use 'treb fork restart <network>' or configure a network")
	}

	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load fork state: %w", err)
	}

	entry := state.GetActiveFork(params.Network)
	if entry == nil {
		return nil, fmt.Errorf("no active fork for network '%s'", params.Network)
	}

	// Stop existing anvil process (may already be dead)
	instance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", entry.Network),
		Port:    portFromURL(entry.ForkURL),
		ChainID: fmt.Sprintf("%d", entry.ChainID),
		PidFile: entry.PidFile,
		LogFile: entry.LogFile,
	}
	_ = uc.anvilManager.Stop(ctx, instance)

	// Restore registry files from initial backup (snapshot 0)
	if err := uc.forkFiles.RestoreFiles(ctx, entry.Network, 0); err != nil {
		return nil, fmt.Errorf("failed to restore registry files: %w", err)
	}

	// Clean up all snapshot directories except 0
	for i := len(entry.Snapshots) - 1; i >= 1; i-- {
		snapshotDir := fmt.Sprintf("%s/.treb/priv/fork/%s/snapshots/%d", uc.cfg.ProjectRoot, entry.Network, entry.Snapshots[i].Index)
		_ = os.RemoveAll(snapshotDir)
	}
	// Also clean snapshot 0 - we'll re-create it after fresh fork
	snapshot0Dir := fmt.Sprintf("%s/.treb/priv/fork/%s/snapshots/0", uc.cfg.ProjectRoot, entry.Network)
	_ = os.RemoveAll(snapshot0Dir)

	// Find a new available port
	port, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Start fresh fork anvil
	newInstance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", entry.Network),
		Port:    fmt.Sprintf("%d", port),
		ChainID: fmt.Sprintf("%d", entry.ChainID),
		ForkURL: entry.OriginalRPC,
		PidFile: entry.PidFile,
		LogFile: entry.LogFile,
	}

	if err := uc.anvilManager.Start(ctx, newInstance); err != nil {
		return nil, fmt.Errorf("failed to start fresh fork anvil: %w", err)
	}

	// Verify anvil is healthy
	status, err := uc.anvilManager.GetStatus(ctx, newInstance)
	if err != nil || !status.Running || !status.RPCHealthy {
		_ = uc.anvilManager.Stop(ctx, newInstance)
		if err != nil {
			return nil, fmt.Errorf("failed to verify fresh fork anvil health: %w", err)
		}
		return nil, fmt.Errorf("fresh fork anvil started but is not healthy")
	}

	forkURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	forkEnvOverrides := map[string]string{
		entry.EnvVarName: forkURL,
	}

	// Re-run SetupFork script if configured
	setupScriptRan := false
	setupScriptRan, err = uc.executeSetupFork(ctx, entry, forkEnvOverrides)
	if err != nil {
		_ = uc.anvilManager.Stop(ctx, newInstance)
		return nil, fmt.Errorf("setup fork script failed during restart: %w", err)
	}

	// Backup registry files to new snapshot 0 (after SetupFork)
	if err := uc.forkFiles.BackupFiles(ctx, entry.Network, 0); err != nil {
		_ = uc.anvilManager.Stop(ctx, newInstance)
		return nil, fmt.Errorf("failed to backup registry files: %w", err)
	}

	// Take new initial EVM snapshot
	snapshotID, err := uc.anvilManager.TakeSnapshot(ctx, newInstance)
	if err != nil {
		_ = uc.anvilManager.Stop(ctx, newInstance)
		return nil, fmt.Errorf("failed to take initial EVM snapshot: %w", err)
	}

	// Update fork entry with new state
	entry.ForkURL = forkURL
	entry.AnvilPID = status.PID
	entry.EnteredAt = time.Now()
	entry.Snapshots = []domain.SnapshotEntry{
		{
			Index:      0,
			SnapshotID: snapshotID,
			Command:    "fork restart",
			Timestamp:  time.Now(),
		},
	}

	// Save updated state
	if err := uc.forkState.Save(ctx, state); err != nil {
		_ = uc.anvilManager.Stop(ctx, newInstance)
		return nil, fmt.Errorf("failed to save fork state: %w", err)
	}

	return &RestartForkResult{
		ForkEntry:      entry,
		SetupScriptRan: setupScriptRan,
		Message:        fmt.Sprintf("Fork restarted for network '%s'", entry.Network),
	}, nil
}

// executeSetupFork runs the configured fork setup script if it exists.
func (uc *RestartFork) executeSetupFork(ctx context.Context, entry *domain.ForkEntry, forkEnvOverrides map[string]string) (bool, error) {
	if uc.cfg.ForkSetup == "" {
		return false, nil
	}

	scriptPath := uc.cfg.ForkSetup
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(uc.cfg.ProjectRoot, scriptPath)
	}

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return false, nil
	}

	network := &config.Network{
		Name:    entry.Network,
		ChainID: entry.ChainID,
		RPCURL:  entry.OriginalRPC,
	}

	script := &models.Contract{
		Name: filepath.Base(uc.cfg.ForkSetup),
		Path: uc.cfg.ForkSetup,
	}

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

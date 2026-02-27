package usecase

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// EnterFork handles entering fork mode for a network
type EnterFork struct {
	cfg          *config.RuntimeConfig
	forkState    ForkStateStore
	forkFiles    ForkFileManager
	anvilManager AnvilManager
}

// NewEnterFork creates a new EnterFork use case
func NewEnterFork(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
	forkFiles ForkFileManager,
	anvilManager AnvilManager,
) *EnterFork {
	return &EnterFork{
		cfg:          cfg,
		forkState:    forkState,
		forkFiles:    forkFiles,
		anvilManager: anvilManager,
	}
}

// EnterForkParams contains parameters for entering fork mode
type EnterForkParams struct {
	Network    string // network name from foundry.toml
	RPCURL     string // resolved RPC URL (after env var expansion)
	ChainID    uint64 // chain ID
	EnvVarName string // env var name that foundry.toml uses for the RPC endpoint
}

// EnterForkResult contains the result of entering fork mode
type EnterForkResult struct {
	ForkEntry *domain.ForkEntry
	Message   string
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
		Name:    fmt.Sprintf("fork-%s", params.Network),
		Port:    fmt.Sprintf("%d", port),
		ChainID: fmt.Sprintf("%d", params.ChainID),
		ForkURL: params.RPCURL,
		PidFile: filepath.Join(privDir, fmt.Sprintf("fork-%s.pid", params.Network)),
		LogFile: filepath.Join(privDir, fmt.Sprintf("fork-%s.log", params.Network)),
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

	// Backup registry files to snapshot 0
	if err := uc.forkFiles.BackupFiles(ctx, params.Network, 0); err != nil {
		_ = uc.anvilManager.Stop(ctx, instance)
		return nil, fmt.Errorf("failed to backup registry files: %w", err)
	}

	// Take initial EVM snapshot
	snapshotID, err := uc.anvilManager.TakeSnapshot(ctx, instance)
	if err != nil {
		_ = uc.anvilManager.Stop(ctx, instance)
		return nil, fmt.Errorf("failed to take initial EVM snapshot: %w", err)
	}

	// Build fork entry
	forkURL := fmt.Sprintf("http://127.0.0.1:%d", port)
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
		ForkEntry: entry,
		Message:   fmt.Sprintf("Fork mode entered for network '%s'", params.Network),
	}, nil
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

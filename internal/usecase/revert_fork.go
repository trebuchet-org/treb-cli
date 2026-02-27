package usecase

import (
	"context"
	"fmt"
	"os"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// RevertFork handles reverting the last treb run in fork mode
type RevertFork struct {
	cfg          *config.RuntimeConfig
	forkState    ForkStateStore
	forkFiles    ForkFileManager
	anvilManager AnvilManager
}

// NewRevertFork creates a new RevertFork use case
func NewRevertFork(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
	forkFiles ForkFileManager,
	anvilManager AnvilManager,
) *RevertFork {
	return &RevertFork{
		cfg:          cfg,
		forkState:    forkState,
		forkFiles:    forkFiles,
		anvilManager: anvilManager,
	}
}

// RevertForkParams contains parameters for reverting fork state
type RevertForkParams struct {
	Network string // network name (empty = use current configured network)
	All     bool   // revert to initial state (revert all runs)
}

// RevertForkResult contains the result of reverting fork state
type RevertForkResult struct {
	RevertedCommand    string // the command that was reverted (single revert)
	RevertedCount      int    // number of snapshots reverted
	RemainingSnapshots int    // number of snapshots remaining
	Message            string
}

// Execute reverts the last treb run (or all runs) in fork mode
func (uc *RevertFork) Execute(ctx context.Context, params RevertForkParams) (*RevertForkResult, error) {
	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load fork state: %w", err)
	}

	if params.Network == "" {
		return nil, fmt.Errorf("no network specified. Use 'treb fork revert <network>' or configure a network")
	}

	entry := state.GetActiveFork(params.Network)
	if entry == nil {
		return nil, fmt.Errorf("no active fork for network '%s'", params.Network)
	}

	if params.All {
		return uc.revertAll(ctx, state, entry)
	}

	return uc.revertLast(ctx, state, entry)
}

// revertLast reverts the most recent treb run
func (uc *RevertFork) revertLast(ctx context.Context, state *domain.ForkState, entry *domain.ForkEntry) (*RevertForkResult, error) {
	if len(entry.Snapshots) <= 1 {
		return nil, fmt.Errorf("nothing to revert - already at initial fork state")
	}

	// Pop the top entry from the snapshot stack
	topSnapshot := entry.Snapshots[len(entry.Snapshots)-1]

	// Revert EVM state to this snapshot's ID
	instance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", entry.Network),
		Port:    portFromURL(entry.ForkURL),
		ChainID: fmt.Sprintf("%d", entry.ChainID),
		PidFile: entry.PidFile,
		LogFile: entry.LogFile,
	}

	if err := uc.anvilManager.RevertSnapshot(ctx, instance, topSnapshot.SnapshotID); err != nil {
		return nil, fmt.Errorf("failed to revert EVM snapshot: %w", err)
	}

	// Restore registry files from the popped snapshot's directory
	if err := uc.forkFiles.RestoreFiles(ctx, entry.Network, topSnapshot.Index); err != nil {
		return nil, fmt.Errorf("failed to restore registry files: %w", err)
	}

	// Remove the snapshot directory after restore
	snapshotDir := fmt.Sprintf("%s/.treb/priv/fork/%s/snapshots/%d", uc.cfg.ProjectRoot, entry.Network, topSnapshot.Index)
	_ = os.RemoveAll(snapshotDir)

	// Update snapshot stack - remove the top entry
	entry.Snapshots = entry.Snapshots[:len(entry.Snapshots)-1]

	// Save updated fork state
	if err := uc.forkState.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save fork state: %w", err)
	}

	return &RevertForkResult{
		RevertedCommand:    topSnapshot.Command,
		RevertedCount:      1,
		RemainingSnapshots: len(entry.Snapshots),
		Message:            fmt.Sprintf("Reverted '%s' on fork '%s'", topSnapshot.Command, entry.Network),
	}, nil
}

// revertAll reverts to the initial fork state (snapshot 0)
func (uc *RevertFork) revertAll(ctx context.Context, state *domain.ForkState, entry *domain.ForkEntry) (*RevertForkResult, error) {
	if len(entry.Snapshots) <= 1 {
		return nil, fmt.Errorf("nothing to revert - already at initial fork state")
	}

	// Revert EVM state to the initial snapshot (index 0)
	initialSnapshot := entry.Snapshots[0]

	instance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", entry.Network),
		Port:    portFromURL(entry.ForkURL),
		ChainID: fmt.Sprintf("%d", entry.ChainID),
		PidFile: entry.PidFile,
		LogFile: entry.LogFile,
	}

	if err := uc.anvilManager.RevertSnapshot(ctx, instance, initialSnapshot.SnapshotID); err != nil {
		return nil, fmt.Errorf("failed to revert EVM snapshot to initial state: %w", err)
	}

	// Restore registry files from initial backup (snapshot 0)
	if err := uc.forkFiles.RestoreFiles(ctx, entry.Network, 0); err != nil {
		return nil, fmt.Errorf("failed to restore registry files: %w", err)
	}

	// Remove all snapshot directories except 0
	revertedCount := len(entry.Snapshots) - 1
	for i := len(entry.Snapshots) - 1; i >= 1; i-- {
		snapshotDir := fmt.Sprintf("%s/.treb/priv/fork/%s/snapshots/%d", uc.cfg.ProjectRoot, entry.Network, entry.Snapshots[i].Index)
		_ = os.RemoveAll(snapshotDir)
	}

	// Keep only the initial snapshot
	entry.Snapshots = entry.Snapshots[:1]

	// Save updated fork state
	if err := uc.forkState.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save fork state: %w", err)
	}

	return &RevertForkResult{
		RevertedCount:      revertedCount,
		RemainingSnapshots: 1,
		Message:            fmt.Sprintf("Reverted %d run(s) on fork '%s' - restored to initial fork state", revertedCount, entry.Network),
	}, nil
}

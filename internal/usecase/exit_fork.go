package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// ExitFork handles exiting fork mode for a network
type ExitFork struct {
	cfg          *config.RuntimeConfig
	forkState    ForkStateStore
	forkFiles    ForkFileManager
	anvilManager AnvilManager
}

// NewExitFork creates a new ExitFork use case
func NewExitFork(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
	forkFiles ForkFileManager,
	anvilManager AnvilManager,
) *ExitFork {
	return &ExitFork{
		cfg:          cfg,
		forkState:    forkState,
		forkFiles:    forkFiles,
		anvilManager: anvilManager,
	}
}

// ExitForkParams contains parameters for exiting fork mode
type ExitForkParams struct {
	Network string // network name (empty = use current configured network)
	All     bool   // exit all active forks
}

// ExitForkResult contains the result of exiting fork mode
type ExitForkResult struct {
	ExitedNetworks []string
	Message        string
}

// Execute exits fork mode for the specified network(s)
func (uc *ExitFork) Execute(ctx context.Context, params ExitForkParams) (*ExitForkResult, error) {
	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load fork state: %w", err)
	}

	if params.All {
		return uc.exitAll(ctx, state)
	}

	if params.Network == "" {
		return nil, fmt.Errorf("no network specified. Use 'treb fork exit <network>' or 'treb fork exit --all'")
	}

	return uc.exitNetwork(ctx, state, params.Network)
}

// exitNetwork exits fork mode for a single network
func (uc *ExitFork) exitNetwork(ctx context.Context, state *domain.ForkState, network string) (*ExitForkResult, error) {
	entry := state.GetActiveFork(network)
	if entry == nil {
		return nil, fmt.Errorf("no active fork for network '%s'", network)
	}

	if err := uc.cleanupFork(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to exit fork for '%s': %w", network, err)
	}

	// Remove fork entry from state
	delete(state.Forks, network)

	// Save or delete state
	if err := uc.saveOrDeleteState(ctx, state); err != nil {
		return nil, err
	}

	return &ExitForkResult{
		ExitedNetworks: []string{network},
		Message:        fmt.Sprintf("Fork mode exited for network '%s'", network),
	}, nil
}

// exitAll exits fork mode for all active networks
func (uc *ExitFork) exitAll(ctx context.Context, state *domain.ForkState) (*ExitForkResult, error) {
	if len(state.Forks) == 0 {
		return nil, fmt.Errorf("no active forks")
	}

	var exitedNetworks []string
	var errors []string

	for network, entry := range state.Forks {
		if err := uc.cleanupFork(ctx, entry); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", network, err))
			continue
		}
		exitedNetworks = append(exitedNetworks, network)
		delete(state.Forks, network)
	}

	// Save or delete state
	if err := uc.saveOrDeleteState(ctx, state); err != nil {
		return nil, err
	}

	if len(errors) > 0 {
		return &ExitForkResult{
			ExitedNetworks: exitedNetworks,
			Message:        fmt.Sprintf("Exited %d fork(s) with %d error(s)", len(exitedNetworks), len(errors)),
		}, fmt.Errorf("some forks failed to exit: %s", errors)
	}

	return &ExitForkResult{
		ExitedNetworks: exitedNetworks,
		Message:        fmt.Sprintf("All %d fork(s) exited", len(exitedNetworks)),
	}, nil
}

// cleanupFork handles the cleanup for a single fork: stop anvil, restore files, clean up dirs
func (uc *ExitFork) cleanupFork(ctx context.Context, entry *domain.ForkEntry) error {
	// Stop anvil process - handle already-dead processes gracefully
	instance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", entry.Network),
		Port:    portFromURL(entry.ForkURL),
		ChainID: fmt.Sprintf("%d", entry.ChainID),
		PidFile: entry.PidFile,
		LogFile: entry.LogFile,
	}

	// Stop is safe to call even if process is already dead
	if err := uc.anvilManager.Stop(ctx, instance); err != nil {
		// Log but don't fail - process may already be dead
		fmt.Printf("Warning: failed to stop anvil for '%s': %v\n", entry.Network, err)
	}

	// Restore registry files from initial backup (snapshot 0)
	if err := uc.forkFiles.RestoreFiles(ctx, entry.Network, 0); err != nil {
		return fmt.Errorf("failed to restore registry files: %w", err)
	}

	// Clean up fork directory
	if err := uc.forkFiles.CleanupForkDir(ctx, entry.Network); err != nil {
		return fmt.Errorf("failed to clean up fork directory: %w", err)
	}

	return nil
}

// saveOrDeleteState saves or deletes the fork state file depending on remaining forks
func (uc *ExitFork) saveOrDeleteState(ctx context.Context, state *domain.ForkState) error {
	if len(state.Forks) == 0 {
		if err := uc.forkState.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete fork state file: %w", err)
		}
	} else {
		if err := uc.forkState.Save(ctx, state); err != nil {
			return fmt.Errorf("failed to save fork state: %w", err)
		}
	}
	return nil
}

// portFromURL extracts the port from a URL like "http://127.0.0.1:12345"
func portFromURL(url string) string {
	// Find the last colon
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == ':' {
			return url[i+1:]
		}
	}
	return ""
}

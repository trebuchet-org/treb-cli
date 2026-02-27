package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// ForkHistory handles showing the snapshot history for a fork
type ForkHistory struct {
	cfg       *config.RuntimeConfig
	forkState ForkStateStore
}

// NewForkHistory creates a new ForkHistory use case
func NewForkHistory(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
) *ForkHistory {
	return &ForkHistory{
		cfg:       cfg,
		forkState: forkState,
	}
}

// ForkHistoryParams contains parameters for the fork history command
type ForkHistoryParams struct {
	Network string // empty means use current configured network
}

// ForkHistoryEntry represents a single entry in the fork history
type ForkHistoryEntry struct {
	Index      int
	Command    string
	Timestamp  string
	IsCurrent  bool // true for the top of the stack (most recent)
	IsInitial  bool // true for index 0 (fork enter point)
}

// ForkHistoryResult contains the result of the fork history command
type ForkHistoryResult struct {
	Network string
	Entries []ForkHistoryEntry
}

// Execute returns the snapshot history for a fork
func (uc *ForkHistory) Execute(ctx context.Context, params ForkHistoryParams) (*ForkHistoryResult, error) {
	network := params.Network
	if network == "" {
		if uc.cfg.Network != nil {
			network = uc.cfg.Network.Name
		}
		if network == "" {
			return nil, fmt.Errorf("no network specified and no current network configured")
		}
	}

	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load fork state: %w", err)
	}

	fork := state.GetActiveFork(network)
	if fork == nil {
		return nil, fmt.Errorf("no active fork for network '%s'", network)
	}

	entries := buildHistoryEntries(fork)

	return &ForkHistoryResult{
		Network: network,
		Entries: entries,
	}, nil
}

// buildHistoryEntries converts snapshot stack entries to history entries
func buildHistoryEntries(fork *domain.ForkEntry) []ForkHistoryEntry {
	entries := make([]ForkHistoryEntry, len(fork.Snapshots))
	topIndex := len(fork.Snapshots) - 1

	for i, snap := range fork.Snapshots {
		entries[i] = ForkHistoryEntry{
			Index:     snap.Index,
			Command:   snap.Command,
			Timestamp: snap.Timestamp.Format("2006-01-02 15:04:05"),
			IsCurrent: i == topIndex,
			IsInitial: snap.Index == 0,
		}
	}

	return entries
}

package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ForkStateStoreAdapter implements ForkStateStore using the file system
type ForkStateStoreAdapter struct {
	statePath string
}

// NewForkStateStoreAdapter creates a new ForkStateStoreAdapter
func NewForkStateStoreAdapter(cfg *config.RuntimeConfig) *ForkStateStoreAdapter {
	return &ForkStateStoreAdapter{
		statePath: filepath.Join(cfg.DataDir, "priv", "fork-state.json"),
	}
}

// Load reads the fork state from disk. Returns an empty state if the file does not exist.
func (s *ForkStateStoreAdapter) Load(_ context.Context) (*domain.ForkState, error) {
	data, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewForkState(), nil
		}
		return nil, fmt.Errorf("failed to read fork state file: %w", err)
	}

	var state domain.ForkState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse fork state file: %w", err)
	}

	if state.Forks == nil {
		state.Forks = make(map[string]*domain.ForkEntry)
	}

	return &state, nil
}

// Save writes the fork state to disk, creating the directory if needed.
func (s *ForkStateStoreAdapter) Save(_ context.Context, state *domain.ForkState) error {
	dir := filepath.Dir(s.statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create fork state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal fork state: %w", err)
	}

	if err := os.WriteFile(s.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write fork state file: %w", err)
	}

	return nil
}

// Delete removes the fork state file from disk.
func (s *ForkStateStoreAdapter) Delete(_ context.Context) error {
	err := os.Remove(s.statePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete fork state file: %w", err)
	}
	return nil
}

// Ensure ForkStateStoreAdapter implements ForkStateStore
var _ usecase.ForkStateStore = (*ForkStateStoreAdapter)(nil)

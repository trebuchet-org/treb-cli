package fs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func newTestForkStateStore(t *testing.T) *ForkStateStoreAdapter {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.RuntimeConfig{
		DataDir: tmpDir,
	}
	return NewForkStateStoreAdapter(cfg)
}

func TestForkStateStore_LoadEmpty(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()

	state, err := store.Load(ctx)
	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.NotNil(t, state.Forks)
	assert.Empty(t, state.Forks)
}

func TestForkStateStore_SaveAndLoad(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	state := &domain.ForkState{
		Forks: map[string]*domain.ForkEntry{
			"sepolia": {
				Network:     "sepolia",
				ChainID:     11155111,
				EnvVarName:  "SEPOLIA_RPC_URL",
				OriginalRPC: "https://rpc.sepolia.org",
				ForkURL:     "http://127.0.0.1:54321",
				AnvilPID:    12345,
				PidFile:     "/tmp/treb-fork-sepolia.pid",
				LogFile:     "/tmp/treb-fork-sepolia.log",
				EnteredAt:   now,
				Snapshots: []domain.SnapshotEntry{
					{
						Index:      0,
						SnapshotID: "0x1",
						Command:    "initial",
						Timestamp:  now,
					},
				},
			},
		},
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	// Verify file was created in .treb/priv/
	assert.FileExists(t, store.statePath)

	loaded, err := store.Load(ctx)
	require.NoError(t, err)

	assert.Len(t, loaded.Forks, 1)
	entry := loaded.Forks["sepolia"]
	require.NotNil(t, entry)
	assert.Equal(t, "sepolia", entry.Network)
	assert.Equal(t, uint64(11155111), entry.ChainID)
	assert.Equal(t, "SEPOLIA_RPC_URL", entry.EnvVarName)
	assert.Equal(t, "https://rpc.sepolia.org", entry.OriginalRPC)
	assert.Equal(t, "http://127.0.0.1:54321", entry.ForkURL)
	assert.Equal(t, 12345, entry.AnvilPID)
	assert.Equal(t, "/tmp/treb-fork-sepolia.pid", entry.PidFile)
	assert.Equal(t, "/tmp/treb-fork-sepolia.log", entry.LogFile)
	assert.True(t, entry.EnteredAt.Equal(now))
	require.Len(t, entry.Snapshots, 1)
	assert.Equal(t, "0x1", entry.Snapshots[0].SnapshotID)
}

func TestForkStateStore_SaveCreatesDirectory(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()

	state := domain.NewForkState()
	err := store.Save(ctx, state)
	require.NoError(t, err)

	dir := filepath.Dir(store.statePath)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestForkStateStore_Delete(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()

	// Save a state first
	state := domain.NewForkState()
	state.Forks["test"] = &domain.ForkEntry{Network: "test"}
	err := store.Save(ctx, state)
	require.NoError(t, err)
	assert.FileExists(t, store.statePath)

	// Delete it
	err = store.Delete(ctx)
	require.NoError(t, err)

	// File should be gone
	_, err = os.Stat(store.statePath)
	assert.True(t, os.IsNotExist(err))

	// Load should return empty state
	loaded, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Empty(t, loaded.Forks)
}

func TestForkStateStore_DeleteNonExistent(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()

	// Deleting a non-existent file should not error
	err := store.Delete(ctx)
	require.NoError(t, err)
}

func TestForkStateStore_MultipleForks(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	state := &domain.ForkState{
		Forks: map[string]*domain.ForkEntry{
			"sepolia": {
				Network:   "sepolia",
				ChainID:   11155111,
				ForkURL:   "http://127.0.0.1:54321",
				EnteredAt: now,
				Snapshots: []domain.SnapshotEntry{},
			},
			"mainnet": {
				Network:   "mainnet",
				ChainID:   1,
				ForkURL:   "http://127.0.0.1:54322",
				EnteredAt: now,
				Snapshots: []domain.SnapshotEntry{},
			},
		},
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	loaded, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, loaded.Forks, 2)
	assert.Contains(t, loaded.Forks, "sepolia")
	assert.Contains(t, loaded.Forks, "mainnet")
}

func TestForkStateStore_FileFormat(t *testing.T) {
	store := newTestForkStateStore(t)
	ctx := context.Background()

	state := domain.NewForkState()
	state.Forks["test"] = &domain.ForkEntry{
		Network: "test",
		ChainID: 31337,
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	// Read file and verify it's pretty-printed JSON
	data, err := os.ReadFile(store.statePath)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Verify pretty-printed (contains newlines and indentation)
	assert.Contains(t, string(data), "\n")
	assert.Contains(t, string(data), "  ")
}

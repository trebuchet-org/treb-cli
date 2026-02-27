package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForkState_SerializationRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	state := &ForkState{
		Forks: map[string]*ForkEntry{
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
				Snapshots: []SnapshotEntry{
					{
						Index:      0,
						SnapshotID: "0x1",
						Command:    "initial",
						Timestamp:  now,
					},
					{
						Index:      1,
						SnapshotID: "0x2",
						Command:    "script/deploy/DeployCounter.s.sol",
						Timestamp:  now.Add(time.Minute),
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(state, "", "  ")
	require.NoError(t, err)

	var restored ForkState
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Len(t, restored.Forks, 1)

	entry := restored.Forks["sepolia"]
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

	require.Len(t, entry.Snapshots, 2)
	assert.Equal(t, 0, entry.Snapshots[0].Index)
	assert.Equal(t, "0x1", entry.Snapshots[0].SnapshotID)
	assert.Equal(t, "initial", entry.Snapshots[0].Command)
	assert.Equal(t, 1, entry.Snapshots[1].Index)
	assert.Equal(t, "0x2", entry.Snapshots[1].SnapshotID)
	assert.Equal(t, "script/deploy/DeployCounter.s.sol", entry.Snapshots[1].Command)
}

func TestForkState_EmptyRoundTrip(t *testing.T) {
	state := NewForkState()

	data, err := json.Marshal(state)
	require.NoError(t, err)

	var restored ForkState
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.NotNil(t, restored.Forks)
	assert.Empty(t, restored.Forks)
}

func TestForkState_IsForkActive(t *testing.T) {
	tests := []struct {
		name    string
		state   *ForkState
		network string
		want    bool
	}{
		{
			name:    "nil state",
			state:   nil,
			network: "sepolia",
			want:    false,
		},
		{
			name:    "nil forks map",
			state:   &ForkState{Forks: nil},
			network: "sepolia",
			want:    false,
		},
		{
			name:    "empty forks",
			state:   NewForkState(),
			network: "sepolia",
			want:    false,
		},
		{
			name: "fork exists",
			state: &ForkState{
				Forks: map[string]*ForkEntry{
					"sepolia": {Network: "sepolia"},
				},
			},
			network: "sepolia",
			want:    true,
		},
		{
			name: "different network",
			state: &ForkState{
				Forks: map[string]*ForkEntry{
					"mainnet": {Network: "mainnet"},
				},
			},
			network: "sepolia",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.IsForkActive(tt.network)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestForkState_GetActiveFork(t *testing.T) {
	entry := &ForkEntry{
		Network:     "sepolia",
		ChainID:     11155111,
		EnvVarName:  "SEPOLIA_RPC_URL",
		OriginalRPC: "https://rpc.sepolia.org",
		ForkURL:     "http://127.0.0.1:54321",
		AnvilPID:    12345,
	}

	state := &ForkState{
		Forks: map[string]*ForkEntry{
			"sepolia": entry,
		},
	}

	t.Run("returns entry for active fork", func(t *testing.T) {
		got := state.GetActiveFork("sepolia")
		require.NotNil(t, got)
		assert.Equal(t, entry, got)
	})

	t.Run("returns nil for inactive fork", func(t *testing.T) {
		got := state.GetActiveFork("mainnet")
		assert.Nil(t, got)
	})

	t.Run("returns nil for nil state", func(t *testing.T) {
		var nilState *ForkState
		got := nilState.GetActiveFork("sepolia")
		assert.Nil(t, got)
	})

	t.Run("returns nil for nil forks map", func(t *testing.T) {
		got := (&ForkState{Forks: nil}).GetActiveFork("sepolia")
		assert.Nil(t, got)
	})
}

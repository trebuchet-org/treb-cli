package domain

import (
	"fmt"
	"time"
)

// ForkState represents the state of all active forks
type ForkState struct {
	Forks map[string]*ForkEntry `json:"forks"` // keyed by network name
}

// ForkEntry represents an active fork for a single network
type ForkEntry struct {
	Network     string          `json:"network"`
	ChainID     uint64          `json:"chainId"`
	EnvVarName  string          `json:"envVarName"`
	OriginalRPC string          `json:"originalRpc"`
	ForkURL     string          `json:"forkUrl"`
	AnvilPID    int             `json:"anvilPid"`
	PidFile     string          `json:"pidFile"`
	LogFile     string          `json:"logFile"`
	EnteredAt   time.Time       `json:"enteredAt"`
	Snapshots   []SnapshotEntry `json:"snapshots"`
	External    bool            `json:"external,omitempty"`
}

// SnapshotEntry represents an EVM snapshot point in the fork
type SnapshotEntry struct {
	Index      int       `json:"index"`
	SnapshotID string    `json:"snapshotId"`
	Command    string    `json:"command"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewForkState creates a new empty ForkState
func NewForkState() *ForkState {
	return &ForkState{
		Forks: make(map[string]*ForkEntry),
	}
}

// IsForkActive returns true if a fork is active for the given network
func (s *ForkState) IsForkActive(network string) bool {
	if s == nil || s.Forks == nil {
		return false
	}
	_, ok := s.Forks[network]
	return ok
}

// GetActiveFork returns the fork entry for the given network, or nil if not active
func (s *ForkState) GetActiveFork(network string) *ForkEntry {
	if s == nil || s.Forks == nil {
		return nil
	}
	return s.Forks[network]
}

// ActiveNetworks returns a list of all active fork network names
func (s *ForkState) ActiveNetworks() []string {
	if s == nil || s.Forks == nil {
		return nil
	}
	networks := make([]string, 0, len(s.Forks))
	for name := range s.Forks {
		networks = append(networks, name)
	}
	return networks
}

// AnvilInstance returns an AnvilInstance for this fork entry.
// For external forks, RPCURL is set to the full fork URL so RPC calls
// go to the correct endpoint instead of constructing http://localhost:<port>.
func (e *ForkEntry) AnvilInstance() *AnvilInstance {
	instance := &AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", e.Network),
		ChainID: fmt.Sprintf("%d", e.ChainID),
		PidFile: e.PidFile,
		LogFile: e.LogFile,
	}

	if e.External {
		instance.RPCURL = e.ForkURL
	} else {
		instance.Port = portFromURL(e.ForkURL)
	}

	return instance
}

// portFromURL extracts the port from a URL like "http://127.0.0.1:12345"
func portFromURL(rawURL string) string {
	// Find the last colon
	for i := len(rawURL) - 1; i >= 0; i-- {
		if rawURL[i] == ':' {
			port := rawURL[i+1:]
			// Trim trailing slash
			if len(port) > 0 && port[len(port)-1] == '/' {
				port = port[:len(port)-1]
			}
			return port
		}
	}
	return ""
}

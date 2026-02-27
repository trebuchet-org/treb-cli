package domain

import "time"

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

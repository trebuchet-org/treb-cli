package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// ForkStatus handles showing the status of active forks
type ForkStatus struct {
	cfg          *config.RuntimeConfig
	forkState    ForkStateStore
	anvilManager AnvilManager
}

// NewForkStatus creates a new ForkStatus use case
func NewForkStatus(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
	anvilManager AnvilManager,
) *ForkStatus {
	return &ForkStatus{
		cfg:          cfg,
		forkState:    forkState,
		anvilManager: anvilManager,
	}
}

// ForkStatusEntry contains status info for a single active fork
type ForkStatusEntry struct {
	Network         string
	ChainID         uint64
	ForkURL         string
	AnvilPID        int
	Uptime          time.Duration
	SnapshotCount   int
	Healthy         bool
	HealthDetail    string // "healthy" or "dead"
	ForkDeployments int    // number of deployments added during fork
	IsCurrent       bool   // true if this is the currently configured network
	LogFile         string // path to anvil log file
}

// ForkStatusResult contains the result of the fork status command
type ForkStatusResult struct {
	Entries  []ForkStatusEntry
	HasForks bool
}

// Execute returns the status of all active forks
func (uc *ForkStatus) Execute(ctx context.Context) (*ForkStatusResult, error) {
	state, err := uc.forkState.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load fork state: %w", err)
	}

	if len(state.Forks) == 0 {
		return &ForkStatusResult{HasForks: false}, nil
	}

	currentNetwork := ""
	if uc.cfg.Network != nil {
		currentNetwork = uc.cfg.Network.Name
	}

	var entries []ForkStatusEntry
	for _, entry := range state.Forks {
		statusEntry := uc.buildStatusEntry(ctx, entry, currentNetwork)
		entries = append(entries, statusEntry)
	}

	return &ForkStatusResult{
		Entries:  entries,
		HasForks: true,
	}, nil
}

// buildStatusEntry builds a status entry for a single fork
func (uc *ForkStatus) buildStatusEntry(ctx context.Context, entry *domain.ForkEntry, currentNetwork string) ForkStatusEntry {
	se := ForkStatusEntry{
		Network:       entry.Network,
		ChainID:       entry.ChainID,
		ForkURL:       entry.ForkURL,
		AnvilPID:      entry.AnvilPID,
		Uptime:        time.Since(entry.EnteredAt),
		SnapshotCount: len(entry.Snapshots),
		IsCurrent:     entry.Network == currentNetwork,
		LogFile:       entry.LogFile,
	}

	// Health check via anvil manager
	instance := &domain.AnvilInstance{
		Name:    fmt.Sprintf("fork-%s", entry.Network),
		Port:    portFromURL(entry.ForkURL),
		ChainID: fmt.Sprintf("%d", entry.ChainID),
		PidFile: entry.PidFile,
		LogFile: entry.LogFile,
	}

	status, err := uc.anvilManager.GetStatus(ctx, instance)
	if err != nil || !status.Running || !status.RPCHealthy {
		se.Healthy = false
		se.HealthDetail = "dead"
	} else {
		se.Healthy = true
		se.HealthDetail = "healthy"
	}

	// Count fork-added deployments
	se.ForkDeployments = uc.countForkDeployments(entry.Network)

	return se
}

// countForkDeployments counts deployments added during fork mode by comparing
// current deployments.json against the initial backup at snapshot 0.
func (uc *ForkStatus) countForkDeployments(network string) int {
	// Load current deployments
	currentPath := filepath.Join(uc.cfg.DataDir, "deployments.json")
	currentIDs := loadDeploymentIDs(currentPath)

	// Load initial backup deployments (snapshot 0)
	backupPath := filepath.Join(uc.cfg.DataDir, "priv", "fork", network, "snapshots", "0", "deployments.json")
	backupIDs := loadDeploymentIDs(backupPath)

	// Count IDs in current that are not in backup
	count := 0
	for id := range currentIDs {
		if !backupIDs[id] {
			count++
		}
	}
	return count
}

// loadDeploymentIDs reads a deployments.json file and returns the set of deployment IDs.
// Returns an empty map if the file doesn't exist or can't be parsed.
func loadDeploymentIDs(path string) map[string]bool {
	data, err := os.ReadFile(path) //nolint:gosec // internally constructed path
	if err != nil {
		return map[string]bool{}
	}

	var deployments map[string]json.RawMessage
	if err := json.Unmarshal(data, &deployments); err != nil {
		return map[string]bool{}
	}

	ids := make(map[string]bool, len(deployments))
	for id := range deployments {
		ids[id] = true
	}
	return ids
}

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// DiffFork handles showing the difference between current state and initial fork state
type DiffFork struct {
	cfg       *config.RuntimeConfig
	forkState ForkStateStore
}

// NewDiffFork creates a new DiffFork use case
func NewDiffFork(
	cfg *config.RuntimeConfig,
	forkState ForkStateStore,
) *DiffFork {
	return &DiffFork{
		cfg:       cfg,
		forkState: forkState,
	}
}

// DiffForkParams contains parameters for the fork diff command
type DiffForkParams struct {
	Network string // empty means use current configured network
}

// ForkDiffEntry represents a single difference item (deployment or transaction)
type ForkDiffEntry struct {
	ID           string `json:"id"`
	ContractName string `json:"contractName,omitempty"`
	Address      string `json:"address,omitempty"`
	Type         string `json:"type"` // "SINGLETON", "PROXY", "LIBRARY"
	ChangeType   string `json:"changeType"` // "added" or "modified"
}

// ForkDiffResult contains the result of the fork diff command
type ForkDiffResult struct {
	Network              string          `json:"network"`
	NewDeployments       []ForkDiffEntry `json:"newDeployments"`
	ModifiedDeployments  []ForkDiffEntry `json:"modifiedDeployments"`
	NewTransactionCount  int             `json:"newTransactionCount"`
	HasChanges           bool            `json:"hasChanges"`
}

// Execute returns the diff between current and initial fork state
func (uc *DiffFork) Execute(ctx context.Context, params DiffForkParams) (*ForkDiffResult, error) {
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

	backupDir := filepath.Join(uc.cfg.DataDir, "priv", "fork", network, "snapshots", "0")

	// Compare deployments
	newDeps, modifiedDeps := uc.diffDeployments(backupDir)

	// Compare transactions
	newTxCount := uc.diffTransactions(backupDir)

	hasChanges := len(newDeps) > 0 || len(modifiedDeps) > 0 || newTxCount > 0

	return &ForkDiffResult{
		Network:             network,
		NewDeployments:      newDeps,
		ModifiedDeployments: modifiedDeps,
		NewTransactionCount: newTxCount,
		HasChanges:          hasChanges,
	}, nil
}

// deploymentJSON represents the minimal fields we need from a deployment JSON entry
type deploymentJSON struct {
	ID           string `json:"id"`
	ContractName string `json:"contractName"`
	Address      string `json:"address"`
	Type         string `json:"type"`
}

// diffDeployments compares current deployments.json against backup and returns new and modified entries
func (uc *DiffFork) diffDeployments(backupDir string) (newDeps []ForkDiffEntry, modifiedDeps []ForkDiffEntry) {
	currentPath := filepath.Join(uc.cfg.DataDir, "deployments.json")
	backupPath := filepath.Join(backupDir, "deployments.json")

	currentMap := loadRawDeployments(currentPath)
	backupMap := loadRawDeployments(backupPath)

	for id, currentRaw := range currentMap {
		backupRaw, existed := backupMap[id]

		var dep deploymentJSON
		if err := json.Unmarshal(currentRaw, &dep); err != nil {
			continue
		}

		entry := ForkDiffEntry{
			ID:           id,
			ContractName: dep.ContractName,
			Address:      dep.Address,
			Type:         dep.Type,
		}

		if !existed {
			entry.ChangeType = "added"
			newDeps = append(newDeps, entry)
		} else {
			// Compare raw JSON - if different, it's modified
			if string(currentRaw) != string(backupRaw) {
				entry.ChangeType = "modified"
				modifiedDeps = append(modifiedDeps, entry)
			}
		}
	}

	return newDeps, modifiedDeps
}

// diffTransactions compares current transactions.json against backup and returns the count of new transactions
func (uc *DiffFork) diffTransactions(backupDir string) int {
	currentPath := filepath.Join(uc.cfg.DataDir, "transactions.json")
	backupPath := filepath.Join(backupDir, "transactions.json")

	currentIDs := loadDeploymentIDs(currentPath)
	backupIDs := loadDeploymentIDs(backupPath)

	count := 0
	for id := range currentIDs {
		if !backupIDs[id] {
			count++
		}
	}
	return count
}

// loadRawDeployments reads a JSON file and returns a map of ID to raw JSON.
// Returns an empty map if the file doesn't exist or can't be parsed.
func loadRawDeployments(path string) map[string]json.RawMessage {
	data, err := os.ReadFile(path) //nolint:gosec // internally constructed path
	if err != nil {
		return map[string]json.RawMessage{}
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]json.RawMessage{}
	}

	return result
}

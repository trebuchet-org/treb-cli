package helpers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/trebuchet-org/treb-cli/pkg/anvil"
)

// AnvilNode wraps anvil.AnvilInstance with additional test-specific data
type AnvilNode struct {
	*anvil.AnvilInstance
	URL string
}

// AnvilSnapshot represents a snapshot state
type AnvilSnapshot struct {
	NodeSnapshots map[string]string // node name -> snapshot ID
}

// AnvilManager manages multiple anvil nodes
type AnvilManager struct {
	nodes map[string]*AnvilNode
}

// NewAnvilManager creates a new anvil manager by parsing foundry.toml
func NewAnvilManager() (*AnvilManager, error) {

	am := &AnvilManager{
		nodes: make(map[string]*AnvilNode),
	}

	// Parse foundry.toml to find anvil nodes
	foundryPath := filepath.Join(fixtureDir, "foundry.toml")
	data, err := os.ReadFile(foundryPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read foundry.toml: %v", err)
	}

	var config struct {
		RPCEndpoints map[string]string `toml:"rpc_endpoints"`
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to parse foundry.toml: %v", err)
	}

	// Find all anvil-* nodes and parse chain ID
	for name, endpoint := range config.RPCEndpoints {
		if strings.HasPrefix(name, "anvil-") {
			// Parse chain ID from name (anvil-31337 -> 31337)
			parts := strings.Split(name, "-")
			if len(parts) != 2 {
				fmt.Printf("Skipping invalid anvil node name: %s\n", name)
				continue
			}
			chainID := parts[1]

			u, err := url.Parse(endpoint)
			if err != nil {
				fmt.Printf("Skipping invalid URL for %s: %v", name, err)
				continue
			}

			port := u.Port()
			if port == "" {
				if u.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
			}

			am.nodes[name] = &AnvilNode{
				AnvilInstance: anvil.NewAnvilInstance(name, port).WithChainID(chainID),
				URL:           endpoint,
			}
		}
	}

	return am, nil
}

// StartAll starts all anvil nodes
func (am *AnvilManager) StartAll() error {
	for _, node := range am.nodes {
		if err := anvil.StartAnvilInstance(node.Name, node.Port, node.ChainID); err != nil {
			return fmt.Errorf("failed to start %s: %w", node.Name, err)
		}
	}
	// CreateX is deployed automatically by anvil.StartAnvilInstance
	return nil
}

// StopAll stops all anvil nodes
func (am *AnvilManager) StopAll() {
	for _, node := range am.nodes {
		if err := anvil.StopAnvilInstance(node.Name, node.Port); err != nil {
			fmt.Printf("Failed to stop %s: %v", node.Name, err)
		}
	}
}

// Snapshot creates a snapshot of all nodes
func (am *AnvilManager) Snapshot() (*AnvilSnapshot, error) {
	snapshot := &AnvilSnapshot{
		NodeSnapshots: make(map[string]string),
	}

	for name, node := range am.nodes {
		id, err := am.takeSnapshot(node)
		if err != nil {
			return nil, fmt.Errorf("failed to snapshot %s: %w", name, err)
		}
		snapshot.NodeSnapshots[name] = id
	}

	return snapshot, nil
}

// Revert reverts all nodes to a snapshot
func (am *AnvilManager) Revert(snapshot *AnvilSnapshot) error {
	for name, snapshotID := range snapshot.NodeSnapshots {
		node, ok := am.nodes[name]
		if !ok {
			return fmt.Errorf("node %s not found", name)
		}

		if err := am.revertSnapshot(node, snapshotID); err != nil {
			return fmt.Errorf("failed to revert %s to snapshot %s: %w", name, snapshotID, err)
		}
	}

	// After reverting, we need to take new snapshots for the next test
	newSnapshot, err := am.Snapshot()
	if err != nil {
		return fmt.Errorf("failed to create new snapshots after revert: %w", err)
	}

	// Update the snapshot with new IDs
	for name, id := range newSnapshot.NodeSnapshots {
		snapshot.NodeSnapshots[name] = id
	}

	return nil
}

func (am *AnvilManager) takeSnapshot(node *AnvilNode) (string, error) {
	cmd := exec.Command("cast", "rpc", "evm_snapshot", "--rpc-url", node.URL)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("snapshot failed: %w", err)
	}

	// Parse the hex snapshot ID from output
	var result string
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse snapshot ID: %w", err)
	}

	return result, nil
}

func (am *AnvilManager) revertSnapshot(node *AnvilNode, snapshotID string) error {
	cmd := exec.Command("cast", "rpc", "evm_revert", snapshotID, "--rpc-url", node.URL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("revert failed: %w\nOutput: %s", err, output)
	}
	return nil
}

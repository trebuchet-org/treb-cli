package integration_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/trebuchet-org/treb-cli/cli/pkg/dev"
)

// AnvilNode wraps dev.AnvilInstance with additional test-specific data
type AnvilNode struct {
	*dev.AnvilInstance
	URL string
}

// AnvilSnapshot represents a snapshot state
type AnvilSnapshot struct {
	NodeSnapshots map[string]string // node name -> snapshot ID
}

// AnvilManager manages multiple anvil nodes
type AnvilManager struct {
	t     *testing.T
	nodes map[string]*AnvilNode
}

// NewAnvilManager creates a new anvil manager by parsing foundry.toml
func NewAnvilManager(t *testing.T) *AnvilManager {
	t.Helper()
	
	am := &AnvilManager{
		t:     t,
		nodes: make(map[string]*AnvilNode),
	}
	
	// Parse foundry.toml to find anvil nodes
	foundryPath := filepath.Join(fixtureDir, "foundry.toml")
	data, err := os.ReadFile(foundryPath)
	if err != nil {
		t.Fatalf("Failed to read foundry.toml: %v", err)
	}
	
	var config struct {
		RPCEndpoints map[string]string `toml:"rpc_endpoints"`
	}
	
	if err := toml.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse foundry.toml: %v", err)
	}
	
	// Find all anvil-* nodes and parse chain ID
	for name, endpoint := range config.RPCEndpoints {
		if strings.HasPrefix(name, "anvil-") {
			// Parse chain ID from name (anvil-31337 -> 31337)
			parts := strings.Split(name, "-")
			if len(parts) != 2 {
				t.Logf("Skipping invalid anvil node name: %s", name)
				continue
			}
			chainID := parts[1]
			
			u, err := url.Parse(endpoint)
			if err != nil {
				t.Logf("Skipping invalid URL for %s: %v", name, err)
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
				AnvilInstance: dev.NewAnvilInstance(name, port).WithChainID(chainID),
				URL:           endpoint,
			}
			
			t.Logf("Found anvil node: %s on port %s with chain ID %s", name, port, chainID)
		}
	}
	
	return am
}

// StartAll starts all anvil nodes
func (am *AnvilManager) StartAll() error {
	for _, node := range am.nodes {
		if err := dev.StartAnvilInstance(node.Name, node.Port, node.ChainID); err != nil {
			return fmt.Errorf("failed to start %s: %w", node.Name, err)
		}
	}
	// CreateX is deployed automatically by dev.StartAnvilInstance
	return nil
}

// StopAll stops all anvil nodes
func (am *AnvilManager) StopAll() {
	for _, node := range am.nodes {
		am.t.Logf("Stopping %s...", node.Name)
		if err := dev.StopAnvilInstance(node.Name, node.Port); err != nil {
			am.t.Logf("Failed to stop %s: %v", node.Name, err)
		} else {
			am.t.Logf("✅ %s stopped", node.Name)
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
		am.t.Logf("Created snapshot %s for %s", id, name)
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
		
		am.t.Logf("Reverted %s to snapshot %s", name, snapshotID)
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


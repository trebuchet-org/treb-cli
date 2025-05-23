package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bogdan/fdeploy/cli/pkg/types"
)

type Manager struct {
	registryPath string
	registry     *Registry
}

type Registry struct {
	Project  ProjectMetadata           `json:"project"`
	Networks map[string]*NetworkEntry `json:"networks"`
}

type ProjectMetadata struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Commit    string    `json:"commit"`
	Timestamp time.Time `json:"timestamp"`
}

type NetworkEntry struct {
	Name        string                        `json:"name"`
	Deployments map[string]*types.DeploymentEntry `json:"deployments"`
}

func NewManager(registryPath string) (*Manager, error) {
	manager := &Manager{
		registryPath: registryPath,
	}
	
	if err := manager.load(); err != nil {
		return nil, err
	}
	
	return manager, nil
}

func (m *Manager) load() error {
	if _, err := os.Stat(m.registryPath); os.IsNotExist(err) {
		// Create empty registry
		m.registry = &Registry{
			Project: ProjectMetadata{
				Name:      "fdeploy-project",
				Version:   "0.1.0",
				Timestamp: time.Now(),
			},
			Networks: make(map[string]*NetworkEntry),
		}
		return nil
	}

	data, err := os.ReadFile(m.registryPath)
	if err != nil {
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	m.registry = &Registry{}
	if err := json.Unmarshal(data, m.registry); err != nil {
		return fmt.Errorf("failed to parse registry file: %w", err)
	}

	// Initialize networks map if nil
	if m.registry.Networks == nil {
		m.registry.Networks = make(map[string]*NetworkEntry)
	}

	return nil
}

func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(m.registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

func (m *Manager) RecordDeployment(contract, env string, result *types.DeploymentResult) error {
	chainID := "11155111" // TODO: Get from result or config
	
	// Ensure network exists
	if m.registry.Networks[chainID] == nil {
		m.registry.Networks[chainID] = &NetworkEntry{
			Name:        "sepolia", // TODO: Map chain ID to name
			Deployments: make(map[string]*types.DeploymentEntry),
		}
	}

	entry := &types.DeploymentEntry{
		Address:      result.Address,
		Type:         "implementation",
		Salt:         result.Salt,
		InitCodeHash: result.InitCodeHash,
		
		Verification: types.Verification{
			Status: "pending",
		},
		
		Deployment: types.DeploymentInfo{
			TxHash:        &result.TxHash,
			BlockNumber:   result.BlockNumber,
			BroadcastFile: result.BroadcastFile,
			Timestamp:     time.Now(),
		},
		
		Metadata: types.ContractMetadata{
			ContractVersion: m.getContractVersion(),
			SourceCommit:    m.getGitCommit(),
			Compiler:        "0.8.22", // TODO: Get from foundry.toml
			SourceHash:      m.calculateSourceHash(contract),
		},
	}

	key := fmt.Sprintf("%s_%s", contract, env)
	m.registry.Networks[chainID].Deployments[key] = entry

	return m.Save()
}

func (m *Manager) GetDeployment(contract, env string) *types.DeploymentEntry {
	chainID := "11155111" // TODO: Get from config
	
	if network := m.registry.Networks[chainID]; network != nil {
		key := fmt.Sprintf("%s_%s", contract, env)
		return network.Deployments[key]
	}
	
	return nil
}

func (m *Manager) GetPendingVerifications(chainID uint64) map[string]*types.DeploymentEntry {
	chainIDStr := fmt.Sprintf("%d", chainID)
	pending := make(map[string]*types.DeploymentEntry)
	
	if network := m.registry.Networks[chainIDStr]; network != nil {
		for key, deployment := range network.Deployments {
			if deployment.Verification.Status == "pending" {
				pending[key] = deployment
			}
		}
	}
	
	return pending
}

func (m *Manager) UpdateDeployment(key string, deployment *types.DeploymentEntry) error {
	chainID := "11155111" // TODO: Get from config
	
	if network := m.registry.Networks[chainID]; network != nil {
		network.Deployments[key] = deployment
		return m.Save()
	}
	
	return fmt.Errorf("network not found")
}

func (m *Manager) getContractVersion() string {
	return m.registry.Project.Version
}

func (m *Manager) getGitCommit() string {
	// TODO: Get current git commit
	return ""
}

func (m *Manager) calculateSourceHash(contract string) string {
	// TODO: Calculate hash of contract source
	return ""
}
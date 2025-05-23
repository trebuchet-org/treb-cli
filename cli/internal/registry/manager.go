package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogdan/fdeploy/cli/pkg/types"
	"github.com/ethereum/go-ethereum/common"
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

	// Try to load with new format first
	m.registry = &Registry{}
	if err := json.Unmarshal(data, m.registry); err != nil {
		// If it fails, try to migrate from old format with byte arrays
		if err := m.migrateFromOldFormat(data); err != nil {
			return fmt.Errorf("failed to parse registry file: %w", err)
		}
		
		// Save the migrated registry
		if err := m.Save(); err != nil {
			return fmt.Errorf("failed to save migrated registry: %w", err)
		}
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

func (m *Manager) RecordDeployment(contract, env string, result *types.DeploymentResult, chainID uint64) error {
	chainIDStr := fmt.Sprintf("%d", chainID)
	
	// Ensure network exists
	if m.registry.Networks[chainIDStr] == nil {
		m.registry.Networks[chainIDStr] = &NetworkEntry{
			Name:        m.getNetworkName(chainID),
			Deployments: make(map[string]*types.DeploymentEntry),
		}
	}

	// Default environment to "default" if not provided
	if env == "" {
		env = "default"
	}
	
	entry := &types.DeploymentEntry{
		Address:      result.Address,
		ContractName: contract,
		Environment:  env,
		Type:         result.DeploymentType, // Now comes from structured output
		Salt:         hex.EncodeToString(result.Salt[:]),         // Convert to hex string
		InitCodeHash: hex.EncodeToString(result.InitCodeHash[:]), // Convert to hex string
		
		// Label for all deployments
		Label:        result.Label,
		
		// Proxy-specific fields
		TargetContract: result.TargetContract,
		ProxyLabel:     result.ProxyLabel, // DEPRECATED: kept for compatibility
		
		// Version tags
		Tags: result.Tags,
		
		Verification: types.Verification{
			Status: "pending",
		},
		
		Deployment: types.DeploymentInfo{
			TxHash:        &result.TxHash,
			BlockNumber:   result.BlockNumber,
			BroadcastFile: result.BroadcastFile,
			Timestamp:     time.Now(),
			Status:        "deployed",
		},
		
		Metadata: types.ContractMetadata{
			ContractVersion: m.getContractVersion(contract),
			SourceCommit:    m.getGitCommit(),
			Compiler:        m.getCompilerVersion(),
			SourceHash:      m.calculateSourceHash(contract),
			ContractPath:    m.getContractPath(contract),
		},
	}

	// Use address as key for uniqueness
	key := strings.ToLower(result.Address.Hex())
	m.registry.Networks[chainIDStr].Deployments[key] = entry

	return m.Save()
}

func (m *Manager) GetDeployment(contract, env string, chainID uint64) *types.DeploymentEntry {
	chainIDStr := fmt.Sprintf("%d", chainID)
	
	// Default environment to "default" if not provided
	if env == "" {
		env = "default"
	}
	
	if network := m.registry.Networks[chainIDStr]; network != nil {
		// Search through deployments to find matching contract and env
		for _, deployment := range network.Deployments {
			if deployment.ContractName == contract && deployment.Environment == env {
				return deployment
			}
		}
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

func (m *Manager) getContractVersion(contract string) string {
	// Try to extract version from the project first
	if m.registry.Project.Version != "" && m.registry.Project.Version != "0.1.0" {
		return m.registry.Project.Version
	}
	
	// Try to extract from foundry.toml
	if version := m.getVersionFromFoundryToml(); version != "" {
		return version
	}
	
	// Try to extract from package.json if it exists
	if version := m.getVersionFromPackageJson(); version != "" {
		return version
	}
	
	// Fallback to git tag
	if version := m.getVersionFromGitTag(); version != "" {
		return version
	}
	
	// Final fallback
	return "1.0.0"
}

func (m *Manager) getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (m *Manager) getCompilerVersion() string {
	// Try to extract from foundry.toml
	if version := m.getCompilerFromFoundryToml(); version != "" {
		return version
	}
	
	// Fallback to default
	return "0.8.22"
}

func (m *Manager) calculateSourceHash(contract string) string {
	// Find the contract source file
	contractPath := m.findContractSourceFile(contract)
	if contractPath == "" {
		return ""
	}
	
	// Read the file and calculate hash
	content, err := os.ReadFile(contractPath)
	if err != nil {
		return ""
	}
	
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func (m *Manager) getContractPath(contract string) string {
	// Find the contract source file
	contractPath := m.findContractSourceFile(contract)
	if contractPath == "" {
		return ""
	}
	
	// Format as ./src/Contract.sol:Contract
	return fmt.Sprintf("./%s:%s", contractPath, contract)
}

func (m *Manager) getVersionFromFoundryToml() string {
	content, err := os.ReadFile("foundry.toml")
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				return version
			}
		}
	}
	return ""
}

func (m *Manager) getVersionFromPackageJson() string {
	content, err := os.ReadFile("package.json")
	if err != nil {
		return ""
	}
	
	var pkg map[string]interface{}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return ""
	}
	
	if version, ok := pkg["version"].(string); ok {
		return version
	}
	return ""
}

func (m *Manager) getVersionFromGitTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (m *Manager) getCompilerFromFoundryToml() string {
	content, err := os.ReadFile("foundry.toml")
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "solc") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				return version
			}
		}
	}
	return ""
}

func (m *Manager) findContractSourceFile(contract string) string {
	// Search in src/ directory
	var contractPath string
	
	filepath.WalkDir("src", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		if !d.IsDir() && strings.HasSuffix(path, ".sol") {
			// Check if this file contains the contract
			if content, err := os.ReadFile(path); err == nil {
				if strings.Contains(string(content), fmt.Sprintf("contract %s", contract)) {
					contractPath = path
					return filepath.SkipAll // Found it, stop walking
				}
			}
		}
		return nil
	})
	
	return contractPath
}

func (m *Manager) getNetworkName(chainID uint64) string {
	// Map common chain IDs to names
	switch chainID {
	case 1:
		return "mainnet"
	case 11155111:
		return "sepolia"
	case 137:
		return "polygon"
	case 44787:
		return "alfajores"
	default:
		return fmt.Sprintf("chain-%d", chainID)
	}
}

// DeploymentInfo represents deployment information for listing
type DeploymentInfo struct {
	Address     common.Address         `json:"address"`
	NetworkName string                 `json:"network_name"`
	ChainID     string                 `json:"chain_id"`
	Entry       *types.DeploymentEntry `json:"entry"`
}

// NetworkSummary represents network information and statistics
type NetworkSummary struct {
	Name            string   `json:"name"`
	DeploymentCount int      `json:"deployment_count"`
	Contracts       []string `json:"contracts"`
}

// RegistryStatus represents overall registry status
type RegistryStatus struct {
	ProjectName         string                   `json:"project_name"`
	ProjectVersion      string                   `json:"project_version"`
	LastUpdated         string                   `json:"last_updated"`
	NetworkCount        int                      `json:"network_count"`
	TotalDeployments    int                      `json:"total_deployments"`
	VerifiedCount       int                      `json:"verified_count"`
	PendingVerification int                      `json:"pending_verification"`
	RecentDeployments   []RecentDeploymentInfo   `json:"recent_deployments"`
}

// RecentDeploymentInfo represents recent deployment information
type RecentDeploymentInfo struct {
	ContractEnv string `json:"contract_env"`  // Keep for backward compatibility
	Contract    string `json:"contract"`
	Environment string `json:"environment"`
	Address     string `json:"address"`
	Network     string `json:"network"`
	Timestamp   string `json:"timestamp"`
	Type        string `json:"type"`  // implementation/proxy
	ProxyLabel  string `json:"proxy_label,omitempty"`
}

// GetAllDeployments returns all deployments across networks
func (m *Manager) GetAllDeployments() []*DeploymentInfo {
	var deployments []*DeploymentInfo
	
	for chainID, network := range m.registry.Networks {
		for _, deployment := range network.Deployments {
			deployments = append(deployments, &DeploymentInfo{
				Address:     deployment.Address,
				NetworkName: network.Name,
				ChainID:     chainID,
				Entry:       deployment,
			})
		}
	}
	
	return deployments
}

// GetNetworkSummary returns network summary information
func (m *Manager) GetNetworkSummary() map[string]*NetworkSummary {
	networks := make(map[string]*NetworkSummary)
	
	for chainID, network := range m.registry.Networks {
		contracts := make(map[string]bool)
		
		for key := range network.Deployments {
			// Extract contract name from key (format: ContractName_env)
			parts := strings.Split(key, "_")
			if len(parts) > 0 {
				contracts[parts[0]] = true
			}
		}
		
		contractNames := make([]string, 0, len(contracts))
		for contract := range contracts {
			contractNames = append(contractNames, contract)
		}
		
		networks[chainID] = &NetworkSummary{
			Name:            network.Name,
			DeploymentCount: len(network.Deployments),
			Contracts:       contractNames,
		}
	}
	
	return networks
}

// GetStatus returns overall registry status
func (m *Manager) GetStatus() *RegistryStatus {
	status := &RegistryStatus{
		ProjectName:    m.registry.Project.Name,
		ProjectVersion: m.registry.Project.Version,
		LastUpdated:    m.registry.Project.Timestamp.Format("2006-01-02 15:04:05"),
		NetworkCount:   len(m.registry.Networks),
	}
	
	// Count deployments and verification status
	recentDeployments := make([]RecentDeploymentInfo, 0)
	
	for _, network := range m.registry.Networks {
		for _, deployment := range network.Deployments {
			status.TotalDeployments++
			
			// Count verification status
			switch deployment.Verification.Status {
			case "verified":
				status.VerifiedCount++
			case "pending":
				status.PendingVerification++
			}
			
			// Add to recent deployments (limit to 5 most recent)
			if len(recentDeployments) < 5 {
				recentDeployments = append(recentDeployments, RecentDeploymentInfo{
					ContractEnv: fmt.Sprintf("%s_%s", deployment.ContractName, deployment.Environment),  // Keep for backward compatibility
					Contract:    deployment.ContractName,
					Environment: deployment.Environment,
					Address:     deployment.Address.Hex(),
					Network:     network.Name,
					Timestamp:   deployment.Deployment.Timestamp.Format("2006-01-02 15:04"),
					Type:        deployment.Type,
					ProxyLabel:  deployment.Label,  // Use Label instead of ProxyLabel
				})
			}
		}
	}
	
	status.RecentDeployments = recentDeployments
	return status
}

// CleanInvalidEntries removes invalid entries from the registry
func (m *Manager) CleanInvalidEntries() int {
	cleaned := 0
	
	for chainID, network := range m.registry.Networks {
		toDelete := make([]string, 0)
		
		// Check if this is the old hardcoded Sepolia chainID (11155111)
		// These are definitely dummy entries since sepolia is not configured in foundry.toml
		if chainID == "11155111" {
			// Remove the entire network
			delete(m.registry.Networks, chainID)
			cleaned += len(network.Deployments)
			continue
		}
		
		for key, deployment := range network.Deployments {
			// Check for dummy entries (all zero salt and init code hash)
			isZeroSalt := deployment.Salt == "" || deployment.Salt == "0000000000000000000000000000000000000000000000000000000000000000"
			isZeroInitCodeHash := deployment.InitCodeHash == "" || deployment.InitCodeHash == "0000000000000000000000000000000000000000000000000000000000000000"
			
			// Mark entries with zero salt AND zero init code hash as potentially invalid
			if isZeroSalt && isZeroInitCodeHash {
				// Check if the broadcast file doesn't exist
				broadcastPath := deployment.Deployment.BroadcastFile
				shouldDelete := false
				
				if broadcastPath == "" {
					shouldDelete = true
				} else if !fileExists(broadcastPath) {
					shouldDelete = true
				}
				
				if shouldDelete {
					toDelete = append(toDelete, key)
				}
			}
		}
		
		// Delete invalid entries
		for _, key := range toDelete {
			delete(network.Deployments, key)
			cleaned++
		}
		
		// Remove empty networks
		if len(network.Deployments) == 0 {
			delete(m.registry.Networks, chainID)
		}
	}
	
	return cleaned
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// migrateFromOldFormat handles migration from byte array format to hex string format
func (m *Manager) migrateFromOldFormat(data []byte) error {
	// Define old format types
	type OldDeploymentEntry struct {
		Address       common.Address         `json:"address"`
		Type          string                 `json:"type"`
		Salt          [32]byte               `json:"salt"`
		InitCodeHash  [32]byte               `json:"init_code_hash"`
		Constructor   []interface{}          `json:"constructor_args,omitempty"`
		Verification  types.Verification     `json:"verification"`
		Deployment    types.DeploymentInfo   `json:"deployment"`
		Metadata      types.ContractMetadata `json:"metadata"`
	}
	
	type OldNetworkEntry struct {
		Name        string                        `json:"name"`
		Deployments map[string]*OldDeploymentEntry `json:"deployments"`
	}
	
	type OldRegistry struct {
		Project  ProjectMetadata           `json:"project"`
		Networks map[string]*OldNetworkEntry `json:"networks"`
	}
	
	// Parse with old format
	var oldRegistry OldRegistry
	if err := json.Unmarshal(data, &oldRegistry); err != nil {
		return err
	}
	
	// Convert to new format
	m.registry = &Registry{
		Project:  oldRegistry.Project,
		Networks: make(map[string]*NetworkEntry),
	}
	
	for chainID, oldNetwork := range oldRegistry.Networks {
		newNetwork := &NetworkEntry{
			Name:        oldNetwork.Name,
			Deployments: make(map[string]*types.DeploymentEntry),
		}
		
		for key, oldDeployment := range oldNetwork.Deployments {
			newDeployment := &types.DeploymentEntry{
				Address:       oldDeployment.Address,
				Type:          oldDeployment.Type,
				Salt:          hex.EncodeToString(oldDeployment.Salt[:]),
				InitCodeHash:  hex.EncodeToString(oldDeployment.InitCodeHash[:]),
				Constructor:   oldDeployment.Constructor,
				Verification:  oldDeployment.Verification,
				Deployment:    oldDeployment.Deployment,
				Metadata:      oldDeployment.Metadata,
			}
			newNetwork.Deployments[key] = newDeployment
		}
		
		m.registry.Networks[chainID] = newNetwork
	}
	
	return nil
}
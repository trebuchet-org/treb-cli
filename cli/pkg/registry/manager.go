package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

type Manager struct {
	registryPath    string
	networkResolver *network.Resolver
	registry        *Registry
	index           map[string]*types.DeploymentEntry
}

type Registry struct {
	Networks  map[string]*NetworkEntry          `json:"networks"`
	Libraries map[string]*types.DeploymentEntry `json:"libraries,omitempty"` // Global libraries by chain
}

type NetworkEntry struct {
	Name        string                            `json:"name"`
	Deployments map[string]*types.DeploymentEntry `json:"deployments"`
}

func NewManager(registryPath string) (*Manager, error) {
	manager := &Manager{
		registryPath: registryPath,
	}

	manager.networkResolver = network.NewResolver(".")

	if err := manager.load(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *Manager) load() error {
	if _, err := os.Stat(m.registryPath); os.IsNotExist(err) {
		// Create empty registry
		m.registry = &Registry{
			Networks:  make(map[string]*NetworkEntry),
			Libraries: make(map[string]*types.DeploymentEntry),
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
		return fmt.Errorf("failed to parse registry file: %w", err)
	}

	// Initialize networks map if nil
	if m.registry.Networks == nil {
		m.registry.Networks = make(map[string]*NetworkEntry)
	}

	m.index = make(map[string]*types.DeploymentEntry)
	for _, network := range m.registry.Networks {
		for _, deployment := range network.Deployments {
			m.index[deployment.FQID] = deployment
			if deployment.NetworkInfo == nil {
				networkInfo, err := m.networkResolver.ResolveNetwork(network.Name)
				if err != nil {
					return fmt.Errorf("failed to resolve network %s: %w", network.Name, err)
				}
				deployment.NetworkInfo = networkInfo
			}
		}
	}

	for _, library := range m.registry.Libraries {
		m.index[library.FQID] = library
	}

	for _, chain := range m.registry.Networks {
		for _, deployment := range chain.Deployments {
			if deployment.Type == types.ProxyDeployment {
				deployment.Target = m.index[deployment.TargetDeploymentFQID]
			}
		}
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

func (m *Manager) RecordDeployment(contractInfo *contracts.ContractInfo, namespace string, result *types.DeploymentResult, chainID uint64) error {
	chainIDStr := fmt.Sprintf("%d", chainID)

	// Ensure network exists
	if m.registry.Networks[chainIDStr] == nil {
		networkName := "unknown"
		if result.NetworkInfo != nil {
			networkName = result.NetworkInfo.Name
		}
		m.registry.Networks[chainIDStr] = &NetworkEntry{
			Name:        networkName,
			Deployments: make(map[string]*types.DeploymentEntry),
		}
	}

	// Default namespace to "default" if not provided
	if namespace == "" {
		namespace = "default"
	}

	entry := &types.DeploymentEntry{
		FQID:         result.FQID,
		ShortID:      result.ShortID,
		Address:      result.Address,
		ContractName: contractInfo.Name,
		Namespace:    namespace,
		Type:         result.DeploymentType, // Now comes from structured output
		Salt:         result.Salt,
		InitCodeHash: result.InitCodeHash,

		// Constructor arguments for verification
		ConstructorArgs: result.ConstructorArgs,

		// Label for all deployments
		Label: result.Label,

		// Proxy-specific fields
		TargetDeploymentFQID: result.TargetDeploymentFQID,

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
			Status:        result.Status,
			SafeAddress:   result.SafeAddress.String(),
			SafeTxHash:    m.getSafeTxHash(result),
			SafeNonce:     m.getSafeNonce(result),
			Deployer:      m.getDeployerFromBroadcast(result.BroadcastFile),
		},

		Metadata: m.buildMetadata(contractInfo, result),
	}

	// Use address as key for uniqueness
	key := strings.ToLower(result.Address.Hex())
	m.registry.Networks[chainIDStr].Deployments[key] = entry

	return m.Save()
}

// RecordLibraryDeployment records a library deployment in the global libraries section
func (m *Manager) RecordLibraryDeployment(contractInfo *contracts.ContractInfo, result *types.DeploymentResult, chainID uint64) error {
	// Initialize libraries map if needed
	if m.registry.Libraries == nil {
		m.registry.Libraries = make(map[string]*types.DeploymentEntry)
	}

	// Create chain-library key (e.g., "44787-MathLib" for Alfajores MathLib)
	key := fmt.Sprintf("%d-%s", chainID, contractInfo.Name)

	entry := &types.DeploymentEntry{
		Address:      result.Address,
		ContractName: contractInfo.Name,
		Namespace:    "global", // Libraries are global
		Type:         "library",
		Salt:         result.Salt,
		InitCodeHash: result.InitCodeHash,

		// No constructor args for libraries typically
		ConstructorArgs: result.ConstructorArgs,

		Verification: types.Verification{
			Status: "pending",
		},

		Deployment: types.DeploymentInfo{
			TxHash:        &result.TxHash,
			BlockNumber:   result.BlockNumber,
			BroadcastFile: result.BroadcastFile,
			Timestamp:     time.Now(),
			Status:        "deployed", // Libraries are always deployed, no Safe
			Deployer:      m.getDeployerFromBroadcast(result.BroadcastFile),
		},

		Metadata: m.buildMetadata(contractInfo, result),
	}

	// Store in libraries section
	m.registry.Libraries[key] = entry

	return m.Save()
}

// GetLibrary retrieves a library deployment for a specific chain
func (m *Manager) GetLibrary(libraryName string, chainID uint64) *types.DeploymentEntry {
	if m.registry.Libraries == nil {
		return nil
	}

	key := fmt.Sprintf("%d-%s", chainID, libraryName)
	return m.registry.Libraries[key]
}

// GetAllLibraries returns all library deployments
func (m *Manager) GetAllLibraries() map[string]*types.DeploymentEntry {
	if m.registry.Libraries == nil {
		return make(map[string]*types.DeploymentEntry)
	}
	return m.registry.Libraries
}

func (m *Manager) GetDeployment(identifier string) *types.DeploymentEntry {
	return m.index[identifier]
}

// QueryDeployments finds deployments matching the given query string
// Query can be:
// - Full FQID: "chainID/env/contractPath:shortID"
// - ShortID: "contract:label"
// - Contract name: "MyToken"
// If chainID and namespace are provided, they are used to narrow down results
func (m *Manager) QueryDeployments(query string, chainID uint64, namespace string) []*DeploymentInfo {
	var results []*DeploymentInfo
	queryLower := strings.ToLower(query)

	// First, check for exact FQID match
	for cID, network := range m.registry.Networks {
		for _, deployment := range network.Deployments {
			if strings.EqualFold(deployment.FQID, query) {
				return []*DeploymentInfo{{
					Address:     deployment.Address,
					NetworkName: network.Name,
					ChainID:     cID,
					Entry:       deployment,
				}}
			}
		}
	}

	// If chainID is provided, only search within that network
	chainIDStr := ""
	if chainID > 0 {
		chainIDStr = fmt.Sprintf("%d", chainID)
	}

	// Search for partial matches
	for cID, network := range m.registry.Networks {
		// Skip if chainID is specified and doesn't match
		if chainIDStr != "" && cID != chainIDStr {
			continue
		}

		for _, deployment := range network.Deployments {
			// Skip if namespace is specified and doesn't match
			if namespace != "" && deployment.Namespace != namespace {
				continue
			}

			matched := false

			// Check if ShortID matches exactly
			if strings.EqualFold(deployment.ShortID, query) {
				matched = true
			} else if strings.EqualFold(deployment.ContractName, query) {
				// Check if contract name matches exactly
				matched = true
			} else if strings.Contains(strings.ToLower(deployment.ShortID), queryLower) {
				// Check if ShortID contains the query
				matched = true
			} else if strings.Contains(strings.ToLower(deployment.ContractName), queryLower) {
				// Check if contract name contains the query
				matched = true
			}

			if matched {
				results = append(results, &DeploymentInfo{
					Address:     deployment.Address,
					NetworkName: network.Name,
					ChainID:     cID,
					Entry:       deployment,
				})
			}
		}
	}

	return results
}

// GetDeploymentWithLabel gets a deployment by contract, namespace, and label
func (m *Manager) GetDeploymentWithLabel(contract, namespace, label string, chainID uint64) *types.DeploymentEntry {
	chainIDStr := fmt.Sprintf("%d", chainID)

	// Default namespace to "default" if not provided
	if namespace == "" {
		namespace = "default"
	}

	if network := m.registry.Networks[chainIDStr]; network != nil {
		// Search through deployments to find matching contract, env, and label
		for _, deployment := range network.Deployments {
			if deployment.ContractName == contract && deployment.Namespace == namespace && deployment.Label == label {
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

func (m *Manager) UpdateDeployment(chainID uint64, deployment *types.DeploymentEntry) error {
	chainIDStr := fmt.Sprintf("%d", chainID)
	address := strings.ToLower(deployment.Address.Hex())
	if network := m.registry.Networks[chainIDStr]; network != nil {
		network.Deployments[address] = deployment
		return m.Save()
	}

	return fmt.Errorf("network not found")
}

func (m *Manager) getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (m *Manager) getDeployerFromBroadcast(broadcastFile string) string {
	if broadcastFile == "" {
		return ""
	}

	content, err := os.ReadFile(broadcastFile)
	if err != nil {
		return ""
	}

	var broadcast struct {
		Transactions []struct {
			Transaction struct {
				From string `json:"from"`
			} `json:"transaction"`
		} `json:"transactions"`
	}

	if err := json.Unmarshal(content, &broadcast); err == nil {
		if len(broadcast.Transactions) > 0 {
			return broadcast.Transactions[0].Transaction.From
		}
	}

	return ""
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
	NetworkCount        int                    `json:"network_count"`
	TotalDeployments    int                    `json:"total_deployments"`
	VerifiedCount       int                    `json:"verified_count"`
	PendingVerification int                    `json:"pending_verification"`
	RecentDeployments   []RecentDeploymentInfo `json:"recent_deployments"`
}

// RecentDeploymentInfo represents recent deployment information
type RecentDeploymentInfo struct {
	Contract    string               `json:"contract"`
	Namespace   string               `json:"namespace"`
	Label       string               `json:"label"`
	Address     string               `json:"address"`
	Network     string               `json:"network"`
	Timestamp   string               `json:"timestamp"`
	Type        types.DeploymentType `json:"type"` // implementation/proxy
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

// AddTag adds a tag to a deployment by address
func (m *Manager) AddTag(address common.Address, tag string) error {
	// Find deployment by address across all networks
	addressLower := strings.ToLower(address.Hex())

	for _, network := range m.registry.Networks {
		if deployment, exists := network.Deployments[addressLower]; exists {
			// Check if tag already exists
			for _, existingTag := range deployment.Tags {
				if existingTag == tag {
					return nil // Tag already exists, no error
				}
			}

			// Add the tag
			deployment.Tags = append(deployment.Tags, tag)
			return nil
		}
	}

	return fmt.Errorf("deployment not found for address %s", address.Hex())
}

// RemoveTag removes a tag from a deployment by address
func (m *Manager) RemoveTag(address common.Address, tag string) error {
	// Find deployment by address across all networks
	addressLower := strings.ToLower(address.Hex())

	for _, network := range m.registry.Networks {
		if deployment, exists := network.Deployments[addressLower]; exists {
			// Find and remove the tag
			for i, existingTag := range deployment.Tags {
				if existingTag == tag {
					// Remove the tag by slicing
					deployment.Tags = append(deployment.Tags[:i], deployment.Tags[i+1:]...)
					return nil
				}
			}
			// Tag not found, but deployment exists - not an error
			return nil
		}
	}

	return fmt.Errorf("deployment not found for address %s", address.Hex())
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
		NetworkCount: len(m.registry.Networks),
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
					Contract:    deployment.ContractName,
					Namespace:   deployment.Namespace,
					Address:     deployment.Address.Hex(),
					Network:     network.Name,
					Timestamp:   deployment.Deployment.Timestamp.Format("2006-01-02 15:04"),
					Type:        deployment.Type,
					Label:       deployment.Label,
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

// buildMetadata builds contract metadata, using values from result.Metadata if available
func (m *Manager) buildMetadata(contractInfo *contracts.ContractInfo, result *types.DeploymentResult) types.ContractMetadata {

	metadata := types.ContractMetadata{
		SourceCommit: m.getGitCommit(),
		Compiler:     contractInfo.Artifact.Metadata.Compiler.Version,
		ScriptPath:   result.Metadata.ScriptPath,
		SourceHash:   result.Metadata.SourceHash,
		ContractPath: contractInfo.Path,
		Extra:        result.Metadata.Extra,
	}

	return metadata
}

// extractContractNameFromPath extracts contract name from contract path
// E.g., "./lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol:TransparentUpgradeableProxy" -> "TransparentUpgradeableProxy"
func extractContractNameFromPath(contractPath string) string {
	// Contract path format: ./path/to/Contract.sol:ContractName
	parts := strings.Split(contractPath, ":")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// getSafeTxHash returns the safe tx hash if it exists
func (m *Manager) getSafeTxHash(result *types.DeploymentResult) *common.Hash {
	if result.SafeTxHash != (common.Hash{}) {
		return &result.SafeTxHash
	}
	return nil
}

// getSafeNonce returns the safe nonce (currently not available in result)
func (m *Manager) getSafeNonce(result *types.DeploymentResult) uint64 {
	// TODO: Extract nonce from Safe transaction if available
	return 0
}

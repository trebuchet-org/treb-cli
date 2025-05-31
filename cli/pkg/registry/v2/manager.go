package v2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

const (
	TrebDir                = ".treb"
	DeploymentsFile        = "deployments.json"
	TransactionsFile       = "transactions.json"
	SafeTransactionsFile   = "safe-txs.json"
	SolidityRegistryFile   = "registry.json"
)

// Manager handles all registry operations for the new data model
type Manager struct {
	rootDir          string
	mu               sync.RWMutex
	deployments      map[string]*types.Deployment
	transactions     map[string]*types.Transaction
	safeTransactions map[string]*types.SafeTransaction
	lookups          *types.LookupIndexes
	solidityRegistry types.SolidityRegistry
}

// NewManager creates a new registry manager
func NewManager(rootDir string) (*Manager, error) {
	trebDir := filepath.Join(rootDir, TrebDir)
	
	// Create .treb directory if it doesn't exist
	if err := os.MkdirAll(trebDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .treb directory: %w", err)
	}

	m := &Manager{
		rootDir:          rootDir,
		deployments:      make(map[string]*types.Deployment),
		transactions:     make(map[string]*types.Transaction),
		safeTransactions: make(map[string]*types.SafeTransaction),
		lookups:          &types.LookupIndexes{
			Version:     "1.0.0",
			ByAddress:   make(map[uint64]map[string]string),
			ByNamespace: make(map[string]map[uint64][]string),
			ByContract:  make(map[string][]string),
			Proxies: types.ProxyIndexes{
				Implementations: make(map[string][]string),
				ProxyToImpl:     make(map[string]string),
			},
			Pending: types.PendingItems{
				SafeTxs: []string{},
			},
		},
		solidityRegistry: make(types.SolidityRegistry),
	}

	// Load existing data
	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return m, nil
}

// load reads all registry files
func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load deployments
	if err := m.loadFile(DeploymentsFile, &m.deployments); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load deployments: %w", err)
	}

	// Load transactions
	if err := m.loadFile(TransactionsFile, &m.transactions); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load transactions: %w", err)
	}

	// Load safe transactions
	if err := m.loadFile(SafeTransactionsFile, &m.safeTransactions); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load safe transactions: %w", err)
	}

	// Load solidity registry
	if err := m.loadFile(SolidityRegistryFile, &m.solidityRegistry); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load solidity registry: %w", err)
	}

	// Build lookups from loaded data
	m.rebuildLookups()

	return nil
}

// loadFile loads a JSON file into the given target
func (m *Manager) loadFile(filename string, target interface{}) error {
	path := filepath.Join(m.rootDir, TrebDir, filename)
	
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(target)
}

// save writes all registry files
func (m *Manager) save() error {
	// Save deployments
	if err := m.saveFile(DeploymentsFile, m.deployments); err != nil {
		return fmt.Errorf("failed to save deployments: %w", err)
	}

	// Save transactions
	if err := m.saveFile(TransactionsFile, m.transactions); err != nil {
		return fmt.Errorf("failed to save transactions: %w", err)
	}

	// Save safe transactions
	if err := m.saveFile(SafeTransactionsFile, m.safeTransactions); err != nil {
		return fmt.Errorf("failed to save safe transactions: %w", err)
	}

	// Save solidity registry
	if err := m.saveFile(SolidityRegistryFile, m.solidityRegistry); err != nil {
		return fmt.Errorf("failed to save solidity registry: %w", err)
	}

	return nil
}

// rebuildLookups rebuilds all lookup indexes from current deployment data
func (m *Manager) rebuildLookups() {
	// Initialize fresh lookup indexes
	m.lookups = &types.LookupIndexes{
		Version:     "1.0.0",
		ByAddress:   make(map[uint64]map[string]string),
		ByNamespace: make(map[string]map[uint64][]string),
		ByContract:  make(map[string][]string),
		Proxies: types.ProxyIndexes{
			Implementations: make(map[string][]string),
			ProxyToImpl:     make(map[string]string),
		},
		Pending: types.PendingItems{
			SafeTxs: []string{},
		},
	}

	// Rebuild from deployments
	for _, deployment := range m.deployments {
		m.updateIndexesForDeployment(deployment)
	}

	// Rebuild pending Safe transactions
	for hash, safeTx := range m.safeTransactions {
		if safeTx.Status == types.TransactionStatusPending {
			m.lookups.Pending.SafeTxs = append(m.lookups.Pending.SafeTxs, hash)
		}
	}
}

// saveFile saves data to a JSON file
func (m *Manager) saveFile(filename string, data interface{}) error {
	path := filepath.Join(m.rootDir, TrebDir, filename)
	
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// GenerateDeploymentID generates a unique deployment ID
func (m *Manager) GenerateDeploymentID(namespace string, chainID uint64, contractName, label string, txHash string) string {
	baseID := fmt.Sprintf("%s/%d/%s:%s", namespace, chainID, contractName, label)
	
	// Check if this ID already exists
	if _, exists := m.deployments[baseID]; !exists {
		return baseID
	}

	// Add transaction prefix for uniqueness
	if txHash != "" && len(txHash) >= 10 {
		return fmt.Sprintf("%s#%s", baseID, txHash[2:6]) // Skip 0x prefix, take first 4 chars
	}

	// Fallback: use timestamp
	return fmt.Sprintf("%s#%d", baseID, time.Now().Unix())
}

// AddDeployment adds a new deployment record
func (m *Manager) AddDeployment(deployment *types.Deployment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to deployments
	m.deployments[deployment.ID] = deployment

	// Update indexes
	m.updateIndexesForDeployment(deployment)

	// Update solidity registry
	m.updateSolidityRegistry(deployment)

	// Save all files
	return m.save()
}

// updateIndexesForDeployment updates lookup indexes for a deployment
func (m *Manager) updateIndexesForDeployment(deployment *types.Deployment) {
	// Update address index (use lowercase for consistency)
	if m.lookups.ByAddress[deployment.ChainID] == nil {
		m.lookups.ByAddress[deployment.ChainID] = make(map[string]string)
	}
	m.lookups.ByAddress[deployment.ChainID][strings.ToLower(deployment.Address)] = deployment.ID

	// Update namespace index
	if m.lookups.ByNamespace[deployment.Namespace] == nil {
		m.lookups.ByNamespace[deployment.Namespace] = make(map[uint64][]string)
	}
	namespaceChain := m.lookups.ByNamespace[deployment.Namespace][deployment.ChainID]
	if !contains(namespaceChain, deployment.ID) {
		m.lookups.ByNamespace[deployment.Namespace][deployment.ChainID] = append(namespaceChain, deployment.ID)
	}

	// Update contract index
	contractList := m.lookups.ByContract[deployment.ContractName]
	if !contains(contractList, deployment.ID) {
		m.lookups.ByContract[deployment.ContractName] = append(contractList, deployment.ID)
	}

	// Update proxy indexes if applicable
	if deployment.Type == types.ProxyDeployment && deployment.ProxyInfo != nil {
		m.lookups.Proxies.ProxyToImpl[deployment.ID] = deployment.ProxyInfo.Implementation
		
		// Find implementation deployment ID by address
		for id, dep := range m.deployments {
			if dep.Address == deployment.ProxyInfo.Implementation {
				implList := m.lookups.Proxies.Implementations[id]
				if !contains(implList, deployment.ID) {
					m.lookups.Proxies.Implementations[id] = append(implList, deployment.ID)
				}
				break
			}
		}
	}
}

// updateSolidityRegistry updates the simplified Solidity registry
func (m *Manager) updateSolidityRegistry(deployment *types.Deployment) {
	if m.solidityRegistry[deployment.ChainID] == nil {
		m.solidityRegistry[deployment.ChainID] = make(map[string]map[string]string)
	}
	if m.solidityRegistry[deployment.ChainID][deployment.Namespace] == nil {
		m.solidityRegistry[deployment.ChainID][deployment.Namespace] = make(map[string]string)
	}

	key := fmt.Sprintf("%s:%s", deployment.ContractName, deployment.Label)
	m.solidityRegistry[deployment.ChainID][deployment.Namespace][key] = deployment.Address
}

// AddTransaction adds a new transaction record
func (m *Manager) AddTransaction(tx *types.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transactions[tx.ID] = tx
	return m.save()
}

// AddSafeTransaction adds a new Safe transaction record
func (m *Manager) AddSafeTransaction(safeTx *types.SafeTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.safeTransactions[safeTx.SafeTxHash] = safeTx

	// Update pending list if needed
	if safeTx.Status == types.TransactionStatusPending {
		if !contains(m.lookups.Pending.SafeTxs, safeTx.SafeTxHash) {
			m.lookups.Pending.SafeTxs = append(m.lookups.Pending.SafeTxs, safeTx.SafeTxHash)
		}
	}

	return m.save()
}

// GetDeployment retrieves a deployment by ID
func (m *Manager) GetDeployment(id string) (*types.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deployment, exists := m.deployments[id]
	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", id)
	}

	return deployment, nil
}

// GetDeploymentByAddress retrieves a deployment by chain and address
func (m *Manager) GetDeploymentByAddress(chainID uint64, address string) (*types.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chainAddresses, exists := m.lookups.ByAddress[chainID]
	if !exists {
		return nil, fmt.Errorf("no deployments on chain %d", chainID)
	}

	deploymentID, exists := chainAddresses[strings.ToLower(address)]
	if !exists {
		return nil, fmt.Errorf("deployment not found at address %s on chain %d", address, chainID)
	}

	return m.deployments[deploymentID], nil
}

// GetDeploymentsByNamespace retrieves all deployments in a namespace
func (m *Manager) GetDeploymentsByNamespace(namespace string, chainID uint64) ([]*types.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaceData, exists := m.lookups.ByNamespace[namespace]
	if !exists {
		return nil, nil // No deployments in this namespace
	}

	deploymentIDs, exists := namespaceData[chainID]
	if !exists {
		return nil, nil // No deployments on this chain
	}

	deployments := make([]*types.Deployment, 0, len(deploymentIDs))
	for _, id := range deploymentIDs {
		if deployment, exists := m.deployments[id]; exists {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// GetAllDeployments returns all deployments as a slice
func (m *Manager) GetAllDeployments() []*types.Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a slice to prevent external modifications
	result := make([]*types.Deployment, 0, len(m.deployments))
	for _, v := range m.deployments {
		result = append(result, v)
	}
	return result
}

// GetAllDeploymentsHydrated returns all deployments with linked transaction data
func (m *Manager) GetAllDeploymentsHydrated() []*types.Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*types.Deployment, 0, len(m.deployments))
	for _, deployment := range m.deployments {
		// Create a copy to avoid modifying the original
		dep := *deployment
		
		// Hydrate with transaction data if available
		if dep.TransactionID != "" {
			if tx, exists := m.transactions[dep.TransactionID]; exists {
				// Add transaction reference to deployment
				dep.Transaction = tx
			}
		}
		
		result = append(result, &dep)
	}
	return result
}

// GetTransaction retrieves a transaction by ID
func (m *Manager) GetTransaction(id string) (*types.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, exists := m.transactions[id]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", id)
	}

	return tx, nil
}

// GetSafeTransaction retrieves a Safe transaction by hash
func (m *Manager) GetSafeTransaction(safeTxHash string) (*types.SafeTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	safeTx, exists := m.safeTransactions[safeTxHash]
	if !exists {
		return nil, fmt.Errorf("safe transaction not found: %s", safeTxHash)
	}

	return safeTx, nil
}

// SaveDeployment updates an existing deployment record
func (m *Manager) SaveDeployment(deployment *types.Deployment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if deployment exists
	if _, exists := m.deployments[deployment.ID]; !exists {
		return fmt.Errorf("deployment not found: %s", deployment.ID)
	}

	// Update the deployment
	deployment.UpdatedAt = time.Now()
	m.deployments[deployment.ID] = deployment

	// Update indexes in case address changed
	m.updateIndexesForDeployment(deployment)

	// Update solidity registry
	m.updateSolidityRegistry(deployment)

	// Save all files
	return m.save()
}

// UpdateDeploymentVerification updates the verification status of a deployment
func (m *Manager) UpdateDeploymentVerification(deploymentID string, status types.VerificationStatus, etherscanURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deployment, exists := m.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	deployment.Verification.Status = status
	deployment.Verification.EtherscanURL = etherscanURL
	if status == types.VerificationStatusVerified {
		now := time.Now()
		deployment.Verification.VerifiedAt = &now
	}
	deployment.UpdatedAt = time.Now()

	return m.save()
}

// AddTag adds a tag to a deployment
func (m *Manager) AddTag(deploymentID string, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deployment, exists := m.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// Check if tag already exists
	for _, existingTag := range deployment.Tags {
		if existingTag == tag {
			return nil // Tag already exists, no error
		}
	}

	// Add tag
	deployment.Tags = append(deployment.Tags, tag)
	deployment.UpdatedAt = time.Now()

	return m.save()
}

// RemoveTag removes a tag from a deployment
func (m *Manager) RemoveTag(deploymentID string, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deployment, exists := m.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// Find and remove tag
	newTags := make([]string, 0, len(deployment.Tags))
	found := false
	for _, existingTag := range deployment.Tags {
		if existingTag != tag {
			newTags = append(newTags, existingTag)
		} else {
			found = true
		}
	}

	if !found {
		return nil // Tag doesn't exist, no error
	}

	// Update tags
	deployment.Tags = newTags
	deployment.UpdatedAt = time.Now()

	return m.save()
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
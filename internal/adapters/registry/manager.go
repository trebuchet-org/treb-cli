package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

const (
	TrebDir              = ".treb"
	DeploymentsFile      = "deployments.json"
	TransactionsFile     = "transactions.json"
	SafeTransactionsFile = "safe-txs.json"
	SolidityRegistryFile = "registry.json"
)

// Manager handles all registry operations for the new data model
type Manager struct {
	rootDir          string
	mu               sync.RWMutex
	deployments      map[string]*domain.Deployment
	transactions     map[string]*domain.Transaction
	safeTransactions map[string]*domain.SafeTransaction
	lookups          *domain.LookupIndexes
	solidityRegistry domain.SolidityRegistry
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
		deployments:      make(map[string]*domain.Deployment),
		transactions:     make(map[string]*domain.Transaction),
		safeTransactions: make(map[string]*domain.SafeTransaction),
		lookups: &domain.LookupIndexes{
			Version:     "1.0.0",
			ByAddress:   make(map[uint64]map[string]string),
			ByNamespace: make(map[string]map[uint64][]string),
			ByContract:  make(map[string][]string),
			Proxies: domain.ProxyIndexes{
				Implementations: make(map[string][]string),
				ProxyToImpl:     make(map[string]string),
			},
			Pending: domain.PendingItems{
				SafeTxs: []string{},
			},
		},
		solidityRegistry: make(domain.SolidityRegistry),
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

// loadFile loads a JSON file from the .treb directory
func (m *Manager) loadFile(filename string, v interface{}) error {
	path := filepath.Join(m.rootDir, TrebDir, filename)
	
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
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

// saveFile saves data to a JSON file in the .treb directory
func (m *Manager) saveFile(filename string, v interface{}) error {
	path := filepath.Join(m.rootDir, TrebDir, filename)
	
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// rebuildLookups rebuilds all lookup indexes from the loaded data
func (m *Manager) rebuildLookups() {
	m.lookups.ByAddress = make(map[uint64]map[string]string)
	m.lookups.ByNamespace = make(map[string]map[uint64][]string)
	m.lookups.ByContract = make(map[string][]string)
	m.lookups.Proxies.Implementations = make(map[string][]string)
	m.lookups.Proxies.ProxyToImpl = make(map[string]string)

	for id, dep := range m.deployments {
		// By address
		if m.lookups.ByAddress[dep.ChainID] == nil {
			m.lookups.ByAddress[dep.ChainID] = make(map[string]string)
		}
		m.lookups.ByAddress[dep.ChainID][strings.ToLower(dep.Address)] = id

		// By namespace
		if m.lookups.ByNamespace[dep.Namespace] == nil {
			m.lookups.ByNamespace[dep.Namespace] = make(map[uint64][]string)
		}
		m.lookups.ByNamespace[dep.Namespace][dep.ChainID] = append(
			m.lookups.ByNamespace[dep.Namespace][dep.ChainID], id,
		)

		// By contract
		m.lookups.ByContract[dep.ContractName] = append(m.lookups.ByContract[dep.ContractName], id)

		// Proxy indexes
		if dep.Type == domain.ProxyDeployment && dep.ProxyInfo != nil {
			implAddr := strings.ToLower(dep.ProxyInfo.Implementation)
			m.lookups.Proxies.Implementations[implAddr] = append(
				m.lookups.Proxies.Implementations[implAddr], id,
			)
			m.lookups.Proxies.ProxyToImpl[strings.ToLower(dep.Address)] = implAddr
		}
	}

	// Rebuild pending items
	m.lookups.Pending.SafeTxs = []string{}
	for id, tx := range m.safeTransactions {
		if tx.Status == domain.SafeTxStatusQueued {
			m.lookups.Pending.SafeTxs = append(m.lookups.Pending.SafeTxs, id)
		}
	}
}

// GetDeployment retrieves a deployment by ID
func (m *Manager) GetDeployment(id string) (*domain.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dep, exists := m.deployments[id]
	if !exists {
		return nil, fmt.Errorf("deployment %s not found", id)
	}

	// Clone to avoid mutations
	clone := *dep
	return &clone, nil
}

// GetDeploymentByAddress retrieves a deployment by chain ID and address
func (m *Manager) GetDeploymentByAddress(chainID uint64, address string) (*domain.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chainAddrs, exists := m.lookups.ByAddress[chainID]
	if !exists {
		return nil, fmt.Errorf("no deployments found on chain %d", chainID)
	}

	id, exists := chainAddrs[strings.ToLower(address)]
	if !exists {
		return nil, fmt.Errorf("deployment at address %s not found on chain %d", address, chainID)
	}

	return m.GetDeployment(id)
}

// GetAllDeployments returns all deployments
func (m *Manager) GetAllDeployments() map[string]*domain.Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid mutations
	result := make(map[string]*domain.Deployment)
	for k, v := range m.deployments {
		clone := *v
		result[k] = &clone
	}
	return result
}

// GetAllDeploymentsHydrated returns all deployments with linked data
func (m *Manager) GetAllDeploymentsHydrated() []*domain.Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Deployment
	for _, dep := range m.deployments {
		clone := *dep
		
		// Link transaction if available
		if dep.TransactionID != "" {
			if tx, exists := m.transactions[dep.TransactionID]; exists {
				txClone := *tx
				clone.Transaction = &txClone
			}
		}

		// Link implementation for proxies
		if dep.Type == domain.ProxyDeployment && dep.ProxyInfo != nil {
			implID, err := m.findImplementationID(dep.ChainID, dep.ProxyInfo.Implementation)
			if err == nil && implID != "" {
				if impl, exists := m.deployments[implID]; exists {
					implClone := *impl
					clone.Implementation = &implClone
				}
			}
		}

		result = append(result, &clone)
	}

	return result
}

// findImplementationID finds the deployment ID for an implementation address
func (m *Manager) findImplementationID(chainID uint64, address string) (string, error) {
	chainAddrs, exists := m.lookups.ByAddress[chainID]
	if !exists {
		return "", fmt.Errorf("no deployments found on chain %d", chainID)
	}

	id, exists := chainAddrs[strings.ToLower(address)]
	if !exists {
		return "", fmt.Errorf("implementation at address %s not found", address)
	}

	return id, nil
}

// SaveDeployment saves or updates a deployment
func (m *Manager) SaveDeployment(deployment *domain.Deployment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set timestamps
	if deployment.CreatedAt.IsZero() {
		deployment.CreatedAt = time.Now()
	}
	deployment.UpdatedAt = time.Now()

	// Save deployment
	m.deployments[deployment.ID] = deployment

	// Rebuild lookups
	m.rebuildLookups()

	// Persist to disk
	return m.save()
}

// SaveTransaction saves or updates a transaction
func (m *Manager) SaveTransaction(tx *domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set timestamp
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}

	// Save transaction
	m.transactions[tx.ID] = tx

	// Persist to disk
	return m.save()
}

// DeleteDeployment removes a deployment by ID
func (m *Manager) DeleteDeployment(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.deployments[id]; !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	delete(m.deployments, id)

	// Rebuild lookups
	m.rebuildLookups()

	// Persist to disk
	return m.save()
}

// UpdateDeploymentVerification updates the verification status of a deployment
func (m *Manager) UpdateDeploymentVerification(id string, status domain.VerificationStatus, verifiers map[string]domain.VerifierStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[id]
	if !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	dep.Verification.Status = status
	if status == domain.VerificationStatusVerified {
		now := time.Now()
		dep.Verification.VerifiedAt = &now
	}
	if verifiers != nil {
		dep.Verification.Verifiers = verifiers
	}
	dep.UpdatedAt = time.Now()

	// Persist to disk
	return m.save()
}

// TagDeployment adds a tag to a deployment
func (m *Manager) TagDeployment(id string, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[id]
	if !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	// Check if tag already exists
	for _, t := range dep.Tags {
		if t == tag {
			return nil // Tag already exists
		}
	}

	dep.Tags = append(dep.Tags, tag)
	dep.UpdatedAt = time.Now()

	// Persist to disk
	return m.save()
}

// GetTransaction retrieves a transaction by ID
func (m *Manager) GetTransaction(id string) (*domain.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, exists := m.transactions[id]
	if !exists {
		return nil, fmt.Errorf("transaction %s not found", id)
	}

	// Clone to avoid mutations
	clone := *tx
	return &clone, nil
}

// GetSafeTransaction retrieves a safe transaction by ID
func (m *Manager) GetSafeTransaction(id string) (*domain.SafeTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, exists := m.safeTransactions[id]
	if !exists {
		return nil, fmt.Errorf("safe transaction %s not found", id)
	}

	// Clone to avoid mutations
	clone := *tx
	return &clone, nil
}

// SaveSafeTransaction saves or updates a safe transaction
func (m *Manager) SaveSafeTransaction(tx *domain.SafeTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set timestamp
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}

	// Save transaction
	m.safeTransactions[tx.ID] = tx

	// Rebuild lookups to update pending items
	m.rebuildLookups()

	// Persist to disk
	return m.save()
}

// GetPendingSafeTransactions returns all pending safe transactions
func (m *Manager) GetPendingSafeTransactions() ([]*domain.SafeTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.SafeTransaction
	for _, id := range m.lookups.Pending.SafeTxs {
		if tx, exists := m.safeTransactions[id]; exists {
			clone := *tx
			result = append(result, &clone)
		}
	}

	return result, nil
}

// GetAllTransactions returns all transactions
func (m *Manager) GetAllTransactions() map[string]*domain.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transactions := make(map[string]*domain.Transaction)
	for k, v := range m.transactions {
		transactions[k] = v
	}
	return transactions
}

// GetAllSafeTransactions returns all safe transactions
func (m *Manager) GetAllSafeTransactions() map[string]*domain.SafeTransaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	safeTransactions := make(map[string]*domain.SafeTransaction)
	for k, v := range m.safeTransactions {
		safeTransactions[k] = v
	}
	return safeTransactions
}

// RemoveDeployment removes a deployment from the registry
func (m *Manager) RemoveDeployment(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.deployments[id]; !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	delete(m.deployments, id)
	return m.save()
}

// RemoveTransaction removes a transaction from the registry
func (m *Manager) RemoveTransaction(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transactions[id]; !exists {
		return fmt.Errorf("transaction %s not found", id)
	}

	delete(m.transactions, id)
	return m.save()
}

// RemoveSafeTransaction removes a safe transaction from the registry
func (m *Manager) RemoveSafeTransaction(safeTxHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var idToRemove string
	for id, tx := range m.safeTransactions {
		if tx.SafeTxHash == safeTxHash {
			idToRemove = id
			break
		}
	}

	if idToRemove == "" {
		return fmt.Errorf("safe transaction %s not found", safeTxHash)
	}

	delete(m.safeTransactions, idToRemove)
	return m.save()
}

// UpdateSafeTransaction updates an existing safe transaction
func (m *Manager) UpdateSafeTransaction(tx *domain.SafeTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tx == nil || tx.ID == "" {
		return fmt.Errorf("invalid safe transaction")
	}

	if _, exists := m.safeTransactions[tx.ID]; !exists {
		return fmt.Errorf("safe transaction %s not found", tx.ID)
	}

	m.safeTransactions[tx.ID] = tx
	return m.save()
}
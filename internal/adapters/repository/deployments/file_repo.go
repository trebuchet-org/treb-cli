package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

const (
	TrebDir              = ".treb"
	DeploymentsFile      = "deployments.json"
	TransactionsFile     = "transactions.json"
	SafeTransactionsFile = "safe-txs.json"
	SolidityRegistryFile = "registry.json"
)

// FileRepository stores the deployments in json files on the system
type FileRepository struct {
	rootDir          string
	lookups          *LookupIndexes
	mu               sync.RWMutex
	deployments      map[string]*models.Deployment
	transactions     map[string]*models.Transaction
	safeTransactions map[string]*models.SafeTransaction
	solidityRegistry SolidityRegistry
}

// NewFileRepository creates a new registry manager
func NewFileRepository(rootDir string) (*FileRepository, error) {
	trebDir := filepath.Join(rootDir, TrebDir)

	// Create .treb directory if it doesn't exist
	if err := os.MkdirAll(trebDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .treb directory: %w", err)
	}

	m := &FileRepository{
		rootDir:          rootDir,
		deployments:      make(map[string]*models.Deployment),
		transactions:     make(map[string]*models.Transaction),
		safeTransactions: make(map[string]*models.SafeTransaction),
		lookups: &LookupIndexes{
			Version:     "1.0.0",
			ByAddress:   make(map[uint64]map[string]string),
			ByNamespace: make(map[string]map[uint64][]string),
			ByContract:  make(map[string][]string),
			Proxies: ProxyIndexes{
				Implementations: make(map[string][]string),
				ProxyToImpl:     make(map[string]string),
			},
			Pending: PendingItems{
				SafeTxs: []string{},
			},
		},
		solidityRegistry: make(SolidityRegistry),
	}

	// Load existing data
	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return m, nil
}

// load reads all registry files
func (m *FileRepository) load() error {
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
func (m *FileRepository) loadFile(filename string, v any) error {
	path := filepath.Join(m.rootDir, TrebDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// save writes all registry files
func (m *FileRepository) save() error {
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
func (m *FileRepository) saveFile(filename string, v any) error {
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
func (m *FileRepository) rebuildLookups() {
	m.lookups.ByAddress = make(map[uint64]map[string]string)
	m.lookups.ByNamespace = make(map[string]map[uint64][]string)
	m.lookups.ByContract = make(map[string][]string)
	m.lookups.Proxies.Implementations = make(map[string][]string)
	m.lookups.Proxies.ProxyToImpl = make(map[string]string)

	// Clear and rebuild solidity registry
	m.solidityRegistry = make(SolidityRegistry)

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
		if dep.Type == models.ProxyDeployment && dep.ProxyInfo != nil {
			implAddr := strings.ToLower(dep.ProxyInfo.Implementation)
			m.lookups.Proxies.Implementations[implAddr] = append(
				m.lookups.Proxies.Implementations[implAddr], id,
			)
			m.lookups.Proxies.ProxyToImpl[strings.ToLower(dep.Address)] = implAddr
		}

		// Update solidity registry
		m.updateSolidityRegistry(dep)
	}

	// Rebuild pending items
	m.lookups.Pending.SafeTxs = []string{}
	for id, tx := range m.safeTransactions {
		if tx.Status == models.TransactionStatusQueued {
			m.lookups.Pending.SafeTxs = append(m.lookups.Pending.SafeTxs, id)
		}
	}
}

// updateSolidityRegistry updates the simplified Solidity registry
func (m *FileRepository) updateSolidityRegistry(deployment *models.Deployment) {
	if m.solidityRegistry[deployment.ChainID] == nil {
		m.solidityRegistry[deployment.ChainID] = make(map[string]map[string]string)
	}
	if m.solidityRegistry[deployment.ChainID][deployment.Namespace] == nil {
		m.solidityRegistry[deployment.ChainID][deployment.Namespace] = make(map[string]string)
	}

	// Use contract display name for registry key
	key := deployment.ContractDisplayName()
	m.solidityRegistry[deployment.ChainID][deployment.Namespace][key] = deployment.Address
}

// GetDeployment retrieves a deployment by ID
func (m *FileRepository) GetDeployment(ctx context.Context, id string) (*models.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dep, exists := m.deployments[id]
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Clone to avoid mutations
	clone := *dep

	// Link transaction if available
	if dep.TransactionID != "" {
		if tx, exists := m.transactions[dep.TransactionID]; exists {
			txClone := *tx
			clone.Transaction = &txClone
		}
	}

	return &clone, nil
}

// GetDeploymentByAddress retrieves a deployment by chain ID and address
func (m *FileRepository) GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*models.Deployment, error) {
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

	// Directly return the deployment to avoid recursive lock
	dep, exists := m.deployments[id]
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Clone to avoid mutations
	clone := *dep

	// Link transaction if available
	if dep.TransactionID != "" {
		if tx, exists := m.transactions[dep.TransactionID]; exists {
			txClone := *tx
			clone.Transaction = &txClone
		}
	}

	return &clone, nil
}

// GetAllDeployments returns all deployments
func (m *FileRepository) GetAllDeployments(ctx context.Context) []*models.Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a slice of cloned deployments
	result := make([]*models.Deployment, 0, len(m.deployments))
	for _, v := range m.deployments {
		clone := *v
		result = append(result, &clone)
	}
	return result
}

// ListDeployments retrieves deployments matching the filter
func (m *FileRepository) ListDeployments(ctx context.Context, filter domain.DeploymentFilter) ([]*models.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Deployment
	for _, dep := range m.deployments {
		// Apply filters
		if filter.Namespace != "" && dep.Namespace != filter.Namespace {
			continue
		}
		if filter.ChainID != 0 && dep.ChainID != filter.ChainID {
			continue
		}
		if filter.ContractName != "" && dep.ContractName != filter.ContractName {
			continue
		}
		if filter.Label != "" && dep.Label != filter.Label {
			continue
		}
		if filter.Type != "" && dep.Type != filter.Type {
			continue
		}

		// Clone and add to result
		clone := *dep
		if dep.TransactionID != "" {
			if tx, exists := m.transactions[dep.TransactionID]; exists {
				txClone := *tx
				clone.Transaction = &txClone
			}
		}
		result = append(result, &clone)
	}

	return result, nil
}

// ListTransactions lists transactions based on filter criteria
func (m *FileRepository) ListTransactions(ctx context.Context, filter domain.TransactionFilter) ([]*models.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Transaction

	for _, tx := range m.transactions {
		// Apply filters
		if filter.ChainID != 0 && tx.ChainID != filter.ChainID {
			continue
		}
		if filter.Status != "" && tx.Status != filter.Status {
			continue
		}
		if filter.Namespace != "" && tx.Environment != filter.Namespace {
			continue
		}

		// Clone and add to result
		clone := *tx
		result = append(result, &clone)
	}

	return result, nil
}

// GetAllDeploymentsHydrated returns all deployments with linked data
func (m *FileRepository) GetAllDeploymentsHydrated() []*models.Deployment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Deployment
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
		if dep.Type == models.ProxyDeployment && dep.ProxyInfo != nil {
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
func (m *FileRepository) findImplementationID(chainID uint64, address string) (string, error) {
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
func (m *FileRepository) SaveDeployment(ctx context.Context, deployment *models.Deployment) error {
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
func (m *FileRepository) SaveTransaction(ctx context.Context, tx *models.Transaction) error {
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
func (m *FileRepository) DeleteDeployment(ctx context.Context, id string) error {
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
func (m *FileRepository) UpdateDeploymentVerification(id string, status models.VerificationStatus, verifiers map[string]models.VerifierStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[id]
	if !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	dep.Verification.Status = status
	if status == models.VerificationStatusVerified {
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
func (m *FileRepository) TagDeployment(id string, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[id]
	if !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	// Check if tag already exists
	if slices.Contains(dep.Tags, tag) {
		return domain.ErrAlreadyExists
	}

	dep.Tags = append(dep.Tags, tag)
	dep.UpdatedAt = time.Now()

	// Persist to disk
	return m.save()
}

// GetTransaction retrieves a transaction by ID
func (m *FileRepository) GetTransaction(ctx context.Context, id string) (*models.Transaction, error) {
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
func (m *FileRepository) GetSafeTransaction(ctx context.Context, id string) (*models.SafeTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, exists := m.safeTransactions[id]
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Clone to avoid mutations
	clone := *tx
	return &clone, nil
}

// SaveSafeTransaction saves or updates a safe transaction
func (m *FileRepository) SaveSafeTransaction(ctx context.Context, tx *models.SafeTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set timestamp
	if tx.ProposedAt.IsZero() {
		tx.ProposedAt = time.Now()
	}

	// Save transaction
	m.safeTransactions[tx.SafeTxHash] = tx

	// Rebuild lookups to update pending items
	m.rebuildLookups()

	// Persist to disk
	return m.save()
}

// GetPendingSafeTransactions returns all pending safe transactions
func (m *FileRepository) GetPendingSafeTransactions() ([]*models.SafeTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.SafeTransaction
	for _, id := range m.lookups.Pending.SafeTxs {
		if tx, exists := m.safeTransactions[id]; exists {
			clone := *tx
			result = append(result, &clone)
		}
	}

	return result, nil
}

// GetAllTransactions returns all transactions
func (m *FileRepository) GetAllTransactions(ctx context.Context) map[string]*models.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transactions := make(map[string]*models.Transaction)
	maps.Copy(transactions, m.transactions)
	return transactions
}

// GetAllSafeTransactions returns all safe transactions
func (m *FileRepository) GetAllSafeTransactions(ctx context.Context) map[string]*models.SafeTransaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	safeTransactions := make(map[string]*models.SafeTransaction)
	for k, v := range m.safeTransactions {
		safeTransactions[k] = v
	}
	return safeTransactions
}

// RemoveDeployment removes a deployment from the registry
func (m *FileRepository) RemoveDeployment(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.deployments[id]; !exists {
		return fmt.Errorf("deployment %s not found", id)
	}

	delete(m.deployments, id)
	return m.save()
}

// RemoveTransaction removes a transaction from the registry
func (m *FileRepository) RemoveTransaction(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transactions[id]; !exists {
		return fmt.Errorf("transaction %s not found", id)
	}

	delete(m.transactions, id)
	return m.save()
}

// RemoveSafeTransaction removes a safe transaction from the registry
func (m *FileRepository) RemoveSafeTransaction(safeTxHash string) error {
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
func (m *FileRepository) UpdateSafeTransaction(ctx context.Context, tx *models.SafeTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tx == nil || tx.SafeTxHash == "" {
		return fmt.Errorf("invalid safe transaction")
	}

	if _, exists := m.safeTransactions[tx.SafeTxHash]; !exists {
		return fmt.Errorf("safe transaction %s not found", tx.SafeTxHash)
	}

	m.safeTransactions[tx.SafeTxHash] = tx
	return m.save()
}

// BatchUpdate applies multiple updates to the registry in a single transaction
type BatchUpdate struct {
	Deployments      []*models.Deployment
	Transactions     []*models.Transaction
	SafeTransactions []*models.SafeTransaction
}

// ListSafeTransactions lists safe transactions based on filter criteria
func (m *FileRepository) ListSafeTransactions(ctx context.Context, filter domain.SafeTransactionFilter) ([]*models.SafeTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.SafeTransaction

	for _, tx := range m.safeTransactions {
		// Apply filters
		if filter.ChainID != 0 && tx.ChainID != filter.ChainID {
			continue
		}
		if filter.SafeAddress != "" && tx.SafeAddress != filter.SafeAddress {
			continue
		}
		if filter.Status != "" && tx.Status != filter.Status {
			continue
		}

		// Clone and add to result
		clone := *tx
		result = append(result, &clone)
	}

	return result, nil
}

// NewFileRepositoryFromConfig creates a new FileRepository from RuntimeConfig
func NewFileRepositoryFromConfig(cfg *config.RuntimeConfig) (*FileRepository, error) {
	return NewFileRepository(cfg.ProjectRoot)
}

var _ usecase.DeploymentRepository = (*FileRepository)(nil)
var _ usecase.DeploymentRepositoryUpdater = (*FileRepository)(nil)

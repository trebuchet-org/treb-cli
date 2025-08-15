package fs

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// RegistryStoreAdapter wraps the existing registry.Manager to implement DeploymentStore
type RegistryStoreAdapter struct {
	manager *registry.Manager
}

// NewRegistryStoreAdapter creates a new adapter wrapping the existing registry manager
func NewRegistryStoreAdapter(cfg *config.RuntimeConfig) (*RegistryStoreAdapter, error) {
	manager, err := registry.NewManager(cfg.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}
	return &RegistryStoreAdapter{manager: manager}, nil
}

// GetDeployment retrieves a deployment by ID
func (r *RegistryStoreAdapter) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	dep, err := r.manager.GetDeployment(id)
	if err != nil {
		// Check if error message indicates not found
		if strings.Contains(err.Error(), "not found") {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return convertToDomainDeployment(dep), nil
}

// GetDeploymentByAddress retrieves a deployment by chain ID and address
func (r *RegistryStoreAdapter) GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*domain.Deployment, error) {
	dep, err := r.manager.GetDeploymentByAddress(chainID, address)
	if err != nil {
		// Check if error message indicates not found
		if strings.Contains(err.Error(), "not found") {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return convertToDomainDeployment(dep), nil
}

// ListDeployments retrieves deployments matching the filter
func (r *RegistryStoreAdapter) ListDeployments(ctx context.Context, filter usecase.DeploymentFilter) ([]*domain.Deployment, error) {
	// Get all deployments and filter them
	allDeps := r.manager.GetAllDeploymentsHydrated()
	
	var result []*domain.Deployment
	for _, dep := range allDeps {
		if matchesFilter(dep, filter) {
			result = append(result, convertToDomainDeployment(dep))
		}
	}
	
	return result, nil
}

// SaveDeployment saves a deployment to the registry
func (r *RegistryStoreAdapter) SaveDeployment(ctx context.Context, deployment *domain.Deployment) error {
	// Convert domain deployment back to types.Deployment
	typeDep := convertFromDomainDeployment(deployment)
	return r.manager.SaveDeployment(typeDep)
}

// DeleteDeployment removes a deployment from the registry
func (r *RegistryStoreAdapter) DeleteDeployment(ctx context.Context, id string) error {
	// The existing registry manager doesn't have a delete method yet
	// This would need to be implemented in the registry package
	return fmt.Errorf("delete deployment not implemented")
}

// matchesFilter checks if a deployment matches the given filter
func matchesFilter(dep *types.Deployment, filter usecase.DeploymentFilter) bool {
	// Check namespace
	if filter.Namespace != "" && dep.Namespace != filter.Namespace {
		return false
	}
	
	// Check chain ID
	if filter.ChainID != 0 && dep.ChainID != filter.ChainID {
		return false
	}
	
	// Check contract name
	if filter.ContractName != "" && dep.ContractName != filter.ContractName {
		return false
	}
	
	// Check label
	if filter.Label != "" && dep.Label != filter.Label {
		return false
	}
	
	// Check type
	if filter.Type != "" && domain.DeploymentType(dep.Type) != filter.Type {
		return false
	}
	
	return true
}

// convertToDomainDeployment converts from pkg/types to domain types
func convertToDomainDeployment(dep *types.Deployment) *domain.Deployment {
	if dep == nil {
		return nil
	}
	
	domainDep := &domain.Deployment{
		ID:            dep.ID,
		Namespace:     dep.Namespace,
		ChainID:       dep.ChainID,
		ContractName:  dep.ContractName,
		Label:         dep.Label,
		Address:       dep.Address,
		Type:          domain.DeploymentType(dep.Type),
		TransactionID: dep.TransactionID,
		DeploymentStrategy: domain.DeploymentStrategy{
			Method:          domain.DeploymentMethod(dep.DeploymentStrategy.Method),
			Salt:            dep.DeploymentStrategy.Salt,
			InitCodeHash:    dep.DeploymentStrategy.InitCodeHash,
			Factory:         dep.DeploymentStrategy.Factory,
			ConstructorArgs: dep.DeploymentStrategy.ConstructorArgs,
			Entropy:         dep.DeploymentStrategy.Entropy,
		},
		Artifact: domain.ArtifactInfo{
			Path:            dep.Artifact.Path,
			CompilerVersion: dep.Artifact.CompilerVersion,
			BytecodeHash:    dep.Artifact.BytecodeHash,
			ScriptPath:      dep.Artifact.ScriptPath,
			GitCommit:       dep.Artifact.GitCommit,
		},
		Verification: domain.VerificationInfo{
			Status:       domain.VerificationStatus(dep.Verification.Status),
			EtherscanURL: dep.Verification.EtherscanURL,
			VerifiedAt:   dep.Verification.VerifiedAt,
			Reason:       dep.Verification.Reason,
		},
		Tags:      dep.Tags,
		CreatedAt: dep.CreatedAt,
		UpdatedAt: dep.UpdatedAt,
	}
	
	// Convert proxy info if present
	if dep.ProxyInfo != nil {
		domainDep.ProxyInfo = &domain.ProxyInfo{
			Type:           dep.ProxyInfo.Type,
			Implementation: dep.ProxyInfo.Implementation,
			Admin:          dep.ProxyInfo.Admin,
		}
		// Convert history
		for _, upgrade := range dep.ProxyInfo.History {
			domainDep.ProxyInfo.History = append(domainDep.ProxyInfo.History, domain.ProxyUpgrade{
				ImplementationID: upgrade.ImplementationID,
				UpgradedAt:       upgrade.UpgradedAt,
				UpgradeTxID:      upgrade.UpgradeTxID,
			})
		}
	}
	
	// Convert verifiers if present
	if dep.Verification.Verifiers != nil {
		domainDep.Verification.Verifiers = make(map[string]domain.VerifierStatus)
		for name, status := range dep.Verification.Verifiers {
			domainDep.Verification.Verifiers[name] = domain.VerifierStatus{
				Status: status.Status,
				URL:    status.URL,
				Reason: status.Reason,
			}
		}
	}
	
	// Convert runtime fields if present
	if dep.Transaction != nil {
		domainDep.Transaction = convertToDomainTransaction(dep.Transaction)
	}
	
	if dep.Implementation != nil {
		domainDep.Implementation = convertToDomainDeployment(dep.Implementation)
	}
	
	return domainDep
}

// convertFromDomainDeployment converts from domain types back to pkg/types
func convertFromDomainDeployment(dep *domain.Deployment) *types.Deployment {
	if dep == nil {
		return nil
	}
	
	typeDep := &types.Deployment{
		ID:            dep.ID,
		Namespace:     dep.Namespace,
		ChainID:       dep.ChainID,
		ContractName:  dep.ContractName,
		Label:         dep.Label,
		Address:       dep.Address,
		Type:          types.DeploymentType(dep.Type),
		TransactionID: dep.TransactionID,
		DeploymentStrategy: types.DeploymentStrategy{
			Method:          types.DeploymentMethod(dep.DeploymentStrategy.Method),
			Salt:            dep.DeploymentStrategy.Salt,
			InitCodeHash:    dep.DeploymentStrategy.InitCodeHash,
			Factory:         dep.DeploymentStrategy.Factory,
			ConstructorArgs: dep.DeploymentStrategy.ConstructorArgs,
			Entropy:         dep.DeploymentStrategy.Entropy,
		},
		Artifact: types.ArtifactInfo{
			Path:            dep.Artifact.Path,
			CompilerVersion: dep.Artifact.CompilerVersion,
			BytecodeHash:    dep.Artifact.BytecodeHash,
			ScriptPath:      dep.Artifact.ScriptPath,
			GitCommit:       dep.Artifact.GitCommit,
		},
		Verification: types.VerificationInfo{
			Status:       types.VerificationStatus(dep.Verification.Status),
			EtherscanURL: dep.Verification.EtherscanURL,
			VerifiedAt:   dep.Verification.VerifiedAt,
			Reason:       dep.Verification.Reason,
		},
		Tags:      dep.Tags,
		CreatedAt: dep.CreatedAt,
		UpdatedAt: dep.UpdatedAt,
	}
	
	// Convert proxy info if present
	if dep.ProxyInfo != nil {
		typeDep.ProxyInfo = &types.ProxyInfo{
			Type:           dep.ProxyInfo.Type,
			Implementation: dep.ProxyInfo.Implementation,
			Admin:          dep.ProxyInfo.Admin,
		}
		// Convert history
		for _, upgrade := range dep.ProxyInfo.History {
			typeDep.ProxyInfo.History = append(typeDep.ProxyInfo.History, types.ProxyUpgrade{
				ImplementationID: upgrade.ImplementationID,
				UpgradedAt:       upgrade.UpgradedAt,
				UpgradeTxID:      upgrade.UpgradeTxID,
			})
		}
	}
	
	// Convert verifiers if present
	if dep.Verification.Verifiers != nil {
		typeDep.Verification.Verifiers = make(map[string]types.VerifierStatus)
		for name, status := range dep.Verification.Verifiers {
			typeDep.Verification.Verifiers[name] = types.VerifierStatus{
				Status: status.Status,
				URL:    status.URL,
				Reason: status.Reason,
			}
		}
	}
	
	return typeDep
}

// convertToDomainTransaction converts from pkg/types Transaction to domain Transaction
func convertToDomainTransaction(tx *types.Transaction) *domain.Transaction {
	if tx == nil {
		return nil
	}
	
	domainTx := &domain.Transaction{
		ID:          tx.ID,
		ChainID:     tx.ChainID,
		Hash:        tx.Hash,
		Status:      domain.TransactionStatus(tx.Status),
		BlockNumber: tx.BlockNumber,
		Sender:      tx.Sender,
		Nonce:       tx.Nonce,
		Deployments: tx.Deployments,
		Environment: tx.Environment,
		CreatedAt:   tx.CreatedAt,
	}
	
	// Convert operations
	for _, op := range tx.Operations {
		domainTx.Operations = append(domainTx.Operations, domain.Operation{
			Type:   op.Type,
			Target: op.Target,
			Method: op.Method,
			Result: op.Result,
		})
	}
	
	// Convert Safe context if present
	if tx.SafeContext != nil {
		domainTx.SafeContext = &domain.SafeContext{
			SafeAddress:     tx.SafeContext.SafeAddress,
			SafeTxHash:      tx.SafeContext.SafeTxHash,
			BatchIndex:      tx.SafeContext.BatchIndex,
			ProposerAddress: tx.SafeContext.ProposerAddress,
		}
	}
	
	return domainTx
}

// GetTransaction retrieves a transaction by ID
func (r *RegistryStoreAdapter) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	tx, err := r.manager.GetTransaction(id)
	if err != nil {
		// Check if error message indicates not found
		if strings.Contains(err.Error(), "not found") {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return convertToDomainTransaction(tx), nil
}

// ListTransactions retrieves transactions matching the filter
func (r *RegistryStoreAdapter) ListTransactions(ctx context.Context, filter usecase.TransactionFilter) ([]*domain.Transaction, error) {
	// Get all transactions from registry
	allTxs := r.manager.GetAllTransactions()
	
	// Apply filters
	var filtered []*domain.Transaction
	for _, tx := range allTxs {
		// Apply chain ID filter
		if filter.ChainID != 0 && tx.ChainID != filter.ChainID {
			continue
		}
		
		// Apply status filter
		if filter.Status != "" && domain.TransactionStatus(tx.Status) != filter.Status {
			continue
		}
		
		// Apply namespace filter
		if filter.Namespace != "" && tx.Environment != filter.Namespace {
			continue
		}
		
		filtered = append(filtered, convertToDomainTransaction(tx))
	}
	
	return filtered, nil
}

// SaveTransaction saves a transaction to the registry
func (r *RegistryStoreAdapter) SaveTransaction(ctx context.Context, transaction *domain.Transaction) error {
	// Convert from domain to legacy types
	legacyTx := convertFromDomainTransaction(transaction)
	return r.manager.AddTransaction(legacyTx)
}

// convertFromDomainTransaction converts from domain types back to pkg/types
func convertFromDomainTransaction(tx *domain.Transaction) *types.Transaction {
	if tx == nil {
		return nil
	}
	
	legacyTx := &types.Transaction{
		ID:          tx.ID,
		ChainID:     tx.ChainID,
		Hash:        tx.Hash,
		Status:      types.TransactionStatus(tx.Status),
		BlockNumber: tx.BlockNumber,
		Sender:      tx.Sender,
		Nonce:       tx.Nonce,
		Deployments: tx.Deployments,
		Environment: tx.Environment,
		CreatedAt:   tx.CreatedAt,
	}
	
	// Convert operations
	for _, op := range tx.Operations {
		legacyTx.Operations = append(legacyTx.Operations, types.Operation{
			Type:   op.Type,
			Target: op.Target,
			Method: op.Method,
			Result: op.Result,
		})
	}
	
	// Convert safe context if present
	if tx.SafeContext != nil {
		legacyTx.SafeContext = &types.SafeContext{
			SafeAddress:     tx.SafeContext.SafeAddress,
			SafeTxHash:      tx.SafeContext.SafeTxHash,
			BatchIndex:      tx.SafeContext.BatchIndex,
			ProposerAddress: tx.SafeContext.ProposerAddress,
		}
	}
	
	return legacyTx
}

// GetSafeTransaction retrieves a Safe transaction by hash
func (r *RegistryStoreAdapter) GetSafeTransaction(ctx context.Context, safeTxHash string) (*domain.SafeTransaction, error) {
	tx, err := r.manager.GetSafeTransaction(safeTxHash)
	if err != nil {
		// Check if error message indicates not found
		if strings.Contains(err.Error(), "not found") {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return convertToDomainSafeTransaction(tx), nil
}

// ListSafeTransactions retrieves Safe transactions matching the filter
func (r *RegistryStoreAdapter) ListSafeTransactions(ctx context.Context, filter usecase.SafeTransactionFilter) ([]*domain.SafeTransaction, error) {
	// Get all Safe transactions from registry
	allSafeTxs := r.manager.GetAllSafeTransactions()
	
	// Apply filters
	var filtered []*domain.SafeTransaction
	for _, safeTx := range allSafeTxs {
		// Apply chain ID filter
		if filter.ChainID != 0 && safeTx.ChainID != filter.ChainID {
			continue
		}
		
		// Apply status filter
		if filter.Status != "" && domain.TransactionStatus(safeTx.Status) != filter.Status {
			continue
		}
		
		// Apply Safe address filter
		if filter.SafeAddress != "" && safeTx.SafeAddress != filter.SafeAddress {
			continue
		}
		
		filtered = append(filtered, convertToDomainSafeTransaction(safeTx))
	}
	
	return filtered, nil
}

// SaveSafeTransaction saves a Safe transaction to the registry
func (r *RegistryStoreAdapter) SaveSafeTransaction(ctx context.Context, safeTx *domain.SafeTransaction) error {
	// Convert from domain to legacy types
	legacySafeTx := convertFromDomainSafeTransaction(safeTx)
	return r.manager.AddSafeTransaction(legacySafeTx)
}

// UpdateSafeTransaction updates a Safe transaction in the registry
func (r *RegistryStoreAdapter) UpdateSafeTransaction(ctx context.Context, safeTx *domain.SafeTransaction) error {
	// Convert from domain to legacy types
	legacySafeTx := convertFromDomainSafeTransaction(safeTx)
	return r.manager.UpdateSafeTransaction(legacySafeTx)
}

// convertToDomainSafeTransaction converts from legacy types to domain
func convertToDomainSafeTransaction(tx *types.SafeTransaction) *domain.SafeTransaction {
	if tx == nil {
		return nil
	}
	
	domainTx := &domain.SafeTransaction{
		SafeTxHash:           tx.SafeTxHash,
		ChainID:              tx.ChainID,
		SafeAddress:          tx.SafeAddress,
		Nonce:                tx.Nonce,
		Status:               domain.TransactionStatus(tx.Status),
		ProposedBy:           tx.ProposedBy,
		TransactionIDs:       tx.TransactionIDs,
		ExecutionTxHash:      tx.ExecutionTxHash,
		ExecutedAt:           tx.ExecutedAt,
		ConfirmationCount:    len(tx.Confirmations),
		CreatedAt:            tx.ProposedAt,
		UpdatedAt:            tx.ProposedAt,
	}
	
	// Extract first transaction data if available
	if len(tx.Transactions) > 0 {
		domainTx.To = tx.Transactions[0].To
		domainTx.Value = tx.Transactions[0].Value
		domainTx.Data = tx.Transactions[0].Data
		domainTx.Operation = int(tx.Transactions[0].Operation)
	}
	
	// Convert confirmations
	for _, conf := range tx.Confirmations {
		domainTx.Confirmations = append(domainTx.Confirmations, domain.Confirmation{
			Signer:      conf.Signer,
			Signature:   conf.Signature,
			ConfirmedAt: conf.ConfirmedAt,
		})
	}
	
	return domainTx
}

// convertFromDomainSafeTransaction converts from domain types back to pkg/types
func convertFromDomainSafeTransaction(tx *domain.SafeTransaction) *types.SafeTransaction {
	if tx == nil {
		return nil
	}
	
	legacyTx := &types.SafeTransaction{
		SafeTxHash:      tx.SafeTxHash,
		ChainID:         tx.ChainID,
		SafeAddress:     tx.SafeAddress,
		Nonce:           tx.Nonce,
		Status:          types.TransactionStatus(tx.Status),
		ProposedBy:      tx.ProposedBy,
		TransactionIDs:  tx.TransactionIDs,
		ExecutionTxHash: tx.ExecutionTxHash,
		ExecutedAt:      tx.ExecutedAt,
		ProposedAt:      tx.CreatedAt,
	}
	
	// Convert transaction data if available
	if tx.To != "" {
		legacyTx.Transactions = []types.SafeTxData{
			{
				To:        tx.To,
				Value:     tx.Value,
				Data:      tx.Data,
				Operation: uint8(tx.Operation),
			},
		}
	}
	
	// Convert confirmations
	for _, conf := range tx.Confirmations {
		legacyTx.Confirmations = append(legacyTx.Confirmations, types.Confirmation{
			Signer:      conf.Signer,
			Signature:   conf.Signature,
			ConfirmedAt: conf.ConfirmedAt,
		})
	}
	
	return legacyTx
}

// Ensure the adapter implements the interfaces
var _ usecase.DeploymentStore = (*RegistryStoreAdapter)(nil)
var _ usecase.TransactionStore = (*RegistryStoreAdapter)(nil)
var _ usecase.SafeTransactionStore = (*RegistryStoreAdapter)(nil)
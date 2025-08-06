package registry

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

type ScriptExecutionUpdater struct {
	manager     *Manager
	execution   *parser.ScriptExecution
	namespace   string
	networkName string
	scriptPath  string
}

func (m *Manager) NewScriptExecutionUpdater(execution *parser.ScriptExecution, namespace string, networkName string, scriptPath string) *ScriptExecutionUpdater {
	return &ScriptExecutionUpdater{
		manager:     m,
		execution:   execution,
		namespace:   namespace,
		networkName: networkName,
		scriptPath:  scriptPath,
	}
}

// UpdateFromScriptExecution updates the registry from a script execution
func (u *ScriptExecutionUpdater) Write() error {
	u.manager.mu.Lock()
	defer u.manager.mu.Unlock()

	// Get the current timestamp for all operations
	now := time.Now()

	// Get git commit hash
	commitHash := getGitCommit()

	// Track deployment IDs for linking to transactions
	deploymentsByTxID := make(map[[32]byte][]string)

	// First, process all deployments
	for _, dep := range u.execution.Deployments {
		deployment, err := u.createDeploymentFromRecord(dep, commitHash, now)
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}

		// Add the deployment
		if err := u.manager.addDeploymentInternal(deployment); err != nil {
			return fmt.Errorf("failed to add deployment %s: %w", deployment.ID, err)
		}

		// Track which transaction this deployment belongs to
		deploymentsByTxID[dep.TransactionID] = append(deploymentsByTxID[dep.TransactionID], deployment.ID)
	}

	// Process transactions
	for _, tx := range u.execution.Transactions {
		transaction := u.createTransactionFromExecution(tx, u.execution.ChainID, u.namespace, now)

		// Link deployments to transaction
		if deploymentIDs, exists := deploymentsByTxID[tx.TransactionId]; exists {
			transaction.Deployments = deploymentIDs
		}

		// Add the transaction
		if err := u.manager.addTransactionInternal(transaction); err != nil {
			return fmt.Errorf("failed to add transaction %s: %w", transaction.ID, err)
		}

		// Update deployment transaction IDs
		for _, depID := range transaction.Deployments {
			if dep, exists := u.manager.deployments[depID]; exists {
				dep.TransactionID = transaction.ID
			}
		}
	}

	// Process Safe transactions
	for _, safeTx := range u.execution.SafeTransactions {
		safeTransaction := u.createSafeTransactionFromExecution(safeTx, u.execution.ChainID, now)

		// Map transaction IDs to registry IDs
		for _, txID := range safeTx.TransactionIDs {
			if tx := u.execution.GetTransactionByID(txID); tx != nil {
				registryTxID := u.manager.getRegistryTransactionID(tx)
				safeTransaction.TransactionIDs = append(safeTransaction.TransactionIDs, registryTxID)
			}
		}

		// Add the Safe transaction
		if existing, exists := u.manager.safeTransactions[safeTransaction.SafeTxHash]; exists {
			// Update existing Safe transaction
			// TODO: Merge confirmations and update status if needed
			existing.TransactionIDs = safeTransaction.TransactionIDs
			existing.Status = safeTransaction.Status
			if safeTx.ExecutionTxHash != nil {
				existing.ExecutionTxHash = safeTx.ExecutionTxHash.Hex()
			}
		} else {
			u.manager.safeTransactions[safeTransaction.SafeTxHash] = safeTransaction
		}
	}

	// Save the updated registry
	return u.manager.save()
}

// createDeploymentFromRecord creates a deployment from a deployment record
func (u *ScriptExecutionUpdater) createDeploymentFromRecord(
	record *parser.DeploymentRecord,
	commitHash string,
	timestamp time.Time,
) (*types.Deployment, error) {
	// Extract contract info from artifact and deployment record
	contractName := record.Contract.Name

	// Determine deployment type
	deploymentType := types.SingletonDeployment

	// Check if this is a library deployment from contract info
	if record.Contract != nil && record.Contract.IsLibrary {
		deploymentType = types.LibraryDeployment
	}

	// Check if this is a proxy based on proxy relationships
	var proxyInfo *types.ProxyInfo
	if proxyRel, isProxy := u.execution.GetProxyInfo(record.Address); isProxy {
		deploymentType = types.ProxyDeployment
		proxyInfo = &types.ProxyInfo{
			Type:           proxyRel.ProxyType,
			Implementation: proxyRel.Implementation.Hex(),
			History:        []types.ProxyUpgrade{},
		}
		if proxyRel.Admin != nil {
			proxyInfo.Admin = proxyRel.Admin.Hex()
		}
	}

	// Determine deployment method from create strategy
	method := mapCreateStrategy(record.Deployment.CreateStrategy)

	deployment := &types.Deployment{
		ID:           formatDeploymentID(u.namespace, u.execution.ChainID, contractName, record.Deployment.Label),
		Namespace:    u.namespace,
		ChainID:      u.execution.ChainID,
		ContractName: contractName,
		Label:        record.Deployment.Label,
		Address:      record.Address.Hex(),
		Type:         deploymentType,
		// TransactionID will be set later when we process transactions
		DeploymentStrategy: types.DeploymentStrategy{
			Method:          method,
			Salt:            fmt.Sprintf("0x%x", record.Deployment.Salt),
			InitCodeHash:    fmt.Sprintf("0x%x", record.Deployment.InitCodeHash),
			ConstructorArgs: fmt.Sprintf("0x%x", record.Deployment.ConstructorArgs),
			Factory:         "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed", // CreateX
			Entropy:         record.Deployment.Entropy,
		},
		ProxyInfo: proxyInfo,
		Artifact: types.ArtifactInfo{
			Path:            record.Contract.Path,
			BytecodeHash:    fmt.Sprintf("0x%x", record.Deployment.BytecodeHash),
			ScriptPath:      u.execution.Script.Path,
			GitCommit:       commitHash,
			CompilerVersion: u.manager.getCompilerVersion(record),
		},
		Verification: types.VerificationInfo{
			Status: types.VerificationStatusUnverified,
		},
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	return deployment, nil
}

// createTransactionFromExecution creates a transaction record from execution data
func (u *ScriptExecutionUpdater) createTransactionFromExecution(
	tx *parser.Transaction,
	chainID uint64,
	namespace string,
	timestamp time.Time,
) *types.Transaction {
	transaction := &types.Transaction{
		ID:          u.manager.getRegistryTransactionID(tx),
		ChainID:     chainID,
		Status:      tx.Status,
		Sender:      tx.Sender.Hex(),
		Environment: namespace,
		CreatedAt:   timestamp,
		Deployments: []string{}, // Will be populated by caller
		Operations:  []types.Operation{},
	}

	// Add execution details if available
	if tx.TxHash != nil {
		transaction.Hash = tx.TxHash.Hex()
	}
	if tx.BlockNumber != nil {
		transaction.BlockNumber = *tx.BlockNumber
	}

	// Add Safe context if this is a Safe transaction
	if tx.SafeTransaction != nil {
		transaction.SafeContext = &types.SafeContext{
			SafeAddress: tx.SafeTransaction.Safe.Hex(),
			SafeTxHash:  common.Hash(tx.SafeTransaction.SafeTxHash).Hex(),
		}
		if tx.SafeBatchIdx != nil {
			transaction.SafeContext.BatchIndex = *tx.SafeBatchIdx
		}
		// TODO: Get proposer address from Safe transaction data
		transaction.SafeContext.ProposerAddress = tx.SafeTransaction.Proposer.Hex()
	}

	// TODO: Build operations from transaction data
	// This would require analyzing the transaction input data

	return transaction
}

// createSafeTransactionFromExecution creates a Safe transaction from execution data
func (u *ScriptExecutionUpdater) createSafeTransactionFromExecution(
	safeTx *parser.SafeTransaction,
	chainID uint64,
	timestamp time.Time,
) *types.SafeTransaction {
	// Determine status based on whether the Safe transaction was executed directly
	status := types.TransactionStatusQueued
	if safeTx.Executed {
		status = types.TransactionStatusExecuted
	}
	
	safeTransaction := &types.SafeTransaction{
		SafeTxHash:     common.Hash(safeTx.SafeTxHash).Hex(),
		SafeAddress:    safeTx.Safe.Hex(),
		ChainID:        chainID,
		Status:         status,
		ProposedBy:     safeTx.Proposer.Hex(),
		ProposedAt:     timestamp,
		Transactions:   []types.SafeTxData{},
		TransactionIDs: []string{}, // Will be populated by caller
		Confirmations:  []types.Confirmation{},
		// TODO: Get nonce from Safe contract or transaction data
		Nonce: 0,
	}

	// Set execution hash if available
	if safeTx.ExecutionTxHash != nil {
		safeTransaction.ExecutionTxHash = safeTx.ExecutionTxHash.Hex()
	}

	// TODO: Build SafeTxData from transaction details
	// This would require access to the actual transaction data from the execution

	return safeTransaction
}

// Helper methods

// getRegistryTransactionID generates a registry transaction ID
func (m *Manager) getRegistryTransactionID(tx *parser.Transaction) string {
	if tx.TxHash != nil {
		return fmt.Sprintf("tx-%s", tx.TxHash.Hex())
	} else if tx.SafeTransaction != nil {
		return fmt.Sprintf("safe-%s-%d", common.Hash(tx.SafeTransaction.SafeTxHash).Hex(), *tx.SafeBatchIdx)
	} else {
		return fmt.Sprintf("tx-internal-%x", tx.TransactionId)
	}
}

// mapCreateStrategy maps create strategy string to deployment method
func mapCreateStrategy(strategy string) types.DeploymentMethod {
	switch strategy {
	case "CREATE":
		return types.DeploymentMethodCreate
	case "CREATE2":
		return types.DeploymentMethodCreate2
	case "CREATE3":
		return types.DeploymentMethodCreate3
	default:
		// Default to CREATE2 if not specified
		return types.DeploymentMethodCreate2
	}
}

// formatDeploymentID creates a deployment ID from components
func formatDeploymentID(namespace string, chainID uint64, contractName string, label string) string {
	if label != "" {
		return fmt.Sprintf("%s/%d/%s:%s", namespace, chainID, contractName, label)
	}
	return fmt.Sprintf("%s/%d/%s", namespace, chainID, contractName)
}

// getGitCommit returns the current git commit hash
func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getCompilerVersion extracts the compiler version from deployment record
func (m *Manager) getCompilerVersion(record *parser.DeploymentRecord) string {
	// Try to get from contract info first
	if record.Contract != nil && record.Contract.Artifact != nil {
		// Extract compiler version from artifact metadata
		if record.Contract.Artifact.Metadata.Compiler.Version != "" {
			return record.Contract.Artifact.Metadata.Compiler.Version
		}
	}

	// Fallback to unknown if not available
	return "unknown"
}

// addDeploymentInternal adds a deployment without locking (assumes lock is held)
func (m *Manager) addDeploymentInternal(deployment *types.Deployment) error {
	// Check if deployment already exists
	if _, exists := m.deployments[deployment.ID]; exists {
		return fmt.Errorf("deployment already exists: %s", deployment.ID)
	}

	// Add to registry
	m.deployments[deployment.ID] = deployment

	// Update indexes
	m.updateIndexesForDeployment(deployment)
	
	// Update solidity registry
	m.updateSolidityRegistry(deployment)

	return nil
}

// addTransactionInternal adds a transaction without locking (assumes lock is held)
func (m *Manager) addTransactionInternal(transaction *types.Transaction) error {
	// Check if transaction already exists
	if _, exists := m.transactions[transaction.ID]; exists {
		// TODO: Update existing transaction with new data if needed
		return nil
	}

	// Add to registry
	m.transactions[transaction.ID] = transaction

	return nil
}

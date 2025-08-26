package deployments

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// BuildChangesetFromRunResult analyzes the execution and prepares registry updates
func (f *FileRepository) BuildChangesetFromRunResult(ctx context.Context, execution *forge.HydratedRunResult) (*models.Changeset, error) {
	changeset := &models.Changeset{
		Create: models.ChangesetModels{
			Deployments:      []*models.Deployment{},
			Transactions:     []*models.Transaction{},
			SafeTransactions: []*models.SafeTransaction{},
		},
	}

	// Get current timestamp
	now := time.Now()

	// Get git commit hash
	commitHash := getGitCommit()

	// Track deployment IDs for linking to transactions
	deploymentsByTxID := make(map[[32]byte][]string)

	// Process deployments from the execution
	for _, dep := range execution.Deployments {
		deployment, err := f.createDeploymentFromRecord(dep, execution, commitHash, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create deployment: %w", err)
		}

		changeset.Create.Deployments = append(changeset.Create.Deployments, deployment)

		// Track which transaction this deployment belongs to
		if dep.TransactionID != [32]byte{} {
			deploymentsByTxID[dep.TransactionID] = append(deploymentsByTxID[dep.TransactionID], deployment.ID)
		}
	}

	// Process transactions
	for _, tx := range execution.Transactions {
		transaction := f.createTransactionFromExecution(tx, execution.ChainID, execution.Namespace, now)

		// Link deployments to transaction
		if deploymentIDs, exists := deploymentsByTxID[tx.TransactionId]; exists {
			transaction.Deployments = deploymentIDs
		}

		changeset.Create.Transactions = append(changeset.Create.Transactions, transaction)

		// Update deployment transaction IDs
		for _, deployment := range changeset.Create.Deployments {
			for _, depID := range transaction.Deployments {
				if deployment.ID == depID {
					deployment.TransactionID = transaction.ID
				}
			}
		}
	}

	// Process Safe transactions
	for _, safeTx := range execution.SafeTransactions {
		safeTransaction := f.createSafeTransactionFromExecution(safeTx, execution.ChainID, now)

		// Map transaction IDs to registry IDs
		for _, txID := range safeTx.TransactionIds {
			if tx := f.getTransactionByID(execution, txID); tx != nil {
				registryTxID := f.getRegistryTransactionID(tx)
				safeTransaction.TransactionIDs = append(safeTransaction.TransactionIDs, registryTxID)
			}
		}

		changeset.Create.SafeTransactions = append(changeset.Create.SafeTransactions, safeTransaction)
	}

	return changeset, nil
}

// ApplyChangeset applies all updates in a single transaction with one lock
func (m *FileRepository) ApplyChangeset(ctx context.Context, changeset *models.Changeset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Apply deletions first
	if changeset.Delete.Count() > 0 {
		// Delete deployments
		for _, deployment := range changeset.Delete.Deployments {
			delete(m.deployments, deployment.ID)
		}

		// Delete transactions
		for _, tx := range changeset.Delete.Transactions {
			delete(m.transactions, tx.ID)
		}

		// Delete safe transactions
		for _, tx := range changeset.Delete.SafeTransactions {
			delete(m.safeTransactions, tx.SafeTxHash)
		}
	}

	// Apply updates
	if changeset.Update.Count() > 0 {
		// Update deployments
		for _, deployment := range changeset.Update.Deployments {
			if existing, exists := m.deployments[deployment.ID]; exists {
				// Preserve original creation timestamp
				deployment.CreatedAt = existing.CreatedAt
				deployment.UpdatedAt = now
				m.deployments[deployment.ID] = deployment
			}
		}

		// Update transactions
		for _, tx := range changeset.Update.Transactions {
			if existing, exists := m.transactions[tx.ID]; exists {
				// Preserve original creation timestamp
				tx.CreatedAt = existing.CreatedAt
				m.transactions[tx.ID] = tx
			}
		}

		// Update safe transactions
		for _, tx := range changeset.Update.SafeTransactions {
			if existing, exists := m.safeTransactions[tx.SafeTxHash]; exists {
				// Preserve original creation timestamp
				tx.ProposedAt = existing.ProposedAt
				m.safeTransactions[tx.SafeTxHash] = tx
			}
		}
	}

	// Apply creations
	if changeset.Create.Count() > 0 {
		// Create deployments
		for _, deployment := range changeset.Create.Deployments {
			// Set timestamps
			if deployment.CreatedAt.IsZero() {
				deployment.CreatedAt = now
			}
			deployment.UpdatedAt = now

			// Save deployment
			m.deployments[deployment.ID] = deployment
		}

		// Create transactions
		for _, tx := range changeset.Create.Transactions {
			// Set timestamp
			if tx.CreatedAt.IsZero() {
				tx.CreatedAt = now
			}

			// Save transaction
			m.transactions[tx.ID] = tx
		}

		// Create safe transactions
		for _, tx := range changeset.Create.SafeTransactions {
			// Set timestamp
			if tx.ProposedAt.IsZero() {
				tx.ProposedAt = now
			}

			// Save transaction
			m.safeTransactions[tx.SafeTxHash] = tx
		}
	}

	// Rebuild lookups once for all changes
	m.rebuildLookups()

	// Save all files once
	return m.save()
}

// createDeploymentFromRecord creates a deployment from a deployment record
func (f *FileRepository) createDeploymentFromRecord(
	record *forge.Deployment,
	execution *forge.HydratedRunResult,
	commitHash string,
	timestamp time.Time,
) (*models.Deployment, error) {
	// Extract contract info from deployment record
	var contractName string
	if record.Contract != nil {
		contractName = record.Contract.Name
	} else if record.Event != nil && record.Event.Artifact != "" {
		// Fallback: try to extract contract name from artifact path
		// Format is usually "path/to/Contract.sol:ContractName"
		parts := strings.Split(record.Event.Artifact, ":")
		if len(parts) == 2 {
			contractName = parts[1]
		} else {
			return nil, fmt.Errorf("deployment record has nil Contract field and cannot extract name from artifact: %s", record.Event.Artifact)
		}
	} else {
		return nil, fmt.Errorf("deployment record has nil Contract field and no artifact information for address %s", record.Address)
	}

	// Determine deployment type
	deploymentType := models.SingletonDeployment

	// Check if this is a library deployment from contract info
	if record.Contract != nil && record.Contract.IsLibrary() {
		deploymentType = models.LibraryDeployment
	}

	// Check if this is a proxy based on proxy relationships
	var proxyInfo *models.ProxyInfo
	if proxyRel, exists := execution.ProxyRelationships[record.Address]; exists {
		f.log.Debug("Recording proxy", "address", record.Address, "info", proxyRel)
		deploymentType = models.ProxyDeployment
		proxyInfo = &models.ProxyInfo{
			Type:           string(proxyRel.ProxyType),
			Implementation: proxyRel.ImplementationAddress.Hex(),
			History:        []models.ProxyUpgrade{},
		}
		if proxyRel.AdminAddress != nil {
			proxyInfo.Admin = proxyRel.AdminAddress.Hex()
		}
	}

	// Determine deployment method from create strategy
	method := models.DeploymentMethodCreate2 // Default
	if record.Event != nil {
		method = mapCreateStrategy(record.Event.CreateStrategy)
	}

	var label string
	var entropy string
	var salt [32]byte
	var initCodeHash [32]byte
	var constructorArgs []byte
	var bytecodeHash [32]byte

	if record.Event != nil {
		label = record.Event.Label
		entropy = record.Event.Entropy
		salt = record.Event.Salt
		initCodeHash = record.Event.InitCodeHash
		constructorArgs = record.Event.ConstructorArgs
		bytecodeHash = record.Event.BytecodeHash
	}

	deployment := &models.Deployment{
		ID:           formatDeploymentID(execution.Namespace, execution.ChainID, contractName, label),
		Namespace:    execution.Namespace,
		ChainID:      execution.ChainID,
		ContractName: contractName,
		Label:        label,
		Address:      record.Address.Hex(),
		Type:         deploymentType,
		// TransactionID will be set later when we process transactions
		DeploymentStrategy: models.DeploymentStrategy{
			Method:          method,
			Salt:            fmt.Sprintf("0x%x", salt),
			InitCodeHash:    fmt.Sprintf("0x%x", initCodeHash),
			ConstructorArgs: fmt.Sprintf("0x%x", constructorArgs),
			Factory:         "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed", // CreateX
			Entropy:         entropy,
		},
		ProxyInfo: proxyInfo,
		Artifact: models.ArtifactInfo{
			Path:            getContractPath(record.Contract),
			BytecodeHash:    fmt.Sprintf("0x%x", bytecodeHash),
			ScriptPath:      execution.Script.Path,
			GitCommit:       commitHash,
			CompilerVersion: f.getCompilerVersion(record),
		},
		Verification: models.VerificationInfo{
			Status: models.VerificationStatusUnverified,
		},
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	return deployment, nil
}

// createTransactionFromExecution creates a transaction record from execution data
func (f *FileRepository) createTransactionFromExecution(
	tx *forge.Transaction,
	chainID uint64,
	namespace string,
	timestamp time.Time,
) *models.Transaction {
	transaction := &models.Transaction{
		ID:          f.getRegistryTransactionID(tx),
		ChainID:     chainID,
		Status:      tx.Status,
		Sender:      tx.Sender.Hex(),
		Environment: namespace,
		CreatedAt:   timestamp,
		Deployments: []string{}, // Will be populated by caller
		Operations:  []models.Operation{},
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
		transaction.SafeContext = &models.SafeContext{
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
func (f *FileRepository) createSafeTransactionFromExecution(
	safeTx *forge.SafeTransaction,
	chainID uint64,
	timestamp time.Time,
) *models.SafeTransaction {
	// Determine status based on whether the Safe transaction was executed directly
	status := models.TransactionStatusQueued
	if safeTx.Executed {
		status = models.TransactionStatusExecuted
	}

	safeTransaction := &models.SafeTransaction{
		SafeTxHash:     common.Hash(safeTx.SafeTxHash).Hex(),
		SafeAddress:    safeTx.Safe.Hex(),
		ChainID:        chainID,
		Status:         status,
		ProposedBy:     safeTx.Proposer.Hex(),
		ProposedAt:     timestamp,
		Transactions:   []models.SafeTxData{},
		TransactionIDs: []string{}, // Will be populated by caller
		Confirmations:  []models.Confirmation{},
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
func (f *FileRepository) getRegistryTransactionID(tx *forge.Transaction) string {
	if tx.TxHash != nil {
		return fmt.Sprintf("tx-%s", tx.TxHash.Hex())
	} else if tx.SafeTransaction != nil {
		return fmt.Sprintf("safe-%s-%d", common.Hash(tx.SafeTransaction.SafeTxHash).Hex(), *tx.SafeBatchIdx)
	} else {
		return fmt.Sprintf("tx-internal-%x", tx.TransactionId)
	}
}

// getTransactionByID finds a transaction by ID in the execution result
func (f *FileRepository) getTransactionByID(execution *forge.HydratedRunResult, txID [32]byte) *forge.Transaction {
	for _, tx := range execution.Transactions {
		if tx.TransactionId == txID {
			return tx
		}
	}
	return nil
}

// mapCreateStrategy maps create strategy string to deployment method
func mapCreateStrategy(strategy string) models.DeploymentMethod {
	switch strategy {
	case "CREATE":
		return models.DeploymentMethodCreate
	case "CREATE2":
		return models.DeploymentMethodCreate2
	case "CREATE3":
		return models.DeploymentMethodCreate3
	default:
		// Default to CREATE2 if not specified
		return models.DeploymentMethodCreate2
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
func (f *FileRepository) getCompilerVersion(record *forge.Deployment) string {
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

// getContractPath gets the contract path from contract info
func getContractPath(contract *models.Contract) string {
	if contract != nil {
		return contract.Path
	}
	return ""
}

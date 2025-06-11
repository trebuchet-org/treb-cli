package registry

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// UpdateFromScriptExecution updates the registry from a script execution
func (m *Manager) UpdateFromScriptExecution(
	execution *parser.ScriptExecution,
	namespace string,
	networkName string,
	scriptPath string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the current timestamp for all operations
	now := time.Now()

	// Get git commit hash
	commitHash := getGitCommit()

	// Track deployment IDs for linking to transactions
	deploymentsByTxID := make(map[[32]byte][]string)

	// First, process all deployments
	for _, dep := range execution.Deployments {
		deployment, err := m.createDeploymentFromRecord(dep, namespace, execution, scriptPath, commitHash, now)
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}

		// Add the deployment
		if err := m.addDeploymentInternal(deployment); err != nil {
			return fmt.Errorf("failed to add deployment %s: %w", deployment.ID, err)
		}

		// Track which transaction this deployment belongs to
		deploymentsByTxID[dep.TransactionID] = append(deploymentsByTxID[dep.TransactionID], deployment.ID)
	}

	// Process transactions
	for _, tx := range execution.Transactions {
		transaction := m.createTransactionFromExecution(tx, execution.ChainID, namespace, now)

		// Link deployments to transaction
		if deploymentIDs, exists := deploymentsByTxID[tx.TransactionId]; exists {
			transaction.Deployments = deploymentIDs
		}

		// Add the transaction
		if err := m.addTransactionInternal(transaction); err != nil {
			return fmt.Errorf("failed to add transaction %s: %w", transaction.ID, err)
		}

		// Update deployment transaction IDs
		for _, depID := range transaction.Deployments {
			if dep, exists := m.deployments[depID]; exists {
				dep.TransactionID = transaction.ID
			}
		}
	}

	// Process Safe transactions
	for _, safeTx := range execution.SafeTransactions {
		safeTransaction := m.createSafeTransactionFromExecution(safeTx, execution.ChainID, now)

		// Map transaction IDs to registry IDs
		for _, txID := range safeTx.TransactionIDs {
			if tx := execution.GetTransactionByID(txID); tx != nil {
				registryTxID := m.getRegistryTransactionID(tx)
				safeTransaction.TransactionIDs = append(safeTransaction.TransactionIDs, registryTxID)
			}
		}

		// Add the Safe transaction
		if existing, exists := m.safeTransactions[safeTransaction.SafeTxHash]; exists {
			// Update existing Safe transaction
			// TODO: Merge confirmations and update status if needed
			existing.TransactionIDs = safeTransaction.TransactionIDs
			existing.Status = safeTransaction.Status
		} else {
			m.safeTransactions[safeTransaction.SafeTxHash] = safeTransaction
		}
	}

	// Save the updated registry
	return m.save()
}

// createDeploymentFromRecord creates a deployment from a deployment record
func (m *Manager) createDeploymentFromRecord(
	record *parser.DeploymentRecord,
	namespace string,
	execution *parser.ScriptExecution,
	scriptPath string,
	commitHash string,
	timestamp time.Time,
) (*types.Deployment, error) {
	// Extract contract info from artifact
	contractName, artifactPath := m.parseArtifact(record.Deployment.Artifact)

	// Determine deployment type
	deploymentType := types.SingletonDeployment

	// Check if this is a proxy based on proxy relationships
	var proxyInfo *types.ProxyInfo
	if proxyRel, isProxy := execution.GetProxyInfo(record.Address); isProxy {
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

	// TODO: Check if this is a library deployment
	// This would require checking contract metadata or bytecode analysis

	// Determine deployment method from create strategy
	method := mapCreateStrategy(record.Deployment.CreateStrategy)

	deployment := &types.Deployment{
		ID:           formatDeploymentID(namespace, execution.ChainID, contractName, record.Deployment.Label),
		Namespace:    namespace,
		ChainID:      execution.ChainID,
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
			Path:         artifactPath,
			BytecodeHash: fmt.Sprintf("0x%x", record.Deployment.BytecodeHash),
			ScriptPath:   extractScriptName(scriptPath),
			GitCommit:    commitHash,
			// TODO: Get compiler version from contract metadata
			CompilerVersion: "unknown",
		},
		Verification: types.VerificationInfo{
			Status: types.VerificationStatusUnverified,
		},
		Tags:      []string{},
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	return deployment, nil
}

// createTransactionFromExecution creates a transaction record from execution data
func (m *Manager) createTransactionFromExecution(
	tx *parser.Transaction,
	chainID uint64,
	namespace string,
	timestamp time.Time,
) *types.Transaction {
	transaction := &types.Transaction{
		ID:          m.getRegistryTransactionID(tx),
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
	if tx.SafeTxHash != nil && tx.SafeAddress != nil {
		transaction.SafeContext = &types.SafeContext{
			SafeAddress: tx.SafeAddress.Hex(),
			SafeTxHash:  tx.SafeTxHash.Hex(),
		}
		if tx.SafeBatchIdx != nil {
			transaction.SafeContext.BatchIndex = *tx.SafeBatchIdx
		}
		// TODO: Get proposer address from Safe transaction data
		transaction.SafeContext.ProposerAddress = ""
	}

	// TODO: Build operations from transaction data
	// This would require analyzing the transaction input data

	return transaction
}

// createSafeTransactionFromExecution creates a Safe transaction from execution data
func (m *Manager) createSafeTransactionFromExecution(
	safeTx *parser.SafeTransaction,
	chainID uint64,
	timestamp time.Time,
) *types.SafeTransaction {
	safeTransaction := &types.SafeTransaction{
		SafeTxHash:     common.Hash(safeTx.SafeTxHash).Hex(),
		SafeAddress:    safeTx.Safe.Hex(),
		ChainID:        chainID,
		Status:         types.TransactionStatusQueued,
		ProposedBy:     safeTx.Proposer.Hex(),
		ProposedAt:     timestamp,
		Transactions:   []types.SafeTxData{},
		TransactionIDs: []string{}, // Will be populated by caller
		Confirmations:  []types.Confirmation{},
		// TODO: Get nonce from Safe contract or transaction data
		Nonce: 0,
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
	} else if tx.SafeTxHash != nil && tx.SafeBatchIdx != nil {
		return fmt.Sprintf("safe-%s-%d", tx.SafeTxHash.Hex(), *tx.SafeBatchIdx)
	} else {
		return fmt.Sprintf("tx-internal-%x", tx.TransactionId)
	}
}

// parseArtifact extracts contract name and path from artifact string
func (m *Manager) parseArtifact(artifact string) (contractName string, artifactPath string) {
	// Handle formats like:
	// - "src/Counter.sol:Counter" -> ("Counter", "src/Counter.sol:Counter")
	// - "Counter" -> ("Counter", "Counter")
	// - "" -> ("Unknown", "Unknown")

	if artifact == "" {
		return "Unknown", "Unknown"
	}

	// Check if it's in the format "path:contractName"
	if idx := strings.LastIndex(artifact, ":"); idx != -1 {
		contractName = artifact[idx+1:]
		artifactPath = artifact
	} else {
		// If no colon, assume the entire string is the contract name
		contractName = artifact
		artifactPath = artifact
	}

	return contractName, artifactPath
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

// extractScriptName extracts the script name from a script path
func extractScriptName(scriptPath string) string {
	// Get the base filename
	filename := filepath.Base(scriptPath)

	// Extract script name by removing .s.sol extension
	if strings.HasSuffix(filename, ".s.sol") {
		scriptName := strings.TrimSuffix(filename, ".s.sol")
		// Return in format "filename:scriptName"
		return fmt.Sprintf("%s:%s", filename, scriptName)
	}

	return filename
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

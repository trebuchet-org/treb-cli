package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// RegistryUpdater handles registry updates without pkg dependencies
type RegistryUpdater struct {
	deploymentStore  usecase.DeploymentStore
	transactionStore usecase.TransactionStore
}

// NewRegistryUpdater creates a new internal registry updater
func NewRegistryUpdater(
	deploymentStore usecase.DeploymentStore,
	transactionStore usecase.TransactionStore,
) *RegistryUpdater {
	return &RegistryUpdater{
		deploymentStore:  deploymentStore,
		transactionStore: transactionStore,
	}
}

// PrepareUpdates analyzes the execution and prepares registry updates
func (u *RegistryUpdater) PrepareUpdates(
	ctx context.Context,
	execution *domain.ScriptExecution,
) (*usecase.RegistryChanges, error) {
	changes := &usecase.RegistryChanges{
		Deployments:  []*domain.Deployment{},
		Transactions: []*domain.Transaction{},
	}

	// Get current timestamp
	now := time.Now()

	// Get git commit hash
	commitHash := getGitCommit()

	// Track deployment IDs for linking to transactions
	deploymentsByTxHash := make(map[string][]string)

	// Process deployments from the execution
	for _, dep := range execution.Deployments {
		deployment, err := u.createDeploymentFromScriptDeployment(dep, execution, commitHash, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create deployment: %w", err)
		}

		changes.Deployments = append(changes.Deployments, deployment)

		// Track which transaction this deployment belongs to
		if dep.TransactionID != [32]byte{} {
			txID := fmt.Sprintf("%x", dep.TransactionID)
			deploymentsByTxHash[txID] = append(deploymentsByTxHash[txID], deployment.ID)
		}
	}

	// Process transactions
	for _, tx := range execution.Transactions {
		transaction := u.createTransactionFromScriptTransaction(&tx, execution.Network.ChainID, execution.Namespace, now)

		// Link deployments to transaction
		txID := fmt.Sprintf("%x", tx.TransactionID)
		if deploymentIDs, exists := deploymentsByTxHash[txID]; exists {
			transaction.Deployments = deploymentIDs
		}

		changes.Transactions = append(changes.Transactions, transaction)

		// Update deployment transaction IDs
		for _, deployment := range changes.Deployments {
			for _, depID := range transaction.Deployments {
				if deployment.ID == depID {
					deployment.TransactionID = transaction.ID
				}
			}
		}
	}

	// Set change counts
	changes.AddedCount = len(changes.Deployments)
	changes.HasChanges = changes.AddedCount > 0

	return changes, nil
}

// ApplyUpdates applies the prepared changes to the registry
func (u *RegistryUpdater) ApplyUpdates(ctx context.Context, changes *usecase.RegistryChanges) error {
	// Save deployments
	for _, deployment := range changes.Deployments {
		if err := u.deploymentStore.SaveDeployment(ctx, deployment); err != nil {
			return fmt.Errorf("failed to save deployment %s: %w", deployment.ID, err)
		}
	}

	// Save transactions
	for _, transaction := range changes.Transactions {
		if err := u.transactionStore.SaveTransaction(ctx, transaction); err != nil {
			return fmt.Errorf("failed to save transaction %s: %w", transaction.ID, err)
		}
	}

	return nil
}

// HasChanges returns true if there are any changes to apply
func (u *RegistryUpdater) HasChanges(changes *usecase.RegistryChanges) bool {
	return changes != nil && changes.HasChanges
}

// createDeploymentFromScriptDeployment creates a deployment from a ScriptDeployment
func (u *RegistryUpdater) createDeploymentFromScriptDeployment(
	dep domain.ScriptDeployment,
	execution *domain.ScriptExecution,
	commitHash string,
	timestamp time.Time,
) (*domain.Deployment, error) {
	// Generate deployment ID
	id := fmt.Sprintf("%s/%d/%s", execution.Namespace, execution.Network.ChainID, dep.ContractName)
	if dep.Label != "" {
		id = fmt.Sprintf("%s:%s", id, dep.Label)
	}

	deployment := &domain.Deployment{
		ID:           id,
		Namespace:    execution.Namespace,
		ChainID:      execution.Network.ChainID,
		ContractName: dep.ContractName,
		Label:        dep.Label,
		Address:      dep.Address,
		Type:         dep.DeploymentType,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
		Artifact: domain.ArtifactInfo{
			Path:      dep.Artifact,
			GitCommit: commitHash,
		},
		Verification: domain.VerificationInfo{
			Status: domain.VerificationStatusUnverified,
		},
	}

	// Set deployment strategy
	deployment.DeploymentStrategy = domain.DeploymentStrategy{
		Method:       mapDeploymentMethod(dep.CreateStrategy),
		Salt:         fmt.Sprintf("0x%x", dep.Salt),
		InitCodeHash: fmt.Sprintf("0x%x", dep.InitCodeHash),
		Factory:      "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed", // CreateX
	}

	// Set proxy info if available
	if dep.ProxyInfo != nil {
		deployment.ProxyInfo = dep.ProxyInfo
	}

	return deployment, nil
}

// createTransactionFromScriptTransaction creates a transaction record
func (u *RegistryUpdater) createTransactionFromScriptTransaction(
	tx *domain.ScriptTransaction,
	chainID uint64,
	namespace string,
	timestamp time.Time,
) *domain.Transaction {
	// Use tx hash if available, otherwise use transaction ID
	id := ""
	hash := ""
	if tx.TxHash != nil && *tx.TxHash != "" {
		hash = *tx.TxHash
		id = fmt.Sprintf("tx-%s", hash)
	} else {
		id = fmt.Sprintf("tx-%x", tx.TransactionID)
	}

	transaction := &domain.Transaction{
		ID:          id,
		ChainID:     chainID,
		Hash:        hash,
		Status:      mapTransactionStatus(tx.Status),
		Sender:      tx.From,
		Nonce:       tx.Nonce,
		Environment: namespace,
		CreatedAt:   timestamp,
	}

	// Set block number if available
	if tx.BlockNumber != nil {
		transaction.BlockNumber = *tx.BlockNumber
	}

	return transaction
}

// Helper functions
func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func mapDeploymentType(deploymentType string) domain.DeploymentType {
	switch strings.ToUpper(deploymentType) {
	case "PROXY":
		return domain.ProxyDeployment
	case "LIBRARY":
		return domain.LibraryDeployment
	case "SINGLETON":
		return domain.SingletonDeployment
	default:
		return domain.UnknownDeployment
	}
}

func mapDeploymentMethod(method string) domain.DeploymentMethod {
	switch strings.ToUpper(method) {
	case "CREATE2":
		return domain.DeploymentMethodCreate2
	case "CREATE3":
		return domain.DeploymentMethodCreate3
	default:
		return domain.DeploymentMethodCreate
	}
}

func mapTransactionStatus(status domain.TransactionStatus) domain.TransactionStatus {
	// Already a domain type, just return it
	return status
}

var _ usecase.RegistryUpdater = (&RegistryUpdater{})

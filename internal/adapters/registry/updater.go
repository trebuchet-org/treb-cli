package registry

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/bindings"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// UpdaterAdapter adapts the existing registry updater to the RegistryUpdater interface
type UpdaterAdapter struct {
	manager *registry.Manager
}

// NewUpdaterAdapter creates a new registry updater adapter
func NewUpdaterAdapter(projectPath string) (*UpdaterAdapter, error) {
	manager, err := registry.NewManager(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}

	return &UpdaterAdapter{
		manager: manager,
	}, nil
}

// PrepareUpdates analyzes the execution and prepares registry updates
func (a *UpdaterAdapter) PrepareUpdates(
	ctx context.Context,
	execution *domain.ScriptExecution,
	namespace string,
	network string,
) (*usecase.RegistryChanges, error) {
	// Convert domain execution to v1 script execution for the updater
	v1Execution := a.convertToV1Execution(execution)

	// Create the updater
	updater := a.manager.NewScriptExecutionUpdater(v1Execution, namespace, network, execution.ScriptPath)

	// Check if there are changes
	if !updater.HasChanges() {
		return &usecase.RegistryChanges{
			HasChanges: false,
		}, nil
	}

	// Prepare the changes
	changes := &usecase.RegistryChanges{
		HasChanges: true,
		AddedCount: len(execution.Deployments),
	}

	// Convert deployments to domain deployments for the result
	for _, scriptDep := range execution.Deployments {
		dep := &domain.Deployment{
			Namespace:    namespace,
			ChainID:      execution.ChainID,
			ContractName: scriptDep.ContractName,
			Label:        scriptDep.Label,
			Address:      scriptDep.Address,
			Type:         scriptDep.DeploymentType,
			ProxyInfo:    scriptDep.ProxyInfo,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Set deployment strategy
		dep.DeploymentStrategy = domain.DeploymentStrategy{
			Method:       mapCreateStrategy(scriptDep.CreateStrategy),
			Salt:         fmt.Sprintf("0x%x", scriptDep.Salt),
			InitCodeHash: fmt.Sprintf("0x%x", scriptDep.InitCodeHash),
			Factory:      "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed", // CreateX
		}

		changes.Deployments = append(changes.Deployments, dep)
	}

	return changes, nil
}

// ApplyUpdates applies the prepared changes to the registry
func (a *UpdaterAdapter) ApplyUpdates(ctx context.Context, changes *usecase.RegistryChanges) error {
	// The v1 updater handles everything in Write(), so we need to
	// reconstruct the execution and call Write() directly
	
	// For now, this is a simplified implementation
	// In a real implementation, we'd need to store the updater instance
	// or reconstruct it from the changes
	
	return fmt.Errorf("apply updates not implemented - use v1 Write() directly")
}

// HasChanges returns true if there are any changes to apply
func (a *UpdaterAdapter) HasChanges(changes *usecase.RegistryChanges) bool {
	return changes != nil && changes.HasChanges
}

// convertToV1Execution converts domain execution to v1 script execution
func (a *UpdaterAdapter) convertToV1Execution(execution *domain.ScriptExecution) *parser.ScriptExecution {
	v1Execution := &parser.ScriptExecution{
		Network:       execution.Network,
		ChainID:       execution.ChainID,
		Success:       execution.Success,
		BroadcastPath: execution.BroadcastPath,
		Logs:          execution.Logs,
		Deployments:   make([]*parser.DeploymentRecord, 0),
		Transactions:  make([]*parser.Transaction, 0),
	}

	// Convert deployments
	for _, dep := range execution.Deployments {
		v1Dep := &parser.DeploymentRecord{
			TransactionID: dep.TransactionID,
			Address:       common.HexToAddress(dep.Address),
			Deployer:      common.HexToAddress(dep.Deployer),
			Deployment: &bindings.ITrebEventsDeploymentDetails{
				Artifact:        dep.Artifact,
				Label:           dep.Label,
				CreateStrategy:  dep.CreateStrategy,
				Salt:            dep.Salt,
				InitCodeHash:    dep.InitCodeHash,
				ConstructorArgs: dep.ConstructorArgs,
				BytecodeHash:    dep.BytecodeHash,
			},
		}
		v1Execution.Deployments = append(v1Execution.Deployments, v1Dep)
	}

	// Convert transactions
	for _, tx := range execution.Transactions {
		v1Tx := &parser.Transaction{
			SimulatedTransaction: bindings.SimulatedTransaction{
				TransactionId: tx.TransactionID,
				SenderId:      [32]byte{}, // TODO: Add sender ID to domain model
				Sender:        common.HexToAddress(tx.Sender),
				ReturnData:    []byte{},   // TODO: Add return data to domain model
				Transaction: bindings.Transaction{
					To:    common.HexToAddress(tx.To),
					Data:  tx.Data,
					Value: big.NewInt(0), // TODO: Parse value from string
				},
			},
			Status: types.TransactionStatus(tx.Status),
		}

		if tx.TxHash != nil {
			hash := common.HexToHash(*tx.TxHash)
			v1Tx.TxHash = &hash
		}
		v1Tx.BlockNumber = tx.BlockNumber
		v1Tx.GasUsed = tx.GasUsed

		// Convert Safe transaction if present
		if tx.SafeTransaction != nil {
			v1Tx.SafeTransaction = &parser.SafeTransaction{
				SafeTxHash: tx.SafeTransaction.SafeTxHash,
				Safe:       common.HexToAddress(tx.SafeTransaction.SafeAddress),
				Proposer:   common.HexToAddress(tx.SafeTransaction.Proposer),
				Executed:   tx.SafeTransaction.Executed,
			}
			if tx.SafeTransaction.ExecutionTxHash != nil {
				hash := common.HexToHash(*tx.SafeTransaction.ExecutionTxHash)
				v1Tx.SafeTransaction.ExecutionTxHash = &hash
			}
			v1Tx.SafeBatchIdx = tx.SafeTransaction.BatchIndex
		}

		v1Execution.Transactions = append(v1Execution.Transactions, v1Tx)
	}

	return v1Execution
}

// mapCreateStrategy maps create strategy string to deployment method
func mapCreateStrategy(strategy string) domain.DeploymentMethod {
	switch strategy {
	case "CREATE":
		return domain.DeploymentMethodCreate
	case "CREATE2":
		return domain.DeploymentMethodCreate2
	case "CREATE3":
		return domain.DeploymentMethodCreate3
	default:
		return domain.DeploymentMethodCreate2
	}
}

// DirectUpdaterAdapter provides direct access to the v1 updater for compatibility
type DirectUpdaterAdapter struct {
	manager *registry.Manager
}

// NewDirectUpdaterAdapter creates a new direct updater adapter
func NewDirectUpdaterAdapter(projectPath string) (*DirectUpdaterAdapter, error) {
	manager, err := registry.NewManager(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry manager: %w", err)
	}

	return &DirectUpdaterAdapter{
		manager: manager,
	}, nil
}

// UpdateFromV1Execution updates the registry using v1 execution directly
func (a *DirectUpdaterAdapter) UpdateFromV1Execution(
	v1Execution *parser.ScriptExecution,
	namespace string,
	network string,
	scriptPath string,
) error {
	updater := a.manager.NewScriptExecutionUpdater(v1Execution, namespace, network, scriptPath)
	if updater.HasChanges() {
		return updater.Write()
	}
	return nil
}
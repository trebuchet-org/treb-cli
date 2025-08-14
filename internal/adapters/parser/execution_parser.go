package parser

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script/parser"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ExecutionParserAdapter adapts the existing script parser to the ExecutionParser interface
type ExecutionParserAdapter struct {
	parser           *parser.Parser
	contractsIndexer *contracts.Indexer
}

// NewExecutionParserAdapter creates a new execution parser adapter
func NewExecutionParserAdapter(contractsIndexer *contracts.Indexer) *ExecutionParserAdapter {
	return &ExecutionParserAdapter{
		parser:           parser.NewParser(contractsIndexer),
		contractsIndexer: contractsIndexer,
	}
}

// ParseExecution parses the script output into a structured execution result
func (a *ExecutionParserAdapter) ParseExecution(
	ctx context.Context,
	output *usecase.ScriptExecutionOutput,
	network string,
	chainID uint64,
) (*domain.ScriptExecution, error) {
	// Convert output to v1 forge script result
	forgeResult := &forge.ScriptResult{
		Success:       output.Success,
		RawOutput:     output.RawOutput,
		BroadcastPath: output.BroadcastPath,
	}

	// Type assert the parsed output
	if output.ParsedOutput != nil {
		if parsed, ok := output.ParsedOutput.(*forge.ParsedOutput); ok {
			forgeResult.ParsedOutput = parsed
		}
	}

	// Parse using v1 parser
	v1Execution, err := a.parser.Parse(forgeResult, network, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script execution: %w", err)
	}

	// Convert to domain execution
	execution := &domain.ScriptExecution{
		Network:       network,
		ChainID:       chainID,
		Success:       v1Execution.Success,
		BroadcastPath: v1Execution.BroadcastPath,
		Logs:          v1Execution.Logs,
	}

	// Convert transactions
	for _, v1Tx := range v1Execution.Transactions {
		tx := domain.ScriptTransaction{
			TransactionID: v1Tx.TransactionId,
			Sender:        v1Tx.Sender.Hex(),
			To:            v1Tx.Transaction.To.Hex(),
			Value:         v1Tx.Transaction.Value.String(),
			Data:          v1Tx.Transaction.Data,
			Status:        domain.TransactionStatus(v1Tx.Status),
		}

		if v1Tx.TxHash != nil {
			hash := v1Tx.TxHash.Hex()
			tx.TxHash = &hash
		}
		tx.BlockNumber = v1Tx.BlockNumber
		tx.GasUsed = v1Tx.GasUsed

		// Convert Safe transaction info if present
		if v1Tx.SafeTransaction != nil {
			tx.SafeTransaction = &domain.SafeTransactionInfo{
				SafeAddress:     v1Tx.SafeTransaction.Safe.Hex(),
				SafeTxHash:      v1Tx.SafeTransaction.SafeTxHash,
				Proposer:        v1Tx.SafeTransaction.Proposer.Hex(),
				Executed:        v1Tx.SafeTransaction.Executed,
				BatchIndex:      v1Tx.SafeBatchIdx,
			}
			if v1Tx.SafeTransaction.ExecutionTxHash != nil {
				hash := v1Tx.SafeTransaction.ExecutionTxHash.Hex()
				tx.SafeTransaction.ExecutionTxHash = &hash
			}
		}

		execution.Transactions = append(execution.Transactions, tx)
	}

	// Convert deployments
	for _, v1Dep := range v1Execution.Deployments {
		dep := domain.ScriptDeployment{
			TransactionID:   v1Dep.TransactionID,
			Address:         v1Dep.Address.Hex(),
			ContractName:    extractContractName(v1Dep.Deployment.Artifact),
			Artifact:        v1Dep.Deployment.Artifact,
			Label:           v1Dep.Deployment.Label,
			Deployer:        v1Dep.Deployer.Hex(),
			CreateStrategy:  v1Dep.Deployment.CreateStrategy,
			Salt:            v1Dep.Deployment.Salt,
			InitCodeHash:    v1Dep.Deployment.InitCodeHash,
			ConstructorArgs: v1Dep.Deployment.ConstructorArgs,
			BytecodeHash:    v1Dep.Deployment.BytecodeHash,
		}

		// Check if it's a proxy
		if proxyInfo, isProxy := v1Execution.GetProxyInfo(v1Dep.Address); isProxy {
			dep.IsProxy = true
			dep.ProxyInfo = &domain.ProxyInfo{
				Type:           proxyInfo.ProxyType,
				Implementation: proxyInfo.Implementation.Hex(),
			}
			if proxyInfo.Admin != nil {
				admin := proxyInfo.Admin.Hex()
				dep.ProxyInfo.Admin = admin
			}
		}

		// Determine deployment type
		if dep.IsProxy {
			dep.DeploymentType = domain.ProxyDeployment
		} else if v1Dep.Contract != nil && v1Dep.Contract.IsLibrary {
			dep.DeploymentType = domain.LibraryDeployment
		} else {
			dep.DeploymentType = domain.SingletonDeployment
		}

		execution.Deployments = append(execution.Deployments, dep)
	}

	// TODO: Parse stages from forge output if available
	// Currently forge doesn't provide stage information in parsed output

	return execution, nil
}

// EnrichFromBroadcast enriches execution data from broadcast files
func (a *ExecutionParserAdapter) EnrichFromBroadcast(
	ctx context.Context,
	execution *domain.ScriptExecution,
	broadcastPath string,
) error {
	// The v1 parser already handles broadcast enrichment during Parse
	// This is a no-op for the adapter since enrichment already happened
	return nil
}

// extractContractName is a helper to extract contract name from artifact path
func extractContractName(artifact string) string {
	// This logic should match what's in the display package
	if idx := len(artifact) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if artifact[i] == ':' {
				return artifact[i+1:]
			}
			if artifact[i] == '/' {
				name := artifact[i+1:]
				// Remove .sol extension if present
				if len(name) > 4 && name[len(name)-4:] == ".sol" {
					return name[:len(name)-4]
				}
				return name
			}
		}
	}
	return artifact
}
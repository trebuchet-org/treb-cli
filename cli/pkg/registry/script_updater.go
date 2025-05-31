package registry

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ScriptUpdater handles building registry updates from script execution
type ScriptUpdater struct {
	indexer      *contracts.Indexer
	proxyTracker *events.ProxyTracker
}

// NewScriptUpdater creates a new script updater
func NewScriptUpdater(indexer *contracts.Indexer) *ScriptUpdater {
	return &ScriptUpdater{
		indexer:      indexer,
		proxyTracker: events.NewProxyTracker(),
	}
}

// BuildRegistryUpdate builds a registry update from script events
func (su *ScriptUpdater) BuildRegistryUpdate(
	scriptEvents []events.ParsedEvent,
	namespace string,
	chainID uint64,
	networkName string,
	scriptPath string,
) *RegistryUpdate {
	// Create new registry update
	update := NewRegistryUpdate(namespace, chainID, networkName, scriptPath)

	// Process events to identify proxy relationships
	su.proxyTracker.ProcessEvents(scriptEvents)

	// First pass: collect all deployment events and broadcast events
	deploymentEvents := make(map[string][]*events.ContractDeployedEvent)
	broadcastEvents := make(map[string]*events.TransactionBroadcastEvent)
	safeTransactions := make(map[string]*events.SafeTransactionQueuedEvent)
	
	for _, event := range scriptEvents {
		switch e := event.(type) {
		case *events.ContractDeployedEvent:
			txID := e.TransactionID.Hex()
			deploymentEvents[txID] = append(deploymentEvents[txID], e)
		case *events.TransactionBroadcastEvent:
			broadcastEvents[e.TransactionID.Hex()] = e
		case *events.SafeTransactionQueuedEvent:
			// Index safe transactions by their transaction IDs
			for _, richTx := range e.Transactions {
				safeTransactions[richTx.TransactionID.Hex()] = e
			}
		}
	}

	// Get git commit hash
	commitHash := getGitCommit()

	// Process deployments
	for internalTxID, deployEvents := range deploymentEvents {
		// Skip zero transaction ID (indicates dry run or no transaction)
		if internalTxID == "0x0000000000000000000000000000000000000000000000000000000000000000" {
			update.Metadata.DryRun = true
			continue
		}

		// Check if this deployment is part of a Safe transaction
		if safeEvent, exists := safeTransactions[internalTxID]; exists {
			// This is part of a Safe transaction - handle it separately
			su.createOrUpdateSafeTransaction(update, safeEvent, namespace, chainID)
			
			// Add deployments without creating a regular transaction
			for _, deployEvent := range deployEvents {
				deployment := su.createDeploymentFromEvent(
					deployEvent,
					namespace,
					chainID,
					scriptPath,
					commitHash,
				)
				
				// Deployment will be associated with Safe transaction
				update.AddDeployment(internalTxID, deployment)
			}
			continue
		}

		// Check if this deployment has a broadcast event
		if broadcastEvent, exists := broadcastEvents[internalTxID]; exists {
			// This transaction was broadcast - create an executed transaction
			tx := &types.Transaction{
				ID:          "", // Will be set when applying
				ChainID:     chainID,
				Hash:        "", // Will be enriched from broadcast
				Status:      types.TransactionStatusExecuted,
				Sender:      broadcastEvent.Sender.Hex(),
				Deployments: []string{},
				Operations:  []types.Operation{},
				Environment: namespace,
				CreatedAt:   time.Now(),
			}

			// Process deployments for this transaction
			for _, deployEvent := range deployEvents {
				deployment := su.createDeploymentFromEvent(
					deployEvent,
					namespace,
					chainID,
					scriptPath,
					commitHash,
				)

				update.AddDeployment(internalTxID, deployment)

				// Add operation to transaction
				tx.Operations = append(tx.Operations, types.Operation{
					Type:   "DEPLOY",
					Target: deployment.Address,
					Method: string(deployment.DeploymentStrategy.Method),
					Result: map[string]interface{}{
						"address": deployment.Address,
					},
				})
			}

			// Add transaction to update
			update.AddTransaction(internalTxID, tx)
		} else {
			// No broadcast event - this is a simulated transaction
			// Don't create a transaction record, just add deployments
			for _, deployEvent := range deployEvents {
				deployment := su.createDeploymentFromEvent(
					deployEvent,
					namespace,
					chainID,
					scriptPath,
					commitHash,
				)
				
				// Mark as dry-run/simulated
				update.Metadata.DryRun = true
				update.AddDeployment(internalTxID, deployment)
			}
		}
	}

	return update
}

// createDeploymentFromEvent creates a deployment record from an event
func (su *ScriptUpdater) createDeploymentFromEvent(
	event *events.ContractDeployedEvent,
	namespace string,
	chainID uint64,
	scriptPath string,
	commitHash string,
) *types.Deployment {
	// Get contract info from indexer
	var contractName string
	var contractPath string
	var compilerVersion string = "unknown"
	var isLibrary bool

	if su.indexer != nil {
		contractInfo := su.indexer.GetContractByBytecodeHash(event.Deployment.BytecodeHash.Hex())
		if contractInfo != nil {
			contractName = contractInfo.Name
			contractPath = fmt.Sprintf("%s:%s", contractInfo.Path, contractInfo.Name)
			isLibrary = contractInfo.IsLibrary
			
			// Extract compiler version from artifact metadata
			if contractInfo.Artifact != nil {
				compilerVersion = contractInfo.Artifact.Metadata.Compiler.Version
			}
		}
	}

	// If indexer lookup failed, try to extract contract name from artifact field
	if contractName == "" && event.Deployment.Artifact != "" {
		contractName, contractPath = extractContractNameFromArtifact(event.Deployment.Artifact)
	}

	// Final fallback to "Unknown" 
	if contractName == "" {
		contractName = "Unknown"
		contractPath = "Unknown"
	}

	// Use label from event
	label := event.Deployment.Label

	// Determine deployment type
	deployType := types.SingletonDeployment
	if isLibrary {
		deployType = types.LibraryDeployment
	} else if strings.Contains(strings.ToLower(contractName), "proxy") {
		deployType = types.ProxyDeployment
	}

	// Check if this is a proxy based on events
	var proxyInfo *types.ProxyInfo
	if rel, isProxy := su.proxyTracker.GetRelationshipForProxy(event.Location); isProxy {
		deployType = types.ProxyDeployment
		proxyInfo = &types.ProxyInfo{
			Type:           string(rel.ProxyType),
			Implementation: rel.ImplementationAddress.Hex(),
			History:        []types.ProxyUpgrade{},
		}
		if rel.AdminAddress != nil {
			proxyInfo.Admin = rel.AdminAddress.Hex()
		}
	}

	// Determine deployment method from event
	method := types.DeploymentMethodCreate2 // Default
	switch event.Deployment.CreateStrategy {
	case "CREATE":
		method = types.DeploymentMethodCreate
	case "CREATE2":
		method = types.DeploymentMethodCreate2
	case "CREATE3":
		method = types.DeploymentMethodCreate3
	default:
		// If no strategy specified, infer from salt
		if event.Deployment.Salt == (common.Hash{}) {
			method = types.DeploymentMethodCreate
		}
	}

	return &types.Deployment{
		Namespace:     namespace,
		ChainID:       chainID,
		ContractName:  contractName,
		Label:         label,
		Address:       event.Location.Hex(),
		Type:          deployType,
		TransactionID: "", // Will be set when applying
		DeploymentStrategy: types.DeploymentStrategy{
			Method:          method,
			Salt:            event.Deployment.Salt.Hex(),
			InitCodeHash:    event.Deployment.InitCodeHash.Hex(),
			ConstructorArgs: fmt.Sprintf("0x%x", event.Deployment.ConstructorArgs),
			Factory:         "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed", // CreateX
			Entropy:         event.Deployment.Entropy,
		},
		ProxyInfo: proxyInfo,
		Artifact: types.ArtifactInfo{
			Path:            contractPath,
			CompilerVersion: compilerVersion,
			BytecodeHash:    event.Deployment.BytecodeHash.Hex(),
			ScriptPath:      extractScriptName(scriptPath),
			GitCommit:       commitHash,
		},
		Verification: types.VerificationInfo{
			Status: types.VerificationStatusUnverified,
		},
		Tags:      []string{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createOrUpdateSafeTransaction creates or updates a Safe transaction record
func (su *ScriptUpdater) createOrUpdateSafeTransaction(
	update *RegistryUpdate,
	event *events.SafeTransactionQueuedEvent,
	namespace string,
	chainID uint64,
) {
	safeTxHash := event.SafeTxHash.Hex()
	
	// Check if we already have this Safe transaction
	if existing, exists := update.SafeTransactions[safeTxHash]; exists {
		// Update internal tx mapping
		for _, richTx := range event.Transactions {
			existing.InternalTxMapping[richTx.TransactionID.Hex()] = len(existing.SafeTransaction.Transactions)
			existing.SafeTransaction.Transactions = append(existing.SafeTransaction.Transactions, types.SafeTxData{
				To:        richTx.Transaction.To.Hex(),
				Value:     richTx.Transaction.Value.String(),
				Data:      fmt.Sprintf("0x%x", richTx.Transaction.Data),
				Operation: 0, // Default to CALL operation
			})
		}
		return
	}

	// Create new Safe transaction
	safeTx := &types.SafeTransaction{
		SafeTxHash:     safeTxHash,
		SafeAddress:    event.Safe.Hex(),
		ChainID:        chainID,
		Status:         types.TransactionStatusPending,
		Nonce:          0, // TODO: Get nonce from Safe
		Transactions:   []types.SafeTxData{},
		TransactionIDs: []string{}, // Will be populated when applying
		ProposedAt:     time.Now(),
		ProposedBy:     event.Proposer.Hex(),
		Confirmations:  []types.Confirmation{},
	}

	// Build internal tx mapping
	internalTxMapping := make(map[string]int)
	
	// Convert transactions
	for i, richTx := range event.Transactions {
		safeTx.Transactions = append(safeTx.Transactions, types.SafeTxData{
			To:        richTx.Transaction.To.Hex(),
			Value:     richTx.Transaction.Value.String(),
			Data:      fmt.Sprintf("0x%x", richTx.Transaction.Data),
			Operation: 0, // Default to CALL operation
		})
		
		internalTxMapping[richTx.TransactionID.Hex()] = i
	}

	update.AddSafeTransaction(safeTx, internalTxMapping)
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

// extractScriptName extracts the script name from a script path
// e.g., "script/deploy/DeployCounter.s.sol" -> "DeployCounter.s.sol:DeployCounter"
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

// extractContractNameFromArtifact extracts contract name and path from artifact string
// Handles formats like:
// - "src/Counter.sol:Counter" -> ("Counter", "src/Counter.sol:Counter")
// - "ERC1967Proxy" -> ("ERC1967Proxy", "ERC1967Proxy")
// - "<user-provided-bytecode>" -> ("UserProvidedBytecode", "<user-provided-bytecode>")
func extractContractNameFromArtifact(artifact string) (string, string) {
	if artifact == "" {
		return "", ""
	}
	
	// Handle special case of user-provided bytecode
	if artifact == "<user-provided-bytecode>" {
		return "UserProvidedBytecode", artifact
	}
	
	// Check if it's in the format "path:contractName"
	if strings.Contains(artifact, ":") {
		parts := strings.Split(artifact, ":")
		if len(parts) >= 2 {
			contractName := parts[len(parts)-1] // Last part is the contract name
			return contractName, artifact
		}
	}
	
	// If no colon, assume the entire string is the contract name
	return artifact, artifact
}


package script

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// UpdateRegistryFromEvents updates the deployment registry with parsed events
func UpdateRegistryFromEvents(
	events []ParsedEvent,
	networkName string,
	chainID uint64,
	namespace string,
	scriptPath string,
	broadcastPath string,
	indexer *contracts.Indexer,
) error {
	// Load registry
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get git commit hash if available
	commitHash := getGitCommit()

	// Load broadcast transactions if available
	var broadcastTxs []broadcast.TransactionInfo
	if broadcastPath != "" {
		// Extract script name from broadcast path (e.g., "broadcast/DeployWithTreb.s.sol/31337/run-latest.json" -> "DeployWithTreb.s.sol")
		scriptName := extractScriptNameFromBroadcastPath(broadcastPath)
		if scriptName != "" {
			// Use script-specific broadcast parser
			parser := broadcast.NewParser(".")
			broadcastFile, err := parser.ParseLatestBroadcast(scriptName, chainID)
			if err == nil {
				// Convert BroadcastFile to TransactionInfo format
				txs := convertBroadcastFileToTransactionInfos(broadcastFile)
				broadcastTxs = txs
				fmt.Printf("📋 Loaded %d broadcast transactions from %s\n", len(txs), broadcastPath)
			} else {
				fmt.Printf("⚠️  Warning: Failed to load broadcast transactions from %s: %v\n", broadcastPath, err)
			}
		} else {
			fmt.Printf("⚠️  Warning: Could not extract script name from broadcast path: %s\n", broadcastPath)
		}
	} else {
		fmt.Printf("⚠️  Warning: No broadcast path provided, transaction hashes will not be available\n")
	}

	// Create proxy tracker to identify proxy relationships
	proxyTracker := NewProxyTracker()
	proxyTracker.ProcessEvents(events)

	// Group events by transaction ID to process deployments together
	txMap := make(map[string][]*ContractDeployedEvent)
	for _, event := range events {
		if deployEvent, ok := event.(*ContractDeployedEvent); ok {
			txID := deployEvent.TransactionID.Hex()
			txMap[txID] = append(txMap[txID], deployEvent)
		}
	}

	// Track statistics
	deploymentsWithTxHash := 0
	deploymentsWithoutTxHash := 0
	
	// Process each transaction
	for txID, deployEvents := range txMap {
		// Process each deployment in the transaction
		for _, deployEvent := range deployEvents {
			// Find matching broadcast transaction for this specific deployment
			var txHash *common.Hash
			var blockNumber uint64
			if len(broadcastTxs) > 0 {
				// Match by deployed contract address
				deployedAddress := deployEvent.Location.Hex()
				matched := false
				
				
				for _, tx := range broadcastTxs {
					if strings.EqualFold(tx.ContractAddr, deployedAddress) {
						hash := common.HexToHash(tx.Hash)
						txHash = &hash
						blockNumber = tx.BlockNumber
						matched = true
						deploymentsWithTxHash++
						break
					}
				}
				
				// If no match by contract address, try to match by CreateX pattern
				// CreateX deployments won't have ContractAddr set in the transaction
				if !matched {
					// Look for transactions to CreateX that might have deployed this contract
					for _, tx := range broadcastTxs {
						// Check if this is a transaction to CreateX (0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed)
						if strings.EqualFold(tx.To, "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed") {
							// For now, we'll use the first CreateX transaction from the same deployer
							// This is a simplified approach - ideally we'd decode the transaction data
							if strings.EqualFold(tx.From, deployEvent.Deployer.Hex()) {
								hash := common.HexToHash(tx.Hash)
								txHash = &hash
								blockNumber = tx.BlockNumber
								matched = true
								deploymentsWithTxHash++
								break
							}
						}
					}
				}
				
				if !matched {
					deploymentsWithoutTxHash++
					// No broadcast transaction found for this deployment
					// This can happen if the script was run with --dry-run or if broadcast failed
					fmt.Printf("⚠️  Warning: No transaction hash found for deployment at %s\n", deployedAddress)
					fmt.Printf("   Deployer: %s\n", deployEvent.Deployer.Hex())
					fmt.Printf("   This can happen if:\n")
					fmt.Printf("   - The script was run with --dry-run\n")
					fmt.Printf("   - The broadcast file is missing or incomplete\n")
					fmt.Printf("   - CreateX deployment without proper contract address in receipt\n")
				}
			} else {
				// No broadcast transactions loaded at all
				deploymentsWithoutTxHash++
			}
			// Get contract info from indexer
			var contractInfo *contracts.ContractInfo
			var contractName string
			var contractPath string
			var compilerVersion string = "unknown"
			
			if indexer != nil {
				contractInfo = indexer.GetContractByBytecodeHash(deployEvent.Deployment.BytecodeHash.Hex())
				if contractInfo != nil {
					contractName = contractInfo.Name
					contractPath = fmt.Sprintf("%s:%s", contractInfo.Path, contractInfo.Name)
					// Extract compiler version from artifact metadata
					if contractInfo.Artifact != nil {
						compilerVersion = contractInfo.Artifact.Metadata.Compiler.Version
					}
				}
			}
			
			if contractName == "" {
				contractName = "Unknown"
				contractPath = "Unknown"
			}
			
			// Fallback to forge version if we couldn't get it from metadata
			if compilerVersion == "" || compilerVersion == "unknown" {
				compilerVersion = getCompilerVersion()
			}

			// Determine deployment type
			deployType := types.SingletonDeployment
			if contractInfo != nil {
				if contractInfo.IsLibrary {
					deployType = types.LibraryDeployment
				} else if strings.Contains(strings.ToLower(contractName), "proxy") {
					deployType = types.ProxyDeployment
				}
			}
			
			// Check if this is a proxy based on events
			if rel, isProxy := proxyTracker.GetRelationshipForProxy(deployEvent.Location); isProxy {
				deployType = types.ProxyDeployment
				// The relationship will be added to metadata later
				_ = rel
			}

			// Use label from the deployment event
			label := deployEvent.Deployment.Label

			// Create FQID and ShortID
			shortID := contractName
			if label != "" {
				shortID = fmt.Sprintf("%s:%s", contractName, label)
			}
			fqid := fmt.Sprintf("%d/%s/%s:%s", chainID, namespace, contractPath, shortID)

			// Create deployment entry
			entry := &types.DeploymentEntry{
				FQID:            fqid,
				ShortID:         shortID,
				Address:         deployEvent.Location,
				ContractName:    contractName,
				Namespace:       namespace,
				Type:            deployType,
				Salt:            deployEvent.Deployment.Salt.Hex(),
				InitCodeHash:    deployEvent.Deployment.InitCodeHash.Hex(),
				ConstructorArgs: fmt.Sprintf("0x%x", deployEvent.Deployment.ConstructorArgs),
				Label:           label,
				Tags:            []string{},
				Verification: types.Verification{
					Status:    "pending",
					Verifiers: make(map[string]types.VerifierStatus),
				},
				Deployment: types.DeploymentInfo{
					TxHash:        txHash,
					BlockNumber:   blockNumber,
					BroadcastFile: broadcastPath,
					Timestamp:     time.Now(),
					Status:        types.StatusExecuted,
					Deployer:      deployEvent.Deployer.Hex(),
				},
				Metadata: types.ContractMetadata{
					SourceCommit: commitHash,
					Compiler:     compilerVersion,
					SourceHash:   "", // TODO: Calculate source hash
					ContractPath: contractPath,
					ScriptPath:   scriptPath,
					Extra: map[string]interface{}{
						"artifact":      deployEvent.Deployment.Artifact,
						"entropy":       deployEvent.Deployment.Entropy,
						"transactionId": txID,
					},
				},
			}
			
			// Add proxy metadata if this is a proxy
			if rel, isProxy := proxyTracker.GetRelationshipForProxy(deployEvent.Location); isProxy {
				entry.Metadata.Extra["proxyType"] = rel.ProxyType
				entry.Metadata.Extra["implementation"] = rel.ImplementationAddress.Hex()
				
				if rel.AdminAddress != nil {
					entry.Metadata.Extra["admin"] = rel.AdminAddress.Hex()
				}
				
				if rel.BeaconAddress != nil {
					entry.Metadata.Extra["beacon"] = rel.BeaconAddress.Hex()
				}
			}

			// Handle transaction status
			if txID != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				// Check if this was a Safe transaction
				for _, event := range events {
					if safeEvent, ok := event.(*SafeTransactionQueuedEvent); ok {
						// Check if this deployment was part of a Safe transaction
						for _, safeTx := range safeEvent.Transactions {
							if safeTx.TransactionID.Hex() == txID {
								entry.Deployment.Status = types.StatusQueued
								entry.Deployment.SafeAddress = safeEvent.Safe.Hex()
								safeTxHash := safeEvent.SafeTxHash
								entry.Deployment.SafeTxHash = &safeTxHash
								// Store additional Safe transaction info in metadata
								entry.Metadata.Extra["safeTransactionLabel"] = safeTx.Transaction.Label
								entry.Metadata.Extra["safeTransactionStatus"] = safeTx.Status
								break
							}
						}
					}
				}
			}

			// Add deployment to registry
			if err := registryManager.AddDeployment(networkName, chainID, entry); err != nil {
				return fmt.Errorf("failed to add deployment %s: %w", contractName, err)
			}

			PrintSuccessMessage(fmt.Sprintf("Updated registry for %s at %s", contractName, deployEvent.Location.Hex()))
		}
	}


	return nil
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

// getCompilerVersion returns the Solidity compiler version
func getCompilerVersion() string {
	cmd := exec.Command("forge", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	// Parse version from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "forge") {
			return strings.TrimSpace(line)
		}
	}
	return "unknown"
}

// extractScriptNameFromBroadcastPath extracts the script name from a broadcast file path
// e.g., "broadcast/DeployWithTreb.s.sol/31337/run-latest.json" -> "DeployWithTreb.s.sol"
func extractScriptNameFromBroadcastPath(broadcastPath string) string {
	// Split the path into components
	parts := strings.Split(filepath.ToSlash(broadcastPath), "/")
	
	// Look for the "broadcast" directory and get the next component
	for i, part := range parts {
		if part == "broadcast" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	
	return ""
}

// convertBroadcastFileToTransactionInfos converts a BroadcastFile to TransactionInfo slice
func convertBroadcastFileToTransactionInfos(broadcastFile *broadcast.BroadcastFile) []broadcast.TransactionInfo {
	var txInfos []broadcast.TransactionInfo
	
	for i, tx := range broadcastFile.Transactions {
		// Get block number from corresponding receipt
		var blockNumber uint64
		contractAddr := tx.ContractAddress
		
		if i < len(broadcastFile.Receipts) {
			receipt := broadcastFile.Receipts[i]
			// Parse hex block number
			if strings.HasPrefix(receipt.BlockNumber, "0x") {
				if parsed, err := strconv.ParseUint(receipt.BlockNumber[2:], 16, 64); err == nil {
					blockNumber = parsed
				}
			}
			// Use receipt's contract address if available
			if receipt.ContractAddress != "" && receipt.ContractAddress != "0x0000000000000000000000000000000000000000" {
				contractAddr = receipt.ContractAddress
			}
		}
		
		txInfos = append(txInfos, broadcast.TransactionInfo{
			Hash:         tx.Hash,
			BlockNumber:  blockNumber,
			From:         tx.Transaction.From,
			To:           tx.Transaction.To,
			Value:        tx.Transaction.Value,
			Data:         tx.Transaction.Data,
			ContractName: tx.ContractName,
			ContractAddr: contractAddr,
		})
		
		// Also add entries for additional contracts (CreateX deployments)
		for _, additional := range tx.AdditionalContracts {
			// Only include CREATE and CREATE2 contracts, not CREATE3 proxy contracts
			if additional.TransactionType == "CREATE" || additional.TransactionType == "CREATE2" {
				txInfos = append(txInfos, broadcast.TransactionInfo{
					Hash:         tx.Hash,
					BlockNumber:  blockNumber,
					From:         tx.Transaction.From,
					To:           tx.Transaction.To,
					Value:        tx.Transaction.Value,
					Data:         tx.Transaction.Data,
					ContractName: tx.ContractName,
					ContractAddr: additional.Address,
				})
			}
		}
	}
	
	return txInfos
}
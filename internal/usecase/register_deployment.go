package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// RegisterDeploymentParams contains parameters for registering an external deployment
type RegisterDeploymentParams struct {
	Address      string   // Contract address (optional if txHash provided)
	ContractPath string   // Contract path in format "path/to/Contract.sol:ContractName" (optional)
	TxHash       string   // Transaction hash (required)
	Label        string   // Optional label for the deployment (for single contract)
	SkipVerify   bool     // Skip bytecode verification
	Contracts    []ContractRegistration // Pre-filled contract registrations (for multiple contracts)
}

// ContractRegistration represents a single contract to register
type ContractRegistration struct {
	Address        string
	ContractPath   string
	Label          string
	Kind           string // CREATE, CREATE2, or CREATE3
	IsProxy        bool   // True if this contract is a proxy
	Implementation string // Address of the implementation contract (if this is a proxy)
	ImplTxHash     string // Transaction hash for implementation (if different from proxy tx)
}

// RegisterDeploymentResult contains the result of registering a deployment
type RegisterDeploymentResult struct {
	DeploymentIDs []string // Multiple deployment IDs if multiple contracts were registered
	Addresses     []string // Multiple addresses
	ContractNames []string // Multiple contract names
	Labels        []string // Multiple labels
}

// RegisterDeployment is the use case for registering external deployments
type RegisterDeployment struct {
	config            *config.RuntimeConfig
	repo              DeploymentRepository
	blockchainChecker BlockchainChecker
	contractRepo      ContractRepository
}

// NewRegisterDeployment creates a new RegisterDeployment use case
func NewRegisterDeployment(
	cfg *config.RuntimeConfig,
	repo DeploymentRepository,
	blockchainChecker BlockchainChecker,
	contractRepo ContractRepository,
) *RegisterDeployment {
	return &RegisterDeployment{
		config:            cfg,
		repo:              repo,
		blockchainChecker: blockchainChecker,
		contractRepo:      contractRepo,
	}
}

// TraceTransaction traces a transaction to find contract creations
// This is a helper method for the CLI to use before prompting for labels
func (uc *RegisterDeployment) TraceTransaction(ctx context.Context, txHash string) ([]models.ContractCreation, error) {
	if uc.config.Network == nil {
		return nil, fmt.Errorf("network must be configured")
	}

	rpcURL := uc.config.Network.RPCURL
	if rpcURL == "" {
		return nil, fmt.Errorf("RPC URL not configured for network %s", uc.config.Network.Name)
	}

	// Use a longer timeout context for connection
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := uc.blockchainChecker.Connect(connectCtx, rpcURL, uc.config.Network.ChainID); err != nil {
		return nil, fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	adapter, ok := uc.blockchainChecker.(interface {
		TraceTransaction(ctx context.Context, txHash string) ([]models.ContractCreation, error)
	})
	if !ok {
		return nil, fmt.Errorf("blockchain checker does not support transaction tracing")
	}

	return adapter.TraceTransaction(ctx, txHash)
}

// Run executes the register deployment use case
func (uc *RegisterDeployment) Run(ctx context.Context, params RegisterDeploymentParams) (*RegisterDeploymentResult, error) {
	if uc.config.Network == nil {
		return nil, fmt.Errorf("network must be configured")
	}

	if uc.config.Namespace == "" {
		return nil, fmt.Errorf("namespace must be configured")
	}

	if params.TxHash == "" {
		return nil, fmt.Errorf("transaction hash is required")
	}

	// Connect to blockchain (only if not already connected)
	// Note: TraceTransaction may have already connected, but Connect is idempotent
	rpcURL := uc.config.Network.RPCURL
	if rpcURL == "" {
		return nil, fmt.Errorf("RPC URL not configured for network %s", uc.config.Network.Name)
	}

	// Use a longer timeout context for connection
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := uc.blockchainChecker.Connect(connectCtx, rpcURL, uc.config.Network.ChainID); err != nil {
		return nil, fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	// Fetch transaction and receipt
	adapter, ok := uc.blockchainChecker.(interface {
		GetTransaction(ctx context.Context, txHash string) (*types.Transaction, *types.Receipt, error)
		TraceTransaction(ctx context.Context, txHash string) ([]models.ContractCreation, error)
	})
	if !ok {
		return nil, fmt.Errorf("blockchain checker does not support transaction fetching")
	}

	tx, receipt, err := adapter.GetTransaction(ctx, params.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	// Extract sender from transaction
	var senderAddr common.Address
	chainID := tx.ChainId()
	if chainID != nil {
		signer := types.NewLondonSigner(chainID)
		sender, err := types.Sender(signer, tx)
		if err != nil {
			senderAddr = common.Address{}
		} else {
			senderAddr = sender
		}
	} else {
		signer := types.HomesteadSigner{}
		sender, err := types.Sender(signer, tx)
		if err != nil {
			senderAddr = common.Address{}
		} else {
			senderAddr = sender
		}
	}

	// Determine which contracts to register
	var contractsToRegister []ContractRegistration

	if len(params.Contracts) > 0 {
		// Use pre-filled contracts (from interactive mode)
		contractsToRegister = params.Contracts
	} else if params.Address != "" {
		// Single contract with explicit address
		contractsToRegister = []ContractRegistration{
			{
				Address:      params.Address,
				ContractPath: params.ContractPath,
				Label:        params.Label,
			},
		}
	} else {
			// Trace transaction to find all contract creations
			creations, err := adapter.TraceTransaction(ctx, params.TxHash)
			if err != nil {
				return nil, fmt.Errorf("failed to trace transaction: %w", err)
			}

			if len(creations) == 0 {
				// Fallback: try to get from receipt
				if receipt.ContractAddress != (common.Address{}) {
					creations = []models.ContractCreation{
						{
							Address: strings.ToLower(receipt.ContractAddress.Hex()),
							Kind:    "CREATE",
						},
					}
				} else {
					return nil, fmt.Errorf("no contract creations found in transaction trace")
				}
			}

		// Convert creations to registrations
		contractsToRegister = make([]ContractRegistration, len(creations))
		for i, creation := range creations {
			contractsToRegister[i] = ContractRegistration{
				Address:      creation.Address,
				ContractPath: params.ContractPath, // Will be filled interactively if needed
				Label:        params.Label,        // Will be filled interactively if needed
			}
		}
	}

	// Register each contract
	result := &RegisterDeploymentResult{
		DeploymentIDs: make([]string, 0, len(contractsToRegister)),
		Addresses:     make([]string, 0, len(contractsToRegister)),
		ContractNames: make([]string, 0, len(contractsToRegister)),
		Labels:        make([]string, 0, len(contractsToRegister)),
	}

	for _, contract := range contractsToRegister {
		// Normalize address
		contractAddr := common.HexToAddress(contract.Address)
		contractAddress := strings.ToLower(contractAddr.Hex())

		// Check if deployment already exists
		existing, err := uc.repo.GetDeploymentByAddress(ctx, uc.config.Network.ChainID, contractAddress)
		if err == nil && existing != nil {
			// If it already exists, skip it (might be implementation already registered)
			continue
		}

		// Determine transaction hash for this contract
		contractTxHash := params.TxHash
		if contract.ImplTxHash != "" {
			contractTxHash = contract.ImplTxHash
		}

		// Extract contract name from path
		contractName := ""
		if contract.ContractPath != "" {
			parts := strings.Split(contract.ContractPath, ":")
			if len(parts) == 2 {
				contractName = parts[1]
			} else {
				pathParts := strings.Split(contract.ContractPath, "/")
				if len(pathParts) > 0 {
					lastPart := pathParts[len(pathParts)-1]
					contractName = strings.TrimSuffix(lastPart, ".sol")
				}
			}
		}

		// If contract name is still empty, use label as fallback
		if contractName == "" {
			if contract.Label != "" {
				contractName = contract.Label
			} else {
				// Try to infer from contract repository by matching bytecode
				if !params.SkipVerify {
					contractName = uc.inferContractName(ctx, contractAddress)
				}
				if contractName == "" {
					return nil, fmt.Errorf("contract name could not be determined for address %s, please provide --contract or --label", contractAddress)
				}
			}
		}

		// Verify bytecode if requested
		if !params.SkipVerify && contract.ContractPath != "" {
			if err := uc.verifyBytecode(ctx, contractAddress, contract.ContractPath); err != nil {
				return nil, fmt.Errorf("bytecode verification failed for %s: %w", contractAddress, err)
			}
		}

		// Generate deployment ID
		deploymentID := uc.generateDeploymentID(contractName, contract.Label)

		// Determine deployment type and proxy info
		deploymentType := models.SingletonDeployment
		var proxyInfo *models.ProxyInfo

		if contract.IsProxy && contract.Implementation != "" {
			deploymentType = models.ProxyDeployment
			proxyInfo = &models.ProxyInfo{
				Type:           "UUPS", // Default to UUPS, could be detected more precisely in the future
				Implementation: strings.ToLower(contract.Implementation),
				History:        []models.ProxyUpgrade{},
			}
		}

		// Determine deployment method from contract kind
		var deploymentMethod models.DeploymentMethod
		switch strings.ToUpper(contract.Kind) {
		case "CREATE2":
			deploymentMethod = models.DeploymentMethodCreate2
		case "CREATE3":
			deploymentMethod = models.DeploymentMethodCreate3
		default:
			deploymentMethod = models.DeploymentMethodCreate
		}

		// Create deployment record
		deployment := &models.Deployment{
			ID:            deploymentID,
			Namespace:    uc.config.Namespace,
			ChainID:       uc.config.Network.ChainID,
			ContractName:  contractName,
			Label:         contract.Label,
			Address:       contractAddress,
			Type:          deploymentType,
			TransactionID: fmt.Sprintf("tx-%s", contractTxHash),
			DeploymentStrategy: models.DeploymentStrategy{
				Method: deploymentMethod,
			},
			ProxyInfo: proxyInfo,
			Artifact: models.ArtifactInfo{
				Path: contract.ContractPath,
			},
			Verification: models.VerificationInfo{
				Status: models.VerificationStatusUnverified,
			},
			Tags:      []string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save deployment
		if err := uc.repo.SaveDeployment(ctx, deployment); err != nil {
			return nil, fmt.Errorf("failed to save deployment: %w", err)
		}

		result.DeploymentIDs = append(result.DeploymentIDs, deploymentID)
		result.Addresses = append(result.Addresses, contractAddress)
		result.ContractNames = append(result.ContractNames, contractName)
		result.Labels = append(result.Labels, contract.Label)
	}

	// Create or update transaction records for all unique transaction hashes
	seenTxHashes := make(map[string]bool)
	for _, contract := range contractsToRegister {
		contractTxHash := params.TxHash
		if contract.ImplTxHash != "" {
			contractTxHash = contract.ImplTxHash
		}

		if seenTxHashes[contractTxHash] {
			continue
		}
		seenTxHashes[contractTxHash] = true

		txID := fmt.Sprintf("tx-%s", contractTxHash)
		existingTx, _ := uc.repo.GetTransaction(ctx, txID)
		if existingTx == nil {
			// Fetch transaction and receipt for this hash
			var txForHash *types.Transaction
			var receiptForHash *types.Receipt
			var senderForHash common.Address

			if contractTxHash == params.TxHash {
				// Use already fetched transaction
				txForHash = tx
				receiptForHash = receipt
				senderForHash = senderAddr
			} else {
				// Fetch new transaction
				txForHash, receiptForHash, err = adapter.GetTransaction(ctx, contractTxHash)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch implementation transaction: %w", err)
				}

				// Extract sender
				chainID := txForHash.ChainId()
				if chainID != nil {
					signer := types.NewLondonSigner(chainID)
					sender, err := types.Sender(signer, txForHash)
					if err == nil {
						senderForHash = sender
					}
				} else {
					signer := types.HomesteadSigner{}
					sender, err := types.Sender(signer, txForHash)
					if err == nil {
						senderForHash = sender
					}
				}
			}

			// Collect all contracts for this transaction
			var txContracts []ContractRegistration
			for _, c := range contractsToRegister {
				cTxHash := params.TxHash
				if c.ImplTxHash != "" {
					cTxHash = c.ImplTxHash
				}
				if cTxHash == contractTxHash {
					txContracts = append(txContracts, c)
				}
			}

			operations := make([]models.Operation, len(txContracts))
			deploymentIDs := make([]string, 0, len(txContracts))
			for i, c := range txContracts {
				operations[i] = models.Operation{
					Type:   "DEPLOY",
					Target: strings.ToLower(c.Address),
					Method: strings.ToUpper(c.Kind),
					Result: map[string]any{
						"address": strings.ToLower(c.Address),
					},
				}
				// Find deployment ID for this contract
				for j, addr := range result.Addresses {
					if strings.EqualFold(addr, c.Address) {
						deploymentIDs = append(deploymentIDs, result.DeploymentIDs[j])
						break
					}
				}
			}

			transaction := &models.Transaction{
				ID:          txID,
				ChainID:     uc.config.Network.ChainID,
				Hash:        contractTxHash,
				Status:      models.TransactionStatusExecuted,
				BlockNumber: receiptForHash.BlockNumber.Uint64(),
				Sender:      strings.ToLower(senderForHash.Hex()),
				Nonce:       txForHash.Nonce(),
				Deployments: deploymentIDs,
				Operations:  operations,
				Environment: uc.config.Namespace,
				CreatedAt:   time.Now(),
			}

			if err := uc.repo.SaveTransaction(ctx, transaction); err != nil {
				return nil, fmt.Errorf("failed to save transaction: %w", err)
			}
		} else {
			// Collect deployment IDs for this transaction
			var txDeploymentIDs []string
			for _, c := range contractsToRegister {
				cTxHash := params.TxHash
				if c.ImplTxHash != "" {
					cTxHash = c.ImplTxHash
				}
				if cTxHash == contractTxHash {
					// Find deployment ID for this contract
					for j, addr := range result.Addresses {
						if strings.EqualFold(addr, c.Address) {
							txDeploymentIDs = append(txDeploymentIDs, result.DeploymentIDs[j])
							break
						}
					}
				}
			}

			// Update existing transaction with new deployments
			existingTx.Deployments = append(existingTx.Deployments, txDeploymentIDs...)
			if err := uc.repo.SaveTransaction(ctx, existingTx); err != nil {
				return nil, fmt.Errorf("failed to update transaction: %w", err)
			}
		}
	}

	return result, nil
}

// generateDeploymentID generates a deployment ID in the format namespace/chainId/ContractName:label
func (uc *RegisterDeployment) generateDeploymentID(contractName, label string) string {
	id := fmt.Sprintf("%s/%d/%s", uc.config.Namespace, uc.config.Network.ChainID, contractName)
	if label != "" {
		id = fmt.Sprintf("%s:%s", id, label)
	}
	return id
}

// verifyBytecode verifies that the on-chain bytecode matches the compiled contract
func (uc *RegisterDeployment) verifyBytecode(ctx context.Context, address, contractPath string) error {
	// Get contract from repository
	contract := uc.contractRepo.GetContractByArtifact(ctx, contractPath)
	if contract == nil {
		return fmt.Errorf("contract not found: %s", contractPath)
	}

	if contract.Artifact == nil {
		return fmt.Errorf("contract artifact not available for %s", contractPath)
	}

	// Get on-chain bytecode
	adapter, ok := uc.blockchainChecker.(interface {
		GetCode(ctx context.Context, address string) ([]byte, error)
	})
	if !ok {
		return fmt.Errorf("blockchain checker does not support code fetching")
	}

	onChainCode, err := adapter.GetCode(ctx, address)
	if err != nil {
		return fmt.Errorf("failed to fetch on-chain bytecode: %w", err)
	}

	// Decode deployed bytecode from artifact (remove 0x prefix if present)
	deployedBytecodeStr := contract.Artifact.DeployedBytecode.Object
	if strings.HasPrefix(deployedBytecodeStr, "0x") {
		deployedBytecodeStr = deployedBytecodeStr[2:]
	}
	expectedBytecode, err := hex.DecodeString(deployedBytecodeStr)
	if err != nil {
		return fmt.Errorf("failed to decode expected bytecode: %w", err)
	}

	// Compare bytecode hashes (deployed bytecode)
	// Note: We compare the full deployed bytecode. In practice, constructor arguments
	// are not part of deployed bytecode, but linked libraries are. If libraries are
	// linked differently, the bytecode will differ.
	onChainHash := sha256.Sum256(onChainCode)
	expectedHash := sha256.Sum256(expectedBytecode)

	if onChainHash != expectedHash {
		// For now, we'll warn but allow it if the bytecode lengths are similar
		// This handles cases where constructor args or linked libraries differ
		lengthDiff := len(onChainCode) - len(expectedBytecode)
		if lengthDiff < 0 {
			lengthDiff = -lengthDiff
		}

		// If lengths are very different, it's likely a different contract
		if lengthDiff > 100 {
			return fmt.Errorf("bytecode mismatch: on-chain bytecode length %d != expected length %d (likely different contract)", len(onChainCode), len(expectedBytecode))
		}

		// If lengths are similar, it might just be constructor args or linked libraries
		// We'll allow it but warn the user
		return fmt.Errorf("bytecode hash mismatch: on-chain %x != expected %x (lengths match, might be constructor args or linked libraries - use --skip-verify to bypass)", onChainHash[:8], expectedHash[:8])
	}

	return nil
}

// inferContractName tries to infer the contract name by matching bytecode
func (uc *RegisterDeployment) inferContractName(ctx context.Context, address string) string {
	// This is a placeholder - in practice, you'd want to:
	// 1. Get on-chain bytecode
	// 2. Search through all contracts in the repository
	// 3. Find the one with matching bytecode hash
	// For now, return empty string to require explicit contract path
	return ""
}


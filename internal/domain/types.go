package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Core deployment types migrated from pkg/types

// DeploymentType represents the type of deployment
type DeploymentType string

const (
	SingletonDeployment DeploymentType = "SINGLETON"
	ProxyDeployment     DeploymentType = "PROXY"
	LibraryDeployment   DeploymentType = "LIBRARY"
	UnknownDeployment   DeploymentType = "UNKNOWN"
)

// DeploymentMethod represents how the contract was deployed
type DeploymentMethod string

const (
	DeploymentMethodCreate  DeploymentMethod = "CREATE"
	DeploymentMethodCreate2 DeploymentMethod = "CREATE2"
	DeploymentMethodCreate3 DeploymentMethod = "CREATE3"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusSimulated TransactionStatus = "SIMULATED"
	TransactionStatusQueued    TransactionStatus = "QUEUED"
	TransactionStatusExecuted  TransactionStatus = "EXECUTED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// VerificationStatus represents the verification status
type VerificationStatus string

const (
	VerificationStatusUnverified VerificationStatus = "UNVERIFIED"
	VerificationStatusVerified   VerificationStatus = "VERIFIED"
	VerificationStatusFailed     VerificationStatus = "FAILED"
	VerificationStatusPartial    VerificationStatus = "PARTIAL"
)

// Deployment represents a contract deployment record
type Deployment struct {
	// Core identification
	ID            string         `json:"id"`        // e.g., "production/1/Counter:v1"
	Namespace     string         `json:"namespace"` // e.g., "production", "staging", "test"
	ChainID       uint64         `json:"chainId"`
	ContractName  string         `json:"contractName"`  // e.g., "Counter"
	Label         string         `json:"label"`         // e.g., "v1", "main", "usdc"
	Address       string         `json:"address"`       // Contract address
	Type          DeploymentType `json:"type"`          // SINGLETON, PROXY, LIBRARY
	TransactionID string         `json:"transactionId"` // Reference to transaction record

	// Deployment strategy
	DeploymentStrategy DeploymentStrategy `json:"deploymentStrategy"`

	// Proxy information (null for non-proxy deployments)
	ProxyInfo *ProxyInfo `json:"proxyInfo"`

	// Contract artifact information
	Artifact ArtifactInfo `json:"artifact"`

	// Verification information
	Verification VerificationInfo `json:"verification"`

	// Metadata
	Tags      []string  `json:"tags"` // User-defined tags
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Runtime fields (not persisted)
	Transaction    *Transaction `json:"-"` // Linked transaction data
	Implementation *Deployment  `json:"-"` // Resolved implementation for proxies
}

// DeploymentStrategy contains deployment method details
type DeploymentStrategy struct {
	Method          DeploymentMethod `json:"method"`         // CREATE, CREATE2, CREATE3
	Salt            string           `json:"salt,omitempty"` // For CREATE2/CREATE3
	InitCodeHash    string           `json:"initCodeHash,omitempty"`
	Factory         string           `json:"factory,omitempty"`         // Factory address (e.g., CreateX)
	ConstructorArgs string           `json:"constructorArgs,omitempty"` // Hex encoded
	Entropy         string           `json:"entropy,omitempty"`         // Human-readable salt components
}

// ProxyInfo contains proxy-specific information
type ProxyInfo struct {
	Type           string         `json:"type"`            // e.g., "ERC1967", "UUPS", "Transparent"
	Implementation string         `json:"implementation"`  // Current implementation address
	Admin          string         `json:"admin,omitempty"` // Admin address (if applicable)
	History        []ProxyUpgrade `json:"history"`         // Upgrade history
}

// ProxyUpgrade represents a proxy upgrade event
type ProxyUpgrade struct {
	ImplementationID string    `json:"implementationId"` // Deployment ID of implementation
	UpgradedAt       time.Time `json:"upgradedAt"`
	UpgradeTxID      string    `json:"upgradeTxId"` // Transaction ID of upgrade
}

// ArtifactInfo contains contract artifact information
type ArtifactInfo struct {
	Path            string `json:"path"`            // e.g., "src/Counter.sol:Counter"
	CompilerVersion string `json:"compilerVersion"` // e.g., "0.8.19"
	BytecodeHash    string `json:"bytecodeHash"`    // Hash of deployed bytecode
	ScriptPath      string `json:"scriptPath"`      // e.g., "DeployCounter.s.sol:DeployCounter"
	GitCommit       string `json:"gitCommit"`       // Git commit hash at deployment time
}

// VerifierStatus represents the status of a verifier
type VerifierStatus struct {
	Status string `json:"status"` // verified/pending/failed
	URL    string `json:"url,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// VerificationInfo contains verification details
type VerificationInfo struct {
	Status       VerificationStatus        `json:"status"`
	EtherscanURL string                    `json:"etherscanUrl,omitempty"`
	VerifiedAt   *time.Time                `json:"verifiedAt,omitempty"`
	Reason       string                    `json:"reason,omitempty"`
	Verifiers    map[string]VerifierStatus `json:"verifiers,omitempty"` // etherscan, sourcify status
}

// Transaction represents a blockchain transaction record
type Transaction struct {
	// Identification
	ID      string            `json:"id"` // e.g., "tx-0x1234abcd..."
	ChainID uint64            `json:"chainId"`
	Hash    string            `json:"hash"`   // Transaction hash
	Status  TransactionStatus `json:"status"` // PENDING, EXECUTED, FAILED

	// Transaction details
	BlockNumber uint64 `json:"blockNumber,omitempty"`
	Sender      string `json:"sender"` // From address
	Nonce       uint64 `json:"nonce"`

	// Deployment references
	Deployments []string `json:"deployments"` // Deployment IDs created in this tx

	// Operations performed
	Operations []Operation `json:"operations"`

	// Safe context (if applicable)
	SafeContext *SafeContext `json:"safeContext,omitempty"`

	// Metadata
	Environment string    `json:"environment"` // Which environment/namespace
	CreatedAt   time.Time `json:"createdAt"`
}

// Operation represents an operation within a transaction
type Operation struct {
	Type   string                 `json:"type"`   // DEPLOY, CALL, etc.
	Target string                 `json:"target"` // Target address
	Method string                 `json:"method"` // Method called or deployment method
	Result map[string]interface{} `json:"result"` // Operation-specific results
}

// SafeContext contains Safe-specific transaction information
type SafeContext struct {
	SafeAddress     string `json:"safeAddress"`
	SafeTxHash      string `json:"safeTxHash"`
	BatchIndex      int    `json:"batchIndex"` // Index within the batch
	ProposerAddress string `json:"proposerAddress"`
}

// SafeTransaction represents a Safe multisig transaction record
type SafeTransaction struct {
	// Identification
	ID          string       `json:"id"`          // e.g., "safe-tx-0x1234abcd..."
	SafeTxHash  string       `json:"safeTxHash"`  // Safe transaction hash
	ChainID     uint64       `json:"chainId"`     // Chain ID
	SafeAddress string       `json:"safeAddress"` // Safe contract address
	Nonce       uint64       `json:"nonce"`       // Safe nonce
	Status      SafeTxStatus `json:"status"`      // QUEUED, EXECUTED, FAILED

	// Transaction details
	To             string   `json:"to"`             // Target address
	Value          string   `json:"value"`          // Value in wei
	Data           string   `json:"data"`           // Transaction data
	Operation      int      `json:"operation"`      // 0 = Call, 1 = DelegateCall
	ProposedBy     string   `json:"proposedBy"`     // Address that proposed the tx
	TransactionIDs []string `json:"transactionIds"` // Related transaction IDs

	// Execution details
	ExecutionTxHash string     `json:"executionTxHash,omitempty"` // Ethereum tx hash when executed
	ExecutedAt      *time.Time `json:"executedAt,omitempty"`      // When executed

	// Confirmation details
	ConfirmationCount     int            `json:"confirmationCount"`
	ConfirmationsRequired int            `json:"confirmationsRequired"`
	Confirmations         []Confirmation `json:"confirmations"`

	// Metadata
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Confirmation represents a confirmation on a Safe transaction
type Confirmation struct {
	Signer      string    `json:"signer"`
	Signature   string    `json:"signature"`
	ConfirmedAt time.Time `json:"confirmedAt"`
}

// Network represents network configuration
type Network struct {
	ChainID     uint64 `json:"chainId"`
	Name        string `json:"name"`
	RPCURL      string `json:"rpcUrl"`
	ExplorerURL string `json:"explorerUrl,omitempty"`
}

// TrebConfig represents treb-specific configuration
type TrebConfig struct {
	Senders         map[string]SenderConfig `json:"senders"`
	LibraryDeployer string                  `json:"libraryDeployer,omitempty"`
}

// SenderConfig represents a sender configuration
type SenderConfig struct {
	Type           string          `json:"type"`
	Account        string          `json:"account,omitempty"`
	PrivateKey     string          `json:"privateKey,omitempty"`
	Safe           string          `json:"safe,omitempty"`
	DerivationPath string          `json:"derivationPath,omitempty"`
	Proposer       *ProposerConfig `json:"proposer,omitempty"`
	Signer         string          `json:"signer,omitempty"` // Legacy v1 field for Safe senders
}

// ProposerConfig represents proposer configuration for Safe transactions
type ProposerConfig struct {
	Type           string `json:"type"`
	PrivateKey     string `json:"privateKey,omitempty"`
	DerivationPath string `json:"derivationPath,omitempty"`
}

// ContractInfo represents information about a discovered contract
type ContractInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	ArtifactPath string    `json:"artifactPath,omitempty"`
	Version      string    `json:"version,omitempty"`
	Artifact     *Artifact `json:"artifact,omitempty"`
}

// GetDisplayName returns a human-friendly name for the deployment
func (d *Deployment) GetDisplayName() string {
	if d.Label != "" {
		return fmt.Sprintf("%s:%s", d.ContractName, d.Label)
	}
	return d.ContractName
}

// GetShortID returns the short identifier (contractName:label or just contractName)
func (d *Deployment) GetShortID() string {
	if d.Label != "" {
		return fmt.Sprintf("%s:%s", d.ContractName, d.Label)
	}
	return d.ContractName
}

// ContractDisplayName returns the display name for the deployment
func (d *Deployment) ContractDisplayName() string {
	return d.GetShortID()
}

// PruneItem represents an item that should be pruned with the reason
type PruneItem struct {
	ID      string
	Address string // For deployments
	Hash    string // For transactions
	Status  TransactionStatus
	Reason  string
}

// SafePruneItem represents a safe transaction that should be pruned
type SafePruneItem struct {
	SafeTxHash  string
	SafeAddress string
	Status      TransactionStatus
	Reason      string
}

// ItemsToPrune contains all items that should be pruned
type ItemsToPrune struct {
	Deployments      []PruneItem
	Transactions     []PruneItem
	SafeTransactions []SafePruneItem
}

// AnvilInstance represents a local anvil node instance
type AnvilInstance struct {
	Name    string `json:"name"`
	Port    string `json:"port"`
	ChainID string `json:"chainId,omitempty"`
	PidFile string `json:"pidFile"`
	LogFile string `json:"logFile"`
}

// AnvilStatus represents the status of an anvil instance
type AnvilStatus struct {
	Running         bool   `json:"running"`
	PID             int    `json:"pid,omitempty"`
	RPCURL          string `json:"rpcUrl,omitempty"`
	LogFile         string `json:"logFile"`
	RPCHealthy      bool   `json:"rpcHealthy"`
	CreateXDeployed bool   `json:"createXDeployed"`
	CreateXAddress  string `json:"createXAddress,omitempty"`
	Error           string `json:"error,omitempty"`
}

// SafeTxStatus represents the status of a safe transaction
type SafeTxStatus string

const (
	SafeTxStatusQueued   SafeTxStatus = "QUEUED"
	SafeTxStatusExecuted SafeTxStatus = "EXECUTED"
	SafeTxStatusFailed   SafeTxStatus = "FAILED"
)

// LookupIndexes contains various indexes for efficient lookups
type LookupIndexes struct {
	Version string `json:"version"`

	// Address to deployment ID mapping
	ByAddress map[uint64]map[string]string `json:"byAddress"` // chainId -> address -> deploymentId

	// Namespace indexes
	ByNamespace map[string]map[uint64][]string `json:"byNamespace"` // namespace -> chainId -> deploymentIds

	// Contract name indexes
	ByContract map[string][]string `json:"byContract"` // contractName -> deploymentIds

	// Proxy relationships
	Proxies ProxyIndexes `json:"proxies"`

	// Pending items
	Pending PendingItems `json:"pending"`
}

// ProxyIndexes contains proxy relationship mappings
type ProxyIndexes struct {
	// Implementation to proxy mappings
	Implementations map[string][]string `json:"implementations"` // implAddr -> proxyIds

	// Proxy to current implementation
	ProxyToImpl map[string]string `json:"proxyToImpl"` // proxyAddr -> implAddr
}

// PendingItems contains pending transactions
type PendingItems struct {
	SafeTxs []string `json:"safeTxs"` // Pending Safe transaction IDs
}

// SolidityRegistry is a simplified format for Solidity contract consumption
// Structure: chainId -> namespace -> "contractName:label" -> address
type SolidityRegistry map[uint64]map[string]map[string]string

// BytecodeObject represents bytecode information in a Foundry artifact
type BytecodeObject struct {
	Object         string                 `json:"object"`
	SourceMap      string                 `json:"sourceMap"`
	LinkReferences map[string]interface{} `json:"linkReferences"`
}

// ArtifactMetadata represents the metadata section of a Foundry artifact
type ArtifactMetadata struct {
	Compiler struct {
		Version string `json:"version"`
	} `json:"compiler"`
	Language string `json:"language"`
	Output   struct {
		ABI      json.RawMessage `json:"abi"`
		DevDoc   json.RawMessage `json:"devdoc"`
		UserDoc  json.RawMessage `json:"userdoc"`
		Metadata string          `json:"metadata"`
	} `json:"output"`
	Settings struct {
		CompilationTarget map[string]string `json:"compilationTarget"`
	} `json:"settings"`
}

// Artifact represents a Foundry compilation artifact
type Artifact struct {
	ABI               json.RawMessage   `json:"abi"`
	Bytecode          BytecodeObject    `json:"bytecode"`
	DeployedBytecode  BytecodeObject    `json:"deployedBytecode"`
	MethodIdentifiers map[string]string `json:"methodIdentifiers"`
	RawMetadata       string            `json:"rawMetadata"`
	Metadata          ArtifactMetadata  `json:"metadata"`
}

// TransactionFilter defines filtering options for transactions
type TransactionFilter struct {
	ChainID   uint64
	Status    TransactionStatus
	Namespace string
}

// DeploymentFilter defines filtering options for deployments
type DeploymentFilter struct {
	Namespace    string
	ChainID      uint64
	ContractName string
	Label        string
	Type         DeploymentType
}

type ContractQuery struct {
	Query       *string
	PathPattern *string
}

func (cq ContractQuery) String() string {
	if cq.Query == nil && cq.PathPattern == nil {
		return "<all contracts>"
	} else if cq.Query != nil && cq.PathPattern == nil {
		return *cq.Query
	} else if cq.Query == nil && cq.PathPattern != nil {
		return *cq.PathPattern
	} else {
		return fmt.Sprintf("%s and %s)", *cq.Query, *cq.PathPattern)
	}
}

// SafeTransactionFilter defines filtering options for Safe transactions
type SafeTransactionFilter struct {
	ChainID     uint64
	Status      TransactionStatus
	SafeAddress string
}

// SafeExecutionInfo contains execution information for a Safe transaction
type SafeExecutionInfo struct {
	IsExecuted            bool
	TxHash                string
	Confirmations         int
	ConfirmationsRequired int
	ConfirmationDetails   []Confirmation
}

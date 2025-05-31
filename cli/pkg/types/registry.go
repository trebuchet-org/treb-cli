package types

import (
	"fmt"
	"time"
)

// Use DeploymentType from deployment.go
// DeploymentType constants:
// - SingletonDeployment = "SINGLETON"
// - ProxyDeployment = "PROXY"
// - LibraryDeployment = "LIBRARY"
// - UnknownDeployment = "UNKNOWN"

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
	TransactionStatusPending  TransactionStatus = "PENDING"
	TransactionStatusExecuted TransactionStatus = "EXECUTED"
	TransactionStatusFailed   TransactionStatus = "FAILED"
)

// VerificationStatus represents the verification status
type VerificationStatus string

const (
	VerificationStatusUnverified VerificationStatus = "UNVERIFIED"
	VerificationStatusPending    VerificationStatus = "PENDING"
	VerificationStatusVerified   VerificationStatus = "VERIFIED"
	VerificationStatusFailed     VerificationStatus = "FAILED"
	VerificationStatusPartial    VerificationStatus = "PARTIAL"
)

// Deployment represents a contract deployment record
type Deployment struct {
	// Core identification
	ID           string         `json:"id"`           // e.g., "production/1/Counter:v1"
	Namespace    string         `json:"namespace"`    // e.g., "production", "staging", "test"
	ChainID      uint64         `json:"chainId"`      
	ContractName string         `json:"contractName"` // e.g., "Counter"
	Label        string         `json:"label"`        // e.g., "v1", "main", "usdc"
	Address      string         `json:"address"`      // Contract address
	Type         DeploymentType `json:"type"`         // SINGLETON, PROXY, LIBRARY
	TransactionID string        `json:"transactionId"` // Reference to transaction record

	// Deployment strategy
	DeploymentStrategy DeploymentStrategy `json:"deploymentStrategy"`

	// Proxy information (null for non-proxy deployments)
	ProxyInfo *ProxyInfo `json:"proxyInfo"`

	// Contract artifact information
	Artifact ArtifactInfo `json:"artifact"`

	// Verification information
	Verification VerificationInfo `json:"verification"`

	// Metadata
	Tags      []string  `json:"tags"`      // User-defined tags
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	
	// V2 specific fields for verification compatibility
	Status           Status                 `json:"status,omitempty"`           // For execution status
	ConstructorArgs  string                 `json:"constructorArgs,omitempty"`  // For verification
	Metadata         *ContractMetadata      `json:"metadata,omitempty"`         // Additional metadata
	
	// Runtime fields (not persisted)
	Transaction      *Transaction           `json:"-"`                          // Linked transaction data
}

// DeploymentStrategy contains deployment method details
type DeploymentStrategy struct {
	Method         DeploymentMethod `json:"method"`         // CREATE, CREATE2, CREATE3
	Salt           string           `json:"salt,omitempty"` // For CREATE2/CREATE3
	InitCodeHash   string           `json:"initCodeHash,omitempty"`
	Factory        string           `json:"factory,omitempty"`      // Factory address (e.g., CreateX)
	ConstructorArgs string          `json:"constructorArgs,omitempty"` // Hex encoded
	Entropy        string           `json:"entropy,omitempty"`      // Human-readable salt components
}

// ProxyInfo contains proxy-specific information
type ProxyInfo struct {
	Type           string         `json:"type"`           // e.g., "ERC1967", "UUPS", "Transparent"
	Implementation string         `json:"implementation"` // Current implementation address
	Admin          string         `json:"admin,omitempty"` // Admin address (if applicable)
	History        []ProxyUpgrade `json:"history"`        // Upgrade history
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
	ID      string            `json:"id"`      // e.g., "tx-0x1234abcd..."
	ChainID uint64            `json:"chainId"`
	Hash    string            `json:"hash"`    // Transaction hash
	Status  TransactionStatus `json:"status"`  // PENDING, EXECUTED, FAILED

	// Transaction details
	BlockNumber uint64 `json:"blockNumber,omitempty"`
	Sender      string `json:"sender"`     // From address
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
	BatchIndex      int    `json:"batchIndex"`      // Index within the batch
	ProposerAddress string `json:"proposerAddress"`
}

// SafeTransaction represents a Safe multisig transaction batch
type SafeTransaction struct {
	SafeTxHash  string                  `json:"safeTxHash"`
	SafeAddress string                  `json:"safeAddress"`
	ChainID     uint64                  `json:"chainId"`
	Status      TransactionStatus       `json:"status"`
	Nonce       uint64                  `json:"nonce"`

	// Batch of transactions
	Transactions []SafeTxData `json:"transactions"`

	// References to executed transactions
	TransactionIDs []string `json:"transactionIds"`

	// Proposal and confirmation details
	ProposedBy    string         `json:"proposedBy"`
	ProposedAt    time.Time      `json:"proposedAt"`
	Confirmations []Confirmation `json:"confirmations"`

	// Execution details
	ExecutedAt      *time.Time `json:"executedAt,omitempty"`
	ExecutionTxHash string     `json:"executionTxHash,omitempty"`
}

// SafeTxData represents a single transaction in a Safe batch
type SafeTxData struct {
	To        string `json:"to"`
	Value     string `json:"value"`
	Data      string `json:"data"`
	Operation uint8  `json:"operation"` // 0 = Call, 1 = DelegateCall
}

// Confirmation represents a Safe transaction confirmation
type Confirmation struct {
	Signer      string    `json:"signer"`
	Signature   string    `json:"signature"`
	ConfirmedAt time.Time `json:"confirmedAt"`
}

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
	Implementations map[string][]string `json:"implementations"` // implId -> proxyIds

	// Proxy to current implementation
	ProxyToImpl map[string]string `json:"proxyToImpl"` // proxyId -> implId
}

// PendingItems contains pending transactions
type PendingItems struct {
	SafeTxs []string `json:"safeTxs"` // Pending Safe transaction hashes
}

// GetDisplayName returns a human-friendly name for the deployment
func (d *Deployment) GetDisplayName() string {
	if d.Label != "" {
		return fmt.Sprintf("%s:%s", d.ContractName, d.Label)
	}
	return d.ContractName
}

// GetShortID returns the short identifier (contractName:label)
func (d *Deployment) GetShortID() string {
	if d.Label != "" {
		return fmt.Sprintf("%s:%s", d.ContractName, d.Label)
	}
	return d.ContractName
}

// SolidityRegistry is a simplified format for Solidity contract consumption
// Structure: chainId -> namespace -> "contractName:label" -> address
type SolidityRegistry map[uint64]map[string]map[string]string

// RegistryFiles represents all registry files
type RegistryFiles struct {
	Deployments      map[string]*Deployment       `json:"-"`
	Transactions     map[string]*Transaction      `json:"-"`
	SafeTransactions map[string]*SafeTransaction  `json:"-"`
	Lookups          *LookupIndexes               `json:"-"`
	SolidityRegistry SolidityRegistry             `json:"-"`
}
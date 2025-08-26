package models

import (
	"fmt"
	"time"
)

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

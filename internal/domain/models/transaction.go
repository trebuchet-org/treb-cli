package models

import "time"

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusSimulated TransactionStatus = "SIMULATED"
	TransactionStatusQueued    TransactionStatus = "QUEUED"
	TransactionStatusExecuted  TransactionStatus = "EXECUTED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// Transaction represents a blockchain transaction record
type Transaction struct {
	// Identification
	ID      string `json:"id"` // e.g., "tx-0x1234abcd..."
	ChainID uint64 `json:"chainId"`
	Hash    string `json:"hash"` // Transaction hash

	// Transaction details
	Status      TransactionStatus `json:"status"` // PENDING, EXECUTED, FAILED
	BlockNumber uint64            `json:"blockNumber,omitempty"`
	Sender      string            `json:"sender"` // From address
	Nonce       uint64            `json:"nonce"`

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
	Type   string         `json:"type"`   // DEPLOY, CALL, etc.
	Target string         `json:"target"` // Target address
	Method string         `json:"method"` // Method called or deployment method
	Result map[string]any `json:"result"` // Operation-specific results
}

// SafeContext contains Safe-specific transaction information
type SafeContext struct {
	SafeAddress     string `json:"safeAddress"`
	SafeTxHash      string `json:"safeTxHash"`
	BatchIndex      int    `json:"batchIndex"` // Index within the batch
	ProposerAddress string `json:"proposerAddress"`
}

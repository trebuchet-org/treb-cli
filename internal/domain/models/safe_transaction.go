package models

import "time"

type SafeTxStatus string

const (
	SafeTxStatusQueued   SafeTxStatus = "QUEUED"
	SafeTxStatusExecuted SafeTxStatus = "EXECUTED"
	SafeTxStatusFailed   SafeTxStatus = "FAILED"
)

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

// SafeExecutionInfo contains execution information for a Safe transaction
type SafeExecutionInfo struct {
	IsExecuted            bool
	TxHash                string
	Confirmations         int
	ConfirmationsRequired int
	ConfirmationDetails   []Confirmation
}

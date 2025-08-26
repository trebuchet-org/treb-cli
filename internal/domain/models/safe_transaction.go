package models

import "time"

// SafeTransaction represents a Safe multisig transaction record
type SafeTransaction struct {
	// Identification
	SafeTxHash  string            `json:"safeTxHash"`
	SafeAddress string            `json:"safeAddress"`
	ChainID     uint64            `json:"chainId"`
	Status      TransactionStatus `json:"status"`
	Nonce       uint64            `json:"nonce"`

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

// Confirmation represents a confirmation on a Safe transaction
type Confirmation struct {
	Signer      string    `json:"signer"`
	Signature   string    `json:"signature"`
	ConfirmedAt time.Time `json:"confirmedAt"`
}

// SafeTxData represents a single transaction in a Safe batch
type SafeTxData struct {
	To        string `json:"to"`
	Value     string `json:"value"`
	Data      string `json:"data"`
	Operation uint8  `json:"operation"` // 0 = Call, 1 = DelegateCall
}

// SafeExecutionInfo contains execution information for a Safe transaction
type SafeExecutionInfo struct {
	IsExecuted            bool
	TxHash                string
	Confirmations         int
	ConfirmationsRequired int
	ConfirmationDetails   []Confirmation
}

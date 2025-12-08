package models

import "time"

// ProposalStatus represents the status of a Governor proposal
type ProposalStatus string

const (
	ProposalStatusPending   ProposalStatus = "pending"
	ProposalStatusActive    ProposalStatus = "active"
	ProposalStatusSucceeded ProposalStatus = "succeeded"
	ProposalStatusQueued    ProposalStatus = "queued"
	ProposalStatusExecuted  ProposalStatus = "executed"
	ProposalStatusCanceled  ProposalStatus = "canceled"
	ProposalStatusDefeated  ProposalStatus = "defeated"
)

// GovernorProposal represents a Governor proposal record for persistence
type GovernorProposal struct {
	// Identification
	ProposalID      string         `json:"proposalId"`
	GovernorAddress string         `json:"governorAddress"`
	TimelockAddress string         `json:"timelockAddress,omitempty"`
	ChainID         uint64         `json:"chainId"`
	Status          ProposalStatus `json:"status"`

	// References to transactions included in the proposal
	TransactionIDs []string `json:"transactionIds"`

	// Proposal details
	ProposedBy  string    `json:"proposedBy"`
	ProposedAt  time.Time `json:"proposedAt"`
	Description string    `json:"description,omitempty"`

	// Execution details (when executed)
	ExecutedAt      *time.Time `json:"executedAt,omitempty"`
	ExecutionTxHash string     `json:"executionTxHash,omitempty"`
}

package domain

import "github.com/trebuchet-org/treb-cli/internal/domain/models"

// PruneItem represents an item that should be pruned with the reason
type PruneItem struct {
	ID      string
	Address string // For deployments
	Hash    string // For transactions
	Status  models.TransactionStatus
	Reason  string
}

// SafePruneItem represents a safe transaction that should be pruned
type SafePruneItem struct {
	SafeTxHash  string
	SafeAddress string
	Status      models.TransactionStatus
	Reason      string
}

// ItemsToPrune contains all items that should be pruned
type ItemsToPrune struct {
	Deployments      []PruneItem
	Transactions     []PruneItem
	SafeTransactions []SafePruneItem
}

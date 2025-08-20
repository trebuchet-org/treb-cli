package domain

import (
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// TransactionFilter defines filtering options for transactions
type TransactionFilter struct {
	ChainID   uint64
	Status    models.TransactionStatus
	Namespace string
}

// DeploymentFilter defines filtering options for deployments
type DeploymentFilter struct {
	Namespace    string
	ChainID      uint64
	ContractName string
	Label        string
	Type         models.DeploymentType
}


// SafeTransactionFilter defines filtering options for Safe transactions
type SafeTransactionFilter struct {
	ChainID     uint64
	Status      models.TransactionStatus
	SafeAddress string
}

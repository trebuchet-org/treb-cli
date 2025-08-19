package domain

import (
	"fmt"

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
	Status      models.TransactionStatus
	SafeAddress string
}

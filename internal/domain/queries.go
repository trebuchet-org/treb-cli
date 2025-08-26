package domain

import "fmt"

// ContractQuery represents a query for finding contracts
type ContractQuery struct {
	// Query is the search string (e.g., "Counter" or "src/Counter.sol:Counter")
	Query *string
	// PathPattern is an optional path pattern to filter by
	PathPattern *string
	// Filter contracts to only libraries
	IsLibrary *bool
	// Filter contracts to only those that can be proxies
	IsProxy *bool
}

// String returns a string representation of the query
func (cq ContractQuery) String() string {
	if cq.Query == nil && cq.PathPattern == nil {
		return "<all contracts>"
	} else if cq.Query != nil && cq.PathPattern == nil {
		return *cq.Query
	} else if cq.Query == nil && cq.PathPattern != nil {
		return *cq.PathPattern
	} else {
		return fmt.Sprintf("%s and %s", *cq.Query, *cq.PathPattern)
	}
}

// DeploymentQuery represents a query for finding deployments
type DeploymentQuery struct {
	// Reference is the deployment identifier (ID, address, contract name, etc.)
	Reference string
	// Optional: Chain ID for filtering
	ChainID uint64
	// Optional: Namespace for filtering
	Namespace string
}

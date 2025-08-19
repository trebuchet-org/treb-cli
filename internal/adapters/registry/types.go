package registry

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
	Implementations map[string][]string `json:"implementations"` // implAddr -> proxyIds

	// Proxy to current implementation
	ProxyToImpl map[string]string `json:"proxyToImpl"` // proxyAddr -> implAddr
}

// PendingItems contains pending transactions
type PendingItems struct {
	SafeTxs []string `json:"safeTxs"` // Pending Safe transaction IDs
}

type SolidityRegistry map[uint64]map[string]map[string]string

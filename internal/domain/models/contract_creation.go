package models

// ContractCreation represents a contract creation found in a transaction trace
type ContractCreation struct {
	Address        string
	ContractName   string // May be empty if unknown
	Kind           string // CREATE, CREATE2, or CREATE3
	IsProxy        bool   // True if this contract is a proxy
	Implementation string // Address of the implementation contract (if this is a proxy)
}



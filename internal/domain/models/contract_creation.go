package models

// ContractCreation represents a contract creation found in a transaction trace
type ContractCreation struct {
	Address      string
	ContractName string // May be empty if unknown
	Kind         string // CREATE, CREATE2, or CREATE3
}



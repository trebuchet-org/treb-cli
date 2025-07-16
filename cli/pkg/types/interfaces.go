package types

type DeploymentLookup interface {
	GetDeploymentByAddress(chainID uint64, address string) (*Deployment, error)
}

type ContractLookup interface {
	GetContractByArtifact(artifact string) *ContractInfo
	GetContract(key string) (*ContractInfo, error)
	GetContractByBytecodeHash(hash string) *ContractInfo
	FindContractByName(name string, filter ContractQueryFilter) []*ContractInfo
	SearchContracts(pattern string) []*ContractInfo
	GetAllContracts() []*ContractInfo
	GetProxyContracts() []*ContractInfo
	GetDeployableContracts() []*ContractInfo
	GetProxyContractsFiltered(filter ContractQueryFilter) []*ContractInfo
	ResolveContractKey(contract *ContractInfo) string
	QueryContracts(filter ContractQueryFilter) []*ContractInfo
}

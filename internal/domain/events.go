package domain

// EventType represents the type of deployment event
type EventType string

const (
	// EventContractDeployed is emitted when a contract is deployed
	EventContractDeployed EventType = "ContractDeployed"
	
	// EventProxyDeployed is emitted when a proxy contract is deployed
	EventProxyDeployed EventType = "ProxyDeployed"
	
	// EventCreateXContractCreation is emitted by CreateX when creating a contract
	EventCreateXContractCreation EventType = "CreateXContractCreation"
)

// DeploymentEvent represents a deployment-related event from the blockchain
type DeploymentEvent struct {
	EventType      EventType
	Address        string
	Implementation string // For proxy deployments
	ContractName   string
	Namespace      string
	ChainID        uint64
	Deployer       string
	TxHash         string
	Label          string
	Salt           [32]byte
}
package parser

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi/bindings"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ScriptExecution represents the parsed result of a script execution
type ScriptExecution struct {
	// Core execution data
	Transactions       []*Transaction                        // All transactions in execution order
	SafeTransactions   []*SafeTransaction                    // Safe transaction batches
	Deployments        []*DeploymentRecord                   // Contract deployments
	ProxyRelationships map[common.Address]*ProxyRelationship // Proxy relationships

	// Raw data
	Events       []interface{}       // All parsed events
	Logs         []string            // Console logs
	TextOutput   string              // Raw text output
	ParsedOutput *forge.ParsedOutput // Original forge output

	// Execution metadata
	Success       bool
	BroadcastPath string
	Network       string
	ChainID       uint64
	Script        *types.ContractInfo
}

// DeploymentRecord represents a contract deployment
type DeploymentRecord struct {
	TransactionID [32]byte
	Deployment    *bindings.ITrebEventsDeploymentDetails
	Address       common.Address
	Deployer      common.Address
	Contract      *types.ContractInfo
}

// SafeTransaction represents a Safe multisig transaction
type SafeTransaction struct {
	SafeTxHash           [32]byte
	Safe                 common.Address
	Proposer             common.Address
	TransactionIDs       [][32]byte
	Executed             bool         // Whether the Safe transaction was executed directly (threshold=1)
	ExecutionTxHash      *common.Hash // The transaction hash that executed this Safe transaction
	ExecutionBlockNumber *uint64      // The block number where this Safe transaction was executed
}

// ProxyInfo contains proxy relationship information
type ProxyInfo struct {
	Implementation common.Address
	ProxyType      string
	Admin          *common.Address
	Beacon         *common.Address
}

// Transaction represents a transaction with enriched data
type Transaction struct {
	bindings.SimulatedTransaction
	Status          types.TransactionStatus
	TxHash          *common.Hash
	BlockNumber     *uint64
	GasUsed         *uint64
	SafeTransaction *SafeTransaction
	SafeBatchIdx    *int
	Deployments     []DeploymentInfo
	TraceData       *forge.TraceOutput
	ReceiptData     *forge.Receipt
}

// DeploymentInfo contains deployment details for a transaction
type DeploymentInfo struct {
	ContractName string
	ContractType string
	Address      common.Address
	IsProxy      bool
	ProxyInfo    *ProxyInfo
	Label        string
}

// ProxyRelationship represents a proxy-implementation relationship
type ProxyRelationship struct {
	ProxyAddress          common.Address
	ImplementationAddress common.Address
	AdminAddress          *common.Address
	BeaconAddress         *common.Address
	ProxyType             ProxyRelationshipType
}

// ProxyRelationshipType represents the type of proxy relationship
type ProxyRelationshipType string

const (
	ProxyTypeMinimal     ProxyRelationshipType = "MINIMAL"
	ProxyTypeUUPS        ProxyRelationshipType = "UUPS"
	ProxyTypeTransparent ProxyRelationshipType = "TRANSPARENT"
	ProxyTypeBeacon      ProxyRelationshipType = "BEACON"
)

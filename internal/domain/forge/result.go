package forge

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain/bindings"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// ScriptResult contains the parsed result of running a script
type RunResult struct {
	DryRun        bool
	Script        *models.Contract
	Success       bool
	RawOutput     []byte
	ParsedOutput  *ParsedOutput
	BroadcastPath string
	Network       string
	ChainID       uint64
	Namespace     string
	Error         error
}

// HydratedRunResult represents the completely hydrated result of running a script
type HydratedRunResult struct {
	RunResult
	// Core execution data
	Transactions       []*Transaction                                       // All transactions in execution order
	SafeTransactions   []*SafeTransaction                                   // Safe transaction batches
	Deployments        []*Deployment                                        // Contract deployments
	ProxyRelationships map[common.Address]*ProxyRelationship                // Proxy relationships
	Collisions         map[common.Address]*bindings.TrebDeploymentCollision // Deployment collisions (contracts already deployed)
	Events             []any                                                // All parsed events
	ExecutionTime      time.Duration
	ExecutedAt         time.Time
}

// DeploymentRecord represents a contract deployment
type Deployment struct {
	Event         *bindings.ITrebEventsDeploymentDetails
	TransactionID [32]byte
	Address       common.Address
	Deployer      common.Address
	Contract      *models.Contract
}

// ProxyRelationshipType represents the type of proxy relationship
type ProxyRelationshipType string

const (
	ProxyTypeMinimal     ProxyRelationshipType = "MINIMAL"
	ProxyTypeUUPS        ProxyRelationshipType = "UUPS"
	ProxyTypeTransparent ProxyRelationshipType = "TRANSPARENT"
	ProxyTypeBeacon      ProxyRelationshipType = "BEACON"
)

// ProxyRelationship represents a proxy-implementation relationship discovered during execution
type ProxyRelationship struct {
	ProxyAddress          common.Address
	ImplementationAddress common.Address
	ProxyType             ProxyRelationshipType
	AdminAddress          *common.Address
	BeaconAddress         *common.Address
}

type Transaction struct {
	bindings.SimulatedTransaction
	Status          models.TransactionStatus
	TxHash          *common.Hash
	BlockNumber     *uint64
	GasUsed         *uint64
	SafeTransaction *SafeTransaction
	SafeBatchIdx    *int
	Deployments     []DeploymentInfo
	TraceData       *TraceOutput
	ReceiptData     *Receipt
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

// SafeTransaction represents a Safe multisig transaction
type SafeTransaction struct {
	SafeTxHash           [32]byte
	Safe                 common.Address
	Proposer             common.Address
	Executor             common.Address
	TransactionIds       [][32]byte
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

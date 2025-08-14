package domain

import "time"

// ScriptType represents the type of deployment script
type ScriptType string

const (
	ScriptTypeContract ScriptType = "contract"
	ScriptTypeLibrary  ScriptType = "library"
	ScriptTypeProxy    ScriptType = "proxy"
)

// ScriptDeploymentStrategy represents the CREATE opcode strategy for script generation
type ScriptDeploymentStrategy string

const (
	StrategyCreate2 ScriptDeploymentStrategy = "CREATE2"
	StrategyCreate3 ScriptDeploymentStrategy = "CREATE3"
)

// ScriptTemplate contains all information needed to generate a deployment script
type ScriptTemplate struct {
	Type            ScriptType
	ContractName    string
	ArtifactPath    string
	Strategy        ScriptDeploymentStrategy
	ProxyInfo       *ScriptProxyInfo // nil for non-proxy deployments
	ConstructorInfo *ConstructorInfo // nil if no constructor
	ScriptPath      string
}

// ScriptProxyInfo contains proxy-specific deployment information for script generation
type ScriptProxyInfo struct {
	ProxyName       string
	ProxyPath       string
	ProxyArtifact   string
	InitializerInfo *InitializerInfo
}

// ConstructorInfo contains constructor parameter information
type ConstructorInfo struct {
	HasConstructor bool
	Parameters     []Parameter
}

// InitializerInfo contains initializer method information
type InitializerInfo struct {
	MethodName string
	Parameters []Parameter
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string
	Type         string
	InternalType string
}

// ContractABI represents the parsed ABI of a contract
type ContractABI struct {
	Name           string
	HasConstructor bool
	Constructor    *Constructor
	Methods        []Method
}

// Constructor represents the constructor in an ABI
type Constructor struct {
	Inputs []Parameter
}

// Method represents a function in the ABI
type Method struct {
	Name   string
	Inputs []Parameter
}

// ScriptExecution represents the complete result of running a script
type ScriptExecution struct {
	ID              string
	ScriptPath      string
	ScriptName      string
	Network         string
	Namespace       string
	ChainID         uint64
	Success         bool
	DryRun          bool
	Transactions    []ScriptTransaction
	Deployments     []ScriptDeployment
	Events          []ScriptEvent
	Logs            []string
	BroadcastPath   string
	ExecutedAt      time.Time
	ExecutionTime   time.Duration
	Stages          []ExecutionStage
}

// ScriptTransaction represents a transaction executed by a script
type ScriptTransaction struct {
	ID              string
	TransactionID   [32]byte // Internal ID from script
	Sender          string
	To              string
	Value           string
	Data            []byte
	Status          TransactionStatus
	TxHash          *string
	BlockNumber     *uint64
	GasUsed         *uint64
	SafeTransaction *SafeTransactionInfo
}

// SafeTransactionInfo contains information about a Safe multisig transaction
type SafeTransactionInfo struct {
	SafeAddress     string
	SafeTxHash      [32]byte
	Proposer        string
	Executed        bool
	ExecutionTxHash *string
	BatchIndex      *int
}

// ScriptDeployment represents a contract deployment from a script
type ScriptDeployment struct {
	TransactionID    [32]byte
	Address          string
	ContractName     string
	Artifact         string
	Label            string
	Deployer         string
	DeploymentType   DeploymentType
	CreateStrategy   string
	Salt             [32]byte
	InitCodeHash     [32]byte
	ConstructorArgs  []byte
	BytecodeHash     [32]byte
	IsProxy          bool
	ProxyInfo        *ProxyInfo
}

// ScriptEvent represents an event emitted during script execution
type ScriptEvent interface {
	GetEventType() string
}

// ExecutionStage represents a stage of script execution (Simulating, Broadcasting, etc)
type ExecutionStage struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Status    StageStatus
	Skipped   bool
	Duration  time.Duration
}

// StageStatus represents the status of an execution stage
type StageStatus string

const (
	StageStatusRunning   StageStatus = "running"
	StageStatusCompleted StageStatus = "completed"
	StageStatusSkipped   StageStatus = "skipped"
	StageStatusFailed    StageStatus = "failed"
)

// ProxyRelationship represents a proxy-implementation relationship discovered during execution
type ProxyRelationship struct {
	ProxyAddress          string
	ImplementationAddress string
	ProxyType             string
	AdminAddress          *string
	BeaconAddress         *string
}

// CollisionInfo represents a deployment collision detected during execution
type CollisionInfo struct {
	ExistingContract string
	Artifact         string
	Label            string
	Entropy          string
	Salt             [32]byte
	CreateStrategy   string
}

// ParameterType represents the type of a script parameter
type ParameterType string

const (
	ParamTypeString     ParameterType = "string"
	ParamTypeAddress    ParameterType = "address"
	ParamTypeUint256    ParameterType = "uint256"
	ParamTypeInt256     ParameterType = "int256"
	ParamTypeBytes32    ParameterType = "bytes32"
	ParamTypeBytes      ParameterType = "bytes"
	ParamTypeBool       ParameterType = "bool"
	ParamTypeSender     ParameterType = "sender"
	ParamTypeDeployment ParameterType = "deployment"
	ParamTypeArtifact   ParameterType = "artifact"
)

// ScriptParameter represents a parameter expected by a script
type ScriptParameter struct {
	Name        string
	Type        ParameterType
	Description string
	Optional    bool
}

// ScriptParameterValue represents a resolved parameter value
type ScriptParameterValue struct {
	Name  string
	Type  ParameterType
	Value string
}

// ScriptInfo represents information about a resolved script
type ScriptInfo struct {
	Path         string
	Name         string
	ContractName string
	Artifact     *ContractArtifact
}

// ContractArtifact represents compiled contract artifact data
type ContractArtifact struct {
	ABI          []byte
	Bytecode     []byte
	DeployedCode []byte
	Metadata     []byte
}
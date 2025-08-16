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
	Script          string // Script path or name
	ScriptPath      string
	ScriptName      string
	Network         string
	Namespace       string
	ChainID         uint64
	Success         bool
	DryRun          bool
	Error           string
	GasUsed         uint64
	Metadata        map[string]string
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
	From            string   // Transaction sender
	Sender          string   // Alias for From
	To              string
	Value           string
	Data            []byte
	Nonce           uint64
	Status          TransactionStatus
	TxHash          *string
	Hash            string   // Transaction hash
	BlockNumber     *uint64
	GasUsed         *uint64
	DeploymentIDs   []string // IDs of deployments created by this transaction
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
	ArtifactPath     string // Path to artifact file
	Label            string
	Deployer         string
	DeploymentType   DeploymentType
	DeploymentMethod string // CREATE2, CREATE3, etc.
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
	ArtifactPath string
	Artifact     *ContractArtifact
}

// ContractArtifact represents compiled contract artifact data
type ContractArtifact struct {
	ABI          []byte
	Bytecode     []byte
	DeployedCode []byte
	Metadata     []byte
}

// ScriptExecutionEvent represents an event emitted during script execution
type ScriptExecutionEvent struct {
	Type string
	Data map[string]interface{}
}

// BroadcastFile represents a Foundry broadcast file
type BroadcastFile struct {
	Chain        uint64                  `json:"chain"`
	Transactions []BroadcastTransaction  `json:"transactions"`
	Receipts     []BroadcastReceipt      `json:"receipts"`
	Timestamp    uint64                  `json:"timestamp"`
	Commit       string                  `json:"commit"`
}

// BroadcastTransaction represents a transaction in a broadcast file
type BroadcastTransaction struct {
	Hash                string                  `json:"hash"`
	Transaction         map[string]interface{}  `json:"transaction"`
	ContractName        string                  `json:"contractName"`
	ContractAddr        string                  `json:"contractAddress"`
	Function            string                  `json:"function"`
	Arguments           []interface{}           `json:"arguments"`
	AdditionalContracts []AdditionalContract    `json:"additionalContracts,omitempty"`
}

// AdditionalContract represents additional contracts deployed in a transaction
type AdditionalContract struct {
	ContractName string `json:"contractName"`
	ContractAddr string `json:"contractAddress"`
}

// BroadcastReceipt represents a receipt in a broadcast file
type BroadcastReceipt struct {
	TransactionHash string         `json:"transactionHash"`
	BlockNumber     string         `json:"blockNumber"`
	GasUsed         string         `json:"gasUsed"`
	Status          string         `json:"status"`
	ContractAddress string         `json:"contractAddress"`
	Logs            []BroadcastLog `json:"logs"`
}

// BroadcastLog represents a log entry from a transaction receipt
type BroadcastLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// ExecutedTransaction represents a transaction that was executed
type ExecutedTransaction struct {
	ID            string
	Hash          string
	From          string
	To            string
	Value         string
	Data          string
	Nonce         uint64
	GasUsed       uint64
	BlockNumber   uint64
	Status        string
	DeploymentIDs []string
	SafeContext   *SafeTransactionContext
}

// SafeTransactionContext contains Safe-specific transaction context
type SafeTransactionContext struct {
	SafeAddress string
	SafeTxHash  string
	Proposer    string
}

// DeploymentResult represents the result of a deployment
type DeploymentResult struct {
	ID                string
	TransactionID     string
	Address           string
	ContractName      string
	ArtifactPath      string
	Deployer          string
	DeploymentMethod  string
	Salt              [32]byte
	BytecodeHash      [32]byte
	ConstructorArgs   []byte
	Label             string
	Namespace         string
}
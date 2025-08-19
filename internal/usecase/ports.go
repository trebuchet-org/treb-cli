package usecase

import (
	"context"
	"io"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// DeploymentStore handles persistence of deployments
type DeploymentStore interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
	GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*domain.Deployment, error)
	ListDeployments(ctx context.Context, filter domain.DeploymentFilter) ([]*domain.Deployment, error)
	SaveDeployment(ctx context.Context, deployment *domain.Deployment) error
	DeleteDeployment(ctx context.Context, id string) error
}

// TransactionStore handles persistence of transactions
type TransactionStore interface {
	GetTransaction(ctx context.Context, id string) (*domain.Transaction, error)
	ListTransactions(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, error)
	SaveTransaction(ctx context.Context, transaction *domain.Transaction) error
}

// ContractIndexer provides access to compiled contracts
type ContractIndexer interface {
	GetContract(ctx context.Context, key string) (*domain.ContractInfo, error)
	SearchContracts(ctx context.Context, query domain.ContractQuery) []*domain.ContractInfo
	GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo
}

// ForgeExecutor handles forge command execution
type ForgeExecutor interface {
	Build(ctx context.Context) error
	RunScript(ctx context.Context, config domain.ForgeRunConfig) (*domain.ForgeRunResult, error)
}

// ContractVerifier handles contract verification
type ContractVerifier interface {
	Verify(ctx context.Context, deployment *domain.Deployment, network *domain.Network) error
	GetVerificationStatus(ctx context.Context, deployment *domain.Deployment) (*domain.VerificationInfo, error)
}

// Progress tracking interfaces

// ProgressEvent represents a progress update
type ProgressEvent struct {
	Stage    string
	Current  int
	Total    int
	Message  string
	Spinner  bool
	Metadata interface{}
}

// ProgressSink receives progress events
type ProgressSink interface {
	OnProgress(ctx context.Context, event ProgressEvent)
	Info(message string)
	Error(message string)
}

// NopProgress is a no-op implementation of ProgressSink
type NopProgress struct{}

func (NopProgress) OnProgress(context.Context, ProgressEvent) {}
func (NopProgress) Info(string)                               {}
func (NopProgress) Error(string)                              {}

// Use case result types

// DeploymentListResult contains the result of listing deployments
type DeploymentListResult struct {
	Deployments []*domain.Deployment
	Summary     DeploymentSummary
}

// DeploymentSummary provides summary statistics
type DeploymentSummary struct {
	Total       int
	ByNamespace map[string]int
	ByChain     map[uint64]int
	ByType      map[domain.DeploymentType]int
}

// ScriptExecutionResult contains the result of script execution
type ScriptExecutionResult struct {
	Success      bool
	Deployments  []*domain.Deployment
	Transactions []*domain.Transaction
	Logs         []string
	GasUsed      uint64
	Error        error
}

// ABIParser parses contract ABIs to extract constructor/initializer info
type ABIParser interface {
	ParseContractABI(ctx context.Context, contractName string) (*domain.ContractABI, error)
	FindInitializeMethod(abi *domain.ContractABI) *domain.Method
	GenerateConstructorArgs(abi *domain.ContractABI) (vars string, encode string)
	GenerateInitializerArgs(method *domain.Method) (vars string, encode string)
}

// ScriptGenerator generates deployment scripts from templates
type ScriptGenerator interface {
	GenerateScript(ctx context.Context, template *domain.ScriptTemplate) (string, error)
}

// FileWriter handles file system operations for scripts
type FileWriter interface {
	WriteScript(ctx context.Context, path string, content string) error
	FileExists(ctx context.Context, path string) (bool, error)
	EnsureDirectory(ctx context.Context, path string) error
}

// DeploymentSelector handles interactive selection of contracts
type ContractSelector interface {
	SelectContract(ctx context.Context, contracts []*domain.ContractInfo, prompt string) (*domain.ContractInfo, error)
}

// DeploymentSelector handles interactive selection of deployments
type DeploymentSelector interface {
	SelectDeployment(ctx context.Context, deployments []*domain.Deployment, prompt string) (*domain.Deployment, error)
}

// AnvilManager manages local anvil node instances
type AnvilManager interface {
	Start(ctx context.Context, instance *domain.AnvilInstance) error
	Stop(ctx context.Context, instance *domain.AnvilInstance) error
	GetStatus(ctx context.Context, instance *domain.AnvilInstance) (*domain.AnvilStatus, error)
	StreamLogs(ctx context.Context, instance *domain.AnvilInstance, writer io.Writer) error
}

// ContractResolver resolves contract references to actual contracts
type ContractResolver interface {
	ResolveContract(ctx context.Context, query domain.ContractQuery) (*domain.ContractInfo, error)
	GetProxyContracts(ctx context.Context) ([]*domain.ContractInfo, error)
	SelectProxyContract(ctx context.Context) (*domain.ContractInfo, error)
	IsLibrary(ctx context.Context, contract *domain.ContractInfo) (bool, error)
}

// NetworkResolver handles network configuration resolution
type NetworkResolver interface {
	GetNetworks(ctx context.Context) []string
	ResolveNetwork(ctx context.Context, networkName string) (*domain.Network, error)
}

// BlockchainChecker checks on-chain state of contracts and transactions
type BlockchainChecker interface {
	Connect(ctx context.Context, rpcURL string, chainID uint64) error
	CheckDeploymentExists(ctx context.Context, address string) (exists bool, reason string, err error)
	CheckTransactionExists(ctx context.Context, txHash string) (exists bool, blockNumber uint64, reason string, err error)
	CheckSafeContract(ctx context.Context, safeAddress string) (exists bool, reason string, err error)
}

// RegistryPruner handles registry pruning operations
type RegistryPruner interface {
	CollectPrunableItems(ctx context.Context, chainID uint64, includePending bool, checker BlockchainChecker) (*domain.ItemsToPrune, error)
	ExecutePrune(ctx context.Context, items *domain.ItemsToPrune) error
}

// LocalConfigStore manages local configuration persistence
type LocalConfigStore interface {
	Exists() bool
	Load(ctx context.Context) (*domain.LocalConfig, error)
	Save(ctx context.Context, config *domain.LocalConfig) error
	GetPath() string
}

// SafeTransactionStore handles persistence of Safe transactions
type SafeTransactionStore interface {
	GetSafeTransaction(ctx context.Context, safeTxHash string) (*domain.SafeTransaction, error)
	ListSafeTransactions(ctx context.Context, filter domain.SafeTransactionFilter) ([]*domain.SafeTransaction, error)
	SaveSafeTransaction(ctx context.Context, safeTx *domain.SafeTransaction) error
	UpdateSafeTransaction(ctx context.Context, safeTx *domain.SafeTransaction) error
}

// SafeClient handles interactions with Safe multisig contracts
type SafeClient interface {
	SetChain(ctx context.Context, chainID uint64) error
	GetTransactionExecutionInfo(ctx context.Context, safeTxHash string) (*domain.SafeExecutionInfo, error)
	GetTransactionDetails(ctx context.Context, safeTxHash string) (*domain.SafeTransaction, error)
}

// ScriptResolver resolves script paths to script information
type ScriptResolver interface {
	// ResolveScript resolves a script path or name to script info
	ResolveScript(ctx context.Context, scriptRef string) (*domain.ContractInfo, error)
	// GetScriptParameters extracts parameters from a script's artifact
	GetScriptParameters(ctx context.Context, script *domain.ContractInfo) ([]domain.ScriptParameter, error)
}

// Parameter Handling Ports

// ParameterResolver resolves script parameter values
type ParameterResolver interface {
	// ResolveParameters resolves parameter values from various sources
	ResolveParameters(ctx context.Context, params []domain.ScriptParameter, values map[string]string) (map[string]string, error)
	// ValidateParameters validates that all required parameters have values
	ValidateParameters(ctx context.Context, params []domain.ScriptParameter, values map[string]string) error
}

// ParameterPrompter prompts for missing parameter values in interactive mode
type ParameterPrompter interface {
	// PromptForParameters prompts the user for missing parameter values
	PromptForParameters(ctx context.Context, params []domain.ScriptParameter, existing map[string]string) (map[string]string, error)
}

// Script Execution Ports

// ScriptExecutor executes Foundry scripts
type ScriptExecutor interface {
	// Execute runs a script and returns the raw output
	Execute(ctx context.Context, config ScriptExecutionConfig) (*ScriptExecutionOutput, error)
}

// ScriptExecutionConfig contains configuration for script execution
type ScriptExecutionConfig struct {
	Script      *domain.ContractInfo
	Network     *domain.Network
	Namespace   string
	Environment map[string]string // Includes resolved parameters and sender configs
	DryRun      bool
	Debug       bool
	DebugJSON   bool
	Progress    ProgressSink
}

// ScriptExecutionOutput contains the raw output from script execution
type ScriptExecutionOutput struct {
	Success       bool
	RawOutput     []byte
	ParsedOutput  any // Forge-specific parsed output
	JSONOutput    any // Parsed JSON output from forge --json
	BroadcastPath string
}

// Execution Parsing Ports

// ExecutionParser parses script execution output into domain models
type ExecutionParser interface {
	// ParseExecution parses the script output into a structured execution result
	ParseExecution(ctx context.Context, output *ScriptExecutionOutput) (*domain.ScriptExecution, error)
	// EnrichFromBroadcast enriches execution data from broadcast files
	EnrichFromBroadcast(ctx context.Context, execution *domain.ScriptExecution, broadcastPath string) error
}

// Registry Update Ports

// RegistryUpdater updates the deployment registry based on script execution
type RegistryUpdater interface {
	// PrepareUpdates analyzes the execution and prepares registry updates
	PrepareUpdates(ctx context.Context, execution *domain.ScriptExecution) (*RegistryChanges, error)
	// ApplyUpdates applies the prepared changes to the registry
	ApplyUpdates(ctx context.Context, changes *RegistryChanges) error
	// HasChanges returns true if there are any changes to apply
	HasChanges(changes *RegistryChanges) bool
}

// RegistryChanges represents changes to be made to the registry
type RegistryChanges struct {
	Deployments  []*domain.Deployment
	Transactions []*domain.Transaction
	AddedCount   int
	UpdatedCount int
	HasChanges   bool
}

// ExecutionStage represents a stage in the execution process
type ExecutionStage string

const (
	StageResolving    ExecutionStage = "Resolving"
	StageParameters   ExecutionStage = "Parameters"
	StageSimulating   ExecutionStage = "Simulating"
	StageBroadcasting ExecutionStage = "Broadcasting"
	StageParsing      ExecutionStage = "Parsing"
	StageUpdating     ExecutionStage = "Updating"
	StageCompleted    ExecutionStage = "Completed"
)

// Environment Building Ports

// EnvironmentBuilder builds environment variables for script execution
type EnvironmentBuilder interface {
	// BuildEnvironment builds the complete environment for script execution
	BuildEnvironment(ctx context.Context, params BuildEnvironmentParams) (map[string]string, error)
}

// BuildEnvironmentParams contains parameters for building the environment
type BuildEnvironmentParams struct {
	Network           string
	Namespace         string
	Parameters        map[string]string
	TrebConfig        *domain.TrebConfig // From RuntimeConfig
	DryRun            bool
	DeployedLibraries []LibraryReference
}

// LibraryReference represents a deployed library
type LibraryReference struct {
	Path    string
	Name    string
	Address string
}

// LibraryResolver resolves deployed libraries for a namespace/network
type LibraryResolver interface {
	// GetDeployedLibraries gets all deployed libraries for the given context
	GetDeployedLibraries(ctx context.Context, namespace string, chainID uint64) ([]LibraryReference, error)
}

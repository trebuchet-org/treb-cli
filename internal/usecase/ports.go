package usecase

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// DeploymentStore handles persistence of deployments
type DeploymentRepository interface {
	GetDeployment(ctx context.Context, id string) (*models.Deployment, error)
	GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*models.Deployment, error)
	ListDeployments(ctx context.Context, filter domain.DeploymentFilter) ([]*models.Deployment, error)
	GetAllDeployments(ctx context.Context) []*models.Deployment
	SaveDeployment(ctx context.Context, deployment *models.Deployment) error
	DeleteDeployment(ctx context.Context, id string) error
	GetTransaction(ctx context.Context, id string) (*models.Transaction, error)
	ListTransactions(ctx context.Context, filter domain.TransactionFilter) ([]*models.Transaction, error)
	GetAllTransactions(ctx context.Context) map[string]*models.Transaction
	SaveTransaction(ctx context.Context, transaction *models.Transaction) error
	GetSafeTransaction(ctx context.Context, safeTxHash string) (*models.SafeTransaction, error)
	ListSafeTransactions(ctx context.Context, filter domain.SafeTransactionFilter) ([]*models.SafeTransaction, error)
	SaveSafeTransaction(ctx context.Context, safeTx *models.SafeTransaction) error
	UpdateSafeTransaction(ctx context.Context, safeTx *models.SafeTransaction) error
	GetAllSafeTransactions(ctx context.Context) map[string]*models.SafeTransaction
}

// ContractIndexer provides access to compiled contracts
type ContractRepository interface {
	GetContract(ctx context.Context, key string) (*models.Contract, error)
	SearchContracts(ctx context.Context, query domain.ContractQuery) []*models.Contract
	GetContractByArtifact(ctx context.Context, artifact string) *models.Contract
}

// ContractVerifier handles contract verification
type ContractVerifier interface {
	Verify(ctx context.Context, deployment *models.Deployment, network *config.Network) error
	GetVerificationStatus(ctx context.Context, deployment *models.Deployment) (*models.VerificationInfo, error)
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
	Deployments []*models.Deployment
	Summary     DeploymentSummary
}

// DeploymentSummary provides summary statistics
type DeploymentSummary struct {
	Total       int
	ByNamespace map[string]int
	ByChain     map[uint64]int
	ByType      map[models.DeploymentType]int
}

// ScriptExecutionResult contains the result of script execution
type ScriptExecutionResult struct {
	Success      bool
	Deployments  []*models.Deployment
	Transactions []*models.Transaction
	Logs         []string
	GasUsed      uint64
	Error        error
}

// ABIParser parses contract ABIs to extract constructor/initializer info
type ABIParser interface {
	FindInitializeMethod(abi *abi.ABI) *abi.Method
	GenerateConstructorArgs(abi *abi.ABI) (vars string, encode string)
	GenerateInitializerArgs(method *abi.Method) (vars, encode, sig string)
	ParseEvent(rawLog *forge.EventLog) (any, error)
	ParseEvents(rawLog *forge.ScriptOutput) ([]any, error)
}

// ABI Resolver looks-up ABIs in different sources and by different input values.
type ABIResolver interface {
	Get(ctx context.Context, artifact *models.Artifact) (*abi.ABI, error)
	FindByRef(ctx context.Context, contractRef string) (*abi.ABI, error)
	FindByAddress(ctx context.Context, address common.Address) (*abi.ABI, error)
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
	SelectContract(ctx context.Context, contracts []*models.Contract, prompt string) (*models.Contract, error)
}

// DeploymentSelector handles interactive selection of deployments
type DeploymentSelector interface {
	SelectDeployment(ctx context.Context, deployments []*models.Deployment, prompt string) (*models.Deployment, error)
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
	ResolveContract(ctx context.Context, query domain.ContractQuery) (*models.Contract, error)
	GetProxyContracts(ctx context.Context) ([]*models.Contract, error)
	SelectProxyContract(ctx context.Context) (*models.Contract, error)
	IsLibrary(ctx context.Context, contract *models.Contract) (bool, error)
}

// NetworkResolver handles network configuration resolution
type NetworkResolver interface {
	GetNetworks(ctx context.Context) []string
	ResolveNetwork(ctx context.Context, networkName string) (*config.Network, error)
}

// BlockchainChecker checks on-chain state of contracts and transactions
type BlockchainChecker interface {
	Connect(ctx context.Context, rpcURL string, chainID uint64) error
	CheckDeploymentExists(ctx context.Context, address string) (exists bool, reason string, err error)
	CheckTransactionExists(ctx context.Context, txHash string) (exists bool, blockNumber uint64, reason string, err error)
	CheckSafeContract(ctx context.Context, safeAddress string) (exists bool, reason string, err error)
}

type DeploymentRepositoryPruner interface {
	CollectPrunableItems(ctx context.Context, chainID uint64, includePending bool) (*models.Changeset, error)
}

// LocalConfigRepository manages local configuration persistence
type LocalConfigRepository interface {
	Exists() bool
	Load(ctx context.Context) (*config.LocalConfig, error)
	Save(ctx context.Context, config *config.LocalConfig) error
	GetPath() string
}

// SafeClient handles interactions with Safe multisig contracts
type SafeClient interface {
	GetTransactionExecutionInfo(ctx context.Context, safeTxHash string) (*models.SafeExecutionInfo, error)
}

// ScriptResolver resolves script paths to script information
type ScriptResolver interface {
	// ResolveScript resolves a script path or name to script info
	ResolveScript(ctx context.Context, scriptRef string) (*models.Contract, error)
	// GetScriptParameters extracts parameters from a script's artifact
	GetScriptParameters(ctx context.Context, script *models.Contract) ([]domain.ScriptParameter, error)
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
type ForgeScriptRunner interface {
	// Execute runs a script and returns the raw output
	RunScript(ctx context.Context, config RunScriptConfig) (*forge.RunResult, error)
}

// ScriptExecutionConfig contains configuration for script execution
type RunScriptConfig struct {
	Script             *models.Contract
	Network            *config.Network
	Namespace          string
	Parameters         map[string]string // Includes resolved parameters and sender configs
	DryRun             bool
	Debug              bool
	DebugJSON          bool
	Libraries          []string
	SenderScriptConfig config.SenderScriptConfig
	Progress           ProgressSink
}

// RunResultHydrator hydrated RunResults with domain models.
type RunResultHydrator interface {
	// ParseExecution parses the script output into a structured execution result
	Hydrate(ctx context.Context, output *forge.RunResult) (*forge.HydratedRunResult, error)
}

// RegistryUpdater updates the deployment registry based on script execution
type DeploymentRepositoryUpdater interface {
	// PrepareUpdates analyzes the execution and prepares registry updates
	BuildChangesetFromRunResult(ctx context.Context, execution *forge.HydratedRunResult) (*models.Changeset, error)
	// ApplyUpdates applies the prepared changes to the registry
	ApplyChangeset(ctx context.Context, changeset *models.Changeset) error
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

// LibraryResolver resolves deployed libraries for a namespace/network
type LibraryResolver interface {
	// GetDeployedLibraries gets all deployed libraries for the given context
	GetDeployedLibraries(ctx context.Context, namespace string, chainID uint64) ([]LibraryReference, error)
}

// LibraryReference represents a deployed library
type LibraryReference struct {
	Path    string
	Name    string
	Address string
}

type SendersManager interface {
	BuildSenderScriptConfig(script *models.Artifact) (*config.SenderScriptConfig, error)
}

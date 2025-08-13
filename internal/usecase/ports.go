package usecase

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// DeploymentStore handles persistence of deployments
type DeploymentStore interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
	GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*domain.Deployment, error)
	ListDeployments(ctx context.Context, filter DeploymentFilter) ([]*domain.Deployment, error)
	SaveDeployment(ctx context.Context, deployment *domain.Deployment) error
	DeleteDeployment(ctx context.Context, id string) error
}

// DeploymentFilter defines filtering options for deployments
type DeploymentFilter struct {
	Namespace    string
	ChainID      uint64
	ContractName string
	Label        string
	Type         domain.DeploymentType
}

// ContractIndexer provides access to compiled contracts
type ContractIndexer interface {
	GetContract(ctx context.Context, key string) (*domain.ContractInfo, error)
	SearchContracts(ctx context.Context, pattern string) []*domain.ContractInfo
	GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo
	RefreshIndex(ctx context.Context) error
}

// ForgeExecutor handles forge command execution
type ForgeExecutor interface {
	Build(ctx context.Context) error
	RunScript(ctx context.Context, config ScriptConfig) (*ScriptResult, error)
}

// ScriptConfig contains configuration for script execution
type ScriptConfig struct {
	Path        string
	Network     string
	Environment map[string]string
	DryRun      bool
	Debug       bool
	Sender      string
	Args        []string
}

// ScriptResult contains the result of script execution
type ScriptResult struct {
	Success    bool
	Output     string
	Broadcasts []string
	Error      error
}


// ContractVerifier handles contract verification
type ContractVerifier interface {
	Verify(ctx context.Context, deployment *domain.Deployment, network *domain.NetworkInfo) error
	GetVerificationStatus(ctx context.Context, deployment *domain.Deployment) (*domain.VerificationInfo, error)
}

// ConfigManager handles configuration management
type ConfigManager interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) (map[string]string, error)
}

// Progress tracking interfaces

// ProgressEvent represents a progress update
type ProgressEvent struct {
	Stage   string
	Current int
	Total   int
	Message string
	Spinner bool
}

// ProgressSink receives progress events
type ProgressSink interface {
	OnProgress(ctx context.Context, event ProgressEvent)
}

// NopProgress is a no-op implementation of ProgressSink
type NopProgress struct{}

func (NopProgress) OnProgress(context.Context, ProgressEvent) {}

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
package usecase

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Script Resolution Ports

// ScriptResolver resolves script paths to script information
type ScriptResolver interface {
	// ResolveScript resolves a script path or name to script info
	ResolveScript(ctx context.Context, pathOrName string) (*domain.ScriptInfo, error)
	// GetScriptParameters extracts parameters from a script's artifact
	GetScriptParameters(ctx context.Context, script *domain.ScriptInfo) ([]domain.ScriptParameter, error)
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
	Script      *domain.ScriptInfo
	Network     string
	NetworkInfo *domain.NetworkInfo
	Namespace   string
	Environment map[string]string // Includes resolved parameters and sender configs
	DryRun      bool
	Debug       bool
	DebugJSON   bool
}

// ScriptExecutionOutput contains the raw output from script execution
type ScriptExecutionOutput struct {
	Success       bool
	RawOutput     []byte
	ParsedOutput  interface{} // Forge-specific parsed output
	BroadcastPath string
}

// Execution Parsing Ports

// ExecutionParser parses script execution output into domain models
type ExecutionParser interface {
	// ParseExecution parses the script output into a structured execution result
	ParseExecution(ctx context.Context, output *ScriptExecutionOutput, network string, chainID uint64) (*domain.ScriptExecution, error)
	// EnrichFromBroadcast enriches execution data from broadcast files
	EnrichFromBroadcast(ctx context.Context, execution *domain.ScriptExecution, broadcastPath string) error
}

// Registry Update Ports

// RegistryUpdater updates the deployment registry based on script execution
type RegistryUpdater interface {
	// PrepareUpdates analyzes the execution and prepares registry updates
	PrepareUpdates(ctx context.Context, execution *domain.ScriptExecution, namespace string, network string) (*RegistryChanges, error)
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

// Progress Reporting Ports

// ProgressReporter reports progress during script execution
type ProgressReporter interface {
	// ReportStage reports the current execution stage
	ReportStage(ctx context.Context, stage ExecutionStage)
	// ReportProgress reports progress within a stage
	ReportProgress(ctx context.Context, event ProgressEvent)
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
	Network          string
	Namespace        string
	Parameters       map[string]string
	TrebConfig       *domain.TrebConfig // From RuntimeConfig
	DryRun           bool
	DeployedLibraries []LibraryReference
}

// LibraryReference represents a deployed library
type LibraryReference struct {
	Path     string
	Name     string
	Address  string
}


// Library Resolution Ports

// LibraryResolver resolves deployed libraries for a namespace/network
type LibraryResolver interface {
	// GetDeployedLibraries gets all deployed libraries for the given context
	GetDeployedLibraries(ctx context.Context, namespace string, chainID uint64) ([]LibraryReference, error)
}
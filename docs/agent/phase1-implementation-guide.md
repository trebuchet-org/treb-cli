# Phase 1 Implementation Guide

## Step-by-Step Implementation for Domain & Infrastructure Setup

### 1. Create Directory Structure

```bash
mkdir -p internal/{app,domain,usecase,adapters/{fs,forge,config,verification,safe},cli/{render,interactive}}
```

### 2. Domain Types Migration

Create `internal/domain/types.go`:

```go
package domain

import (
    "time"
    "github.com/ethereum/go-ethereum/common"
)

// Core deployment types (migrated from pkg/types)
type Deployment struct {
    ID                  string
    Namespace          string
    ChainID            uint64
    ContractName       string
    Address            string
    DeploymentType     DeploymentType
    Artifact           string
    Transaction        Transaction
    Verification       VerificationInfo
    Tags               []string
    Label              string
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type DeploymentType string

const (
    SingletonDeployment DeploymentType = "singleton"
    ProxyDeployment     DeploymentType = "proxy"
)

type Transaction struct {
    Hash       string
    Status     TransactionStatus
    BlockNumber uint64
    GasUsed     uint64
}

type TransactionStatus string

const (
    TransactionStatusPending   TransactionStatus = "pending"
    TransactionStatusQueued    TransactionStatus = "queued"
    TransactionStatusSimulated TransactionStatus = "simulated"
    TransactionStatusExecuted  TransactionStatus = "executed"
)
```

### 3. Port Interfaces Definition

Create `internal/usecase/ports.go`:

```go
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

type DeploymentFilter struct {
    Namespace    string
    ChainID      uint64
    ContractName string
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

type ScriptConfig struct {
    Path         string
    Network      string
    Environment  map[string]string
    DryRun       bool
    Debug        bool
}

type ScriptResult struct {
    Success    bool
    Output     string
    Broadcasts []string
}
```

### 4. Adapter Implementations

Create `internal/adapters/fs/registry_store.go`:

```go
package fs

import (
    "context"
    "github.com/trebuchet-org/treb-cli/internal/domain"
    "github.com/trebuchet-org/treb-cli/internal/usecase"
    "github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

// RegistryStoreAdapter wraps the existing registry.Manager
type RegistryStoreAdapter struct {
    manager *registry.Manager
}

func NewRegistryStoreAdapter(rootDir string) (*RegistryStoreAdapter, error) {
    manager, err := registry.NewManager(rootDir)
    if err != nil {
        return nil, err
    }
    return &RegistryStoreAdapter{manager: manager}, nil
}

// Implement DeploymentStore interface
func (r *RegistryStoreAdapter) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
    // Convert from existing types
    dep, err := r.manager.GetDeployment(id)
    if err != nil {
        return nil, err
    }
    return convertToDoaminDeployment(dep), nil
}

func (r *RegistryStoreAdapter) ListDeployments(ctx context.Context, filter usecase.DeploymentFilter) ([]*domain.Deployment, error) {
    // Get all deployments and filter
    allDeps := r.manager.GetAllDeploymentsHydrated()
    
    var result []*domain.Deployment
    for _, dep := range allDeps {
        if matchesFilter(dep, filter) {
            result = append(result, convertToDomainDeployment(dep))
        }
    }
    return result, nil
}

// ... implement other methods
```

Create `internal/adapters/forge/executor.go`:

```go
package forge

import (
    "context"
    "github.com/trebuchet-org/treb-cli/internal/usecase"
    "github.com/trebuchet-org/treb-cli/cli/pkg/forge"
)

type ForgeExecutorAdapter struct {
    forge *forge.Forge
}

func NewForgeExecutorAdapter(projectRoot string) *ForgeExecutorAdapter {
    return &ForgeExecutorAdapter{
        forge: forge.NewForge(projectRoot),
    }
}

func (f *ForgeExecutorAdapter) Build(ctx context.Context) error {
    return f.forge.Build()
}

func (f *ForgeExecutorAdapter) RunScript(ctx context.Context, config usecase.ScriptConfig) (*usecase.ScriptResult, error) {
    // Convert config to forge flags
    var flags []string
    if config.Network != "" {
        flags = append(flags, "--fork-url", config.Network)
    }
    if config.DryRun {
        flags = append(flags, "--simulate")
    }
    
    output, err := f.forge.RunScript(config.Path, flags, config.Environment)
    
    return &usecase.ScriptResult{
        Success: err == nil,
        Output:  output,
    }, err
}
```

### 5. Wire Setup

Create `internal/adapters/providers.go`:

```go
package adapters

import (
    "github.com/google/wire"
    "github.com/trebuchet-org/treb-cli/internal/adapters/fs"
    "github.com/trebuchet-org/treb-cli/internal/adapters/forge"
    "github.com/trebuchet-org/treb-cli/internal/usecase"
)

// FSSet provides filesystem-based implementations
var FSSet = wire.NewSet(
    fs.NewRegistryStoreAdapter,
    wire.Bind(new(usecase.DeploymentStore), new(*fs.RegistryStoreAdapter)),
    
    fs.NewContractIndexerAdapter,
    wire.Bind(new(usecase.ContractIndexer), new(*fs.ContractIndexerAdapter)),
)

// ForgeSet provides forge-based implementations
var ForgeSet = wire.NewSet(
    forge.NewForgeExecutorAdapter,
    wire.Bind(new(usecase.ForgeExecutor), new(*forge.ForgeExecutorAdapter)),
)
```

Create `internal/app/wire.go`:

```go
//go:build wireinject
// +build wireinject

package app

import (
    "github.com/google/wire"
    "github.com/trebuchet-org/treb-cli/internal/adapters"
    "github.com/trebuchet-org/treb-cli/internal/usecase"
)

func InitApp(cfg Config, sink usecase.ProgressSink) (*App, error) {
    wire.Build(
        wire.FieldsOf(new(Config), "ProjectRoot", "Network"),
        
        adapters.FSSet,
        adapters.ForgeSet,
        
        // Pass through progress sink
        func(s usecase.ProgressSink) usecase.ProgressSink { return s },
        
        // Use cases
        usecase.NewListDeployments,
        usecase.NewShowDeployment,
        
        NewApp,
    )
    return nil, nil
}
```

### 6. Sample Use Case Implementation

Create `internal/usecase/list_deployments.go`:

```go
package usecase

import (
    "context"
    "sort"
    "github.com/trebuchet-org/treb-cli/internal/domain"
)

type ListDeploymentsParams struct {
    Namespace    string
    ChainID      uint64
    ContractName string
}

type ListDeployments struct {
    store DeploymentStore
    sink  ProgressSink
}

func NewListDeployments(store DeploymentStore, sink ProgressSink) *ListDeployments {
    return &ListDeployments{
        store: store,
        sink:  sink,
    }
}

func (uc *ListDeployments) Run(ctx context.Context, params ListDeploymentsParams) ([]*domain.Deployment, error) {
    uc.sink.OnProgress(ctx, ProgressEvent{
        Stage:   "loading",
        Message: "Loading deployments from registry",
    })
    
    filter := DeploymentFilter{
        Namespace:    params.Namespace,
        ChainID:      params.ChainID,
        ContractName: params.ContractName,
    }
    
    deployments, err := uc.store.ListDeployments(ctx, filter)
    if err != nil {
        return nil, err
    }
    
    // Sort by namespace, chain, contract name, label
    sort.Slice(deployments, func(i, j int) bool {
        if deployments[i].Namespace != deployments[j].Namespace {
            return deployments[i].Namespace < deployments[j].Namespace
        }
        if deployments[i].ChainID != deployments[j].ChainID {
            return deployments[i].ChainID < deployments[j].ChainID
        }
        if deployments[i].ContractName != deployments[j].ContractName {
            return deployments[i].ContractName < deployments[j].ContractName
        }
        return deployments[i].Label < deployments[j].Label
    })
    
    uc.sink.OnProgress(ctx, ProgressEvent{
        Stage:   "complete",
        Current: len(deployments),
        Total:   len(deployments),
        Message: "Deployments loaded",
    })
    
    return deployments, nil
}
```

### 7. CLI Command Migration Example

Update `internal/cli/list.go` (new file):

```go
package cli

import (
    "context"
    "github.com/spf13/cobra"
    "github.com/trebuchet-org/treb-cli/internal/app"
    "github.com/trebuchet-org/treb-cli/internal/usecase"
    "github.com/trebuchet-org/treb-cli/internal/cli/render"
)

func NewListCmd(baseCfg *app.Config) *cobra.Command {
    var (
        namespace    string
        chainID      uint64
        contractName string
    )
    
    cmd := &cobra.Command{
        Use:     "list",
        Aliases: []string{"ls"},
        Short:   "List deployments from registry",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Use NOP progress for now (preserves exact output)
            progressSink := usecase.NopProgress{}
            
            // Initialize app with Wire
            app, err := app.InitApp(*baseCfg, progressSink)
            if err != nil {
                return err
            }
            
            // Run use case
            params := usecase.ListDeploymentsParams{
                Namespace:    namespace,
                ChainID:      chainID,
                ContractName: contractName,
            }
            
            deployments, err := app.ListDeployments.Run(context.Background(), params)
            if err != nil {
                return err
            }
            
            // Render output (preserve existing format exactly)
            renderer := render.NewTableRenderer(cmd.OutOrStdout())
            return renderer.RenderDeployments(deployments)
        },
    }
    
    // Flags
    cmd.Flags().StringVar(&namespace, "namespace", "", "Filter by namespace")
    cmd.Flags().Uint64Var(&chainID, "chain", 0, "Filter by chain ID")
    cmd.Flags().StringVar(&contractName, "contract", "", "Filter by contract name")
    
    return cmd
}
```

### 8. Backwards Compatibility Bridge

In the existing `cmd/list.go`, update to use new implementation:

```go
func init() {
    // Feature flag to switch implementations
    if os.Getenv("TREB_NEW_ARCH") == "true" {
        // Use new implementation
        cfg := app.LoadConfig()
        rootCmd.AddCommand(cli.NewListCmd(&cfg))
    } else {
        // Keep existing implementation
        rootCmd.AddCommand(listCmd)
    }
}
```

## Testing the Migration

### 1. Integration Test

```go
func TestListCommandCompatibility(t *testing.T) {
    // Capture output from old implementation
    oldOutput := captureOutput(t, []string{"list", "--namespace", "test"})
    
    // Enable new architecture
    os.Setenv("TREB_NEW_ARCH", "true")
    defer os.Unsetenv("TREB_NEW_ARCH")
    
    // Capture output from new implementation  
    newOutput := captureOutput(t, []string{"list", "--namespace", "test"})
    
    // Compare outputs
    assert.Equal(t, oldOutput, newOutput, "Output should be identical")
}
```

### 2. Adapter Tests

```go
func TestRegistryStoreAdapter(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()
    
    // Initialize adapter
    adapter, err := fs.NewRegistryStoreAdapter(tmpDir)
    require.NoError(t, err)
    
    // Test operations
    ctx := context.Background()
    
    // List should return empty initially
    deps, err := adapter.ListDeployments(ctx, usecase.DeploymentFilter{})
    assert.NoError(t, err)
    assert.Empty(t, deps)
}
```

## Rollout Strategy

1. **Development Testing:**
   - Run with `TREB_NEW_ARCH=true` locally
   - Compare outputs for all commands
   - Fix any discrepancies

2. **CI Integration:**
   - Add parallel test jobs (old vs new)
   - Gate on 100% compatibility

3. **Gradual Rollout:**
   - Ship with feature flag disabled
   - Enable for beta users first
   - Monitor for issues
   - Full rollout after validation

## Next Steps

After Phase 1 is complete and tested:
1. Begin Phase 2 with read-only commands
2. Add comprehensive integration tests
3. Set up benchmarking to ensure no performance regression
4. Document any behavior changes needed
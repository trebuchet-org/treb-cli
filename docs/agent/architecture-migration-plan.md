# Treb CLI Architecture Migration Plan

## Current State Analysis

### CLI Commands Overview

The treb CLI currently has the following commands:

**Main Commands:**
- `run` - Execute Foundry scripts with treb infrastructure
- `list` - Display deployments from registry  
- `show` - Show detailed deployment information
- `generate` - Generate deployment scripts and proxies
- `verify` - Verify deployed contracts on explorers
- `init` - Initialize new treb project

**Management Commands:**
- `config` - Manage treb configuration
- `sync` - Sync registry with on-chain state
- `tag` - Tag deployments with versions/labels
- `compose` - Execute orchestrated deployments from YAML configuration
- `register` - Register an existing contract deployment in the registry
- `prune` - Prune registry entries that no longer exist on-chain
- `reset` - Reset all registry entries for the current namespace and network
- `networks` - List available networks
- `dev` - Development tools (anvil management)
- `version` - Show version information

### Current Service Architecture

The current codebase has the following major service packages:

#### Core Services:
1. **Registry Management** (`pkg/registry/`)
   - `Manager` - Handles deployment registry CRUD operations
   - `ScriptExecutionUpdater` - Updates registry from script execution
   - `Pruner` - Removes deployments

2. **Script Execution** (`pkg/script/`)
   - `runner.ScriptRunner` - Orchestrates script execution
   - `executor.Executor` - Executes forge scripts
   - `parser.Parser` - Parses script execution results
   - `parameters` - Handles script parameter resolution
   - `display` - Presentation layer for script output
   - `senders` - Manages transaction senders

3. **Configuration** (`pkg/config/`)
   - `Manager` - Handles .treb/config.json
   - `TrebConfig` - Foundry.toml deployment configuration
   - `SenderConfigs` - Transaction sender configurations
   - Environment variable resolution

4. **Contract Management** (`pkg/contracts/`)
   - `Indexer` - Indexes compiled contracts
   - `Generator` - Generates deployment scripts
   - `Resolver` - Resolves contract references

5. **Network Management** (`pkg/network/`)
   - `Resolver` - Resolves network configurations
   - Explorer helpers for verification URLs

6. **Verification** (`pkg/verification/`)
   - `Manager` - Handles contract verification

7. **Forge Integration** (`pkg/forge/`)
   - `Forge` - Wrapper for forge commands
   - Script output processing

8. **Interactive Components** (`pkg/interactive/`)
   - Contract/deployment pickers
   - Fuzzy search functionality

### Key Interfaces and Dependencies

Current interfaces found:
- `types.DeploymentLookup` - Registry lookup interface
- `types.ContractLookup` - Contract indexing interface
- `abi.ABIResolver` - ABI resolution for decoding
- Various informal interfaces in services

### Mapping to New Architecture

## Service Layer Mapping

### Domain Layer (`internal/domain/`)
**Current → New:**
- `pkg/types/*.go` → `domain/types.go`
  - Deployment, Transaction, Contract types
  - Verification status enums
  - Registry data structures

### Use Cases Layer (`internal/usecase/`)
**Current Commands → Use Cases:**
- `cmd/run.go` → `usecase/run_script.go`
- `cmd/list.go` → `usecase/list_deployments.go`
- `cmd/show.go` → `usecase/show_deployment.go`
- `cmd/generate.go` → `usecase/generate_deployment.go`
- `cmd/verify.go` → `usecase/verify_deployment.go`
- `cmd/init.go` → `usecase/init_project.go`
- `cmd/config.go` → `usecase/manage_config.go`
- `cmd/sync.go` → `usecase/sync_registry.go`
- `cmd/tag.go` → `usecase/tag_deployment.go`
- `cmd/compose.go` → `usecase/compose_deployment.go`

### Ports (Interfaces) (`internal/usecase/ports.go`)
```go
// Registry operations
type DeploymentStore interface {
    GetDeployment(id string) (*domain.Deployment, error)
    ListDeployments(filter DeploymentFilter) ([]*domain.Deployment, error)
    SaveDeployment(d *domain.Deployment) error
    DeleteDeployment(id string) error
}

// Contract operations
type ContractIndexer interface {
    GetContract(key string) (*domain.ContractInfo, error)
    SearchContracts(pattern string) []*domain.ContractInfo
    GetContractByArtifact(artifact string) *domain.ContractInfo
}

// Forge operations
type ForgeExecutor interface {
    Build() error
    RunScript(path string, config ScriptConfig) (string, error)
}

// Network operations
type NetworkResolver interface {
    ResolveNetwork(name string) (*domain.NetworkInfo, error)
    GetPreferredNetwork(chainID uint64) (string, error)
}

// Verification operations
type ContractVerifier interface {
    Verify(deployment *domain.Deployment, network *domain.NetworkInfo) error
}

// Progress reporting
type ProgressSink interface {
    OnProgress(ctx context.Context, event ProgressEvent)
}

// Presentation interfaces (for use case results)
type DeploymentListResult struct {
    Deployments []*domain.Deployment
    Summary     DeploymentSummary
}

type DeploymentSummary struct {
    Total       int
    ByNamespace map[string]int
    ByChain     map[uint64]int
    ByType      map[domain.DeploymentType]int
}

type ScriptExecutionResult struct {
    Success      bool
    Deployments  []*domain.Deployment
    Transactions []*domain.Transaction
    Logs         []string
    GasUsed      uint64
}
```

### Adapters Layer (`internal/adapters/`)
**Current → New:**
- `pkg/registry/` → `adapters/fs/registry_store.go`
- `pkg/contracts/` → `adapters/fs/contract_indexer.go`
- `pkg/forge/` → `adapters/forge/executor.go`
- `pkg/network/` → `adapters/config/network_resolver.go`
- `pkg/verification/` → `adapters/verification/verifier.go`
- `pkg/safe/` → `adapters/safe/client.go`

### CLI Layer (`internal/cli/`)
**Current → New:**
- `cmd/*.go` → `cli/*.go` (thin command handlers)
- `pkg/script/display/` → `cli/render/` (presentation)
- `pkg/interactive/` → `cli/interactive/` (UI components)

### Presentation Layer Architecture
To enable future `--json` flag support without changing use cases:

```go
// internal/cli/render/interfaces.go
type Renderer interface {
    RenderDeploymentList(result *usecase.DeploymentListResult) error
    RenderDeployment(deployment *domain.Deployment) error
    RenderScriptExecution(result *usecase.ScriptExecutionResult) error
    RenderError(err error) error
    RenderProgress(event usecase.ProgressEvent) error
}

// TableRenderer - current UI implementation
type TableRenderer struct {
    out io.Writer
    color bool
}

// Future: JSONRenderer for --json flag
type JSONRenderer struct {
    out io.Writer
    pretty bool
}
```

## Phased Migration Plan

### Phase 0: Integration Testing Infrastructure
**Goal:** Set up comprehensive testing framework before migration begins

1. **Output Fixture System:**
   - Create `testdata/fixtures/` directory structure
   - Implement golden file testing utilities
   - Record current CLI output for all commands
   - Set up diff visualization tools

2. **Test Harness:**
   ```go
   type CLITest struct {
       Name     string
       Args     []string
       Env      map[string]string
       Fixture  string // path to golden file
       Setup    func() error
       Teardown func() error
   }
   ```

3. **Compatibility Test Suite:**
   - Full command coverage tests
   - Multi-command workflow tests
   - Error scenario tests
   - Interactive mode tests

4. **CI Integration:**
   - Automated fixture updates
   - Regression detection
   - Performance benchmarking

**Deliverables:**
- Complete test coverage of existing CLI
- Golden files for all command outputs
- Automated compatibility checking

### Phase 1: Domain & Infrastructure Setup
**Goal:** Establish new structure without breaking existing functionality

1. Create new directory structure under `internal/`
2. Copy and adapt domain types from `pkg/types/`
3. Define port interfaces in `internal/usecase/ports.go`
4. Create adapter wrappers around existing services
5. Set up Wire for dependency injection

**Backwards Compatibility:**
- Keep existing `pkg/` structure intact
- Adapters delegate to existing implementations
- No changes to CLI commands yet

### Phase 2: Use Case Migration (Commands in Groups)
**Goal:** Migrate commands to use cases incrementally with decoupled presentation

**Group 1 - Read Operations:**
- `list` → `ListDeployments` use case
- `show` → `ShowDeployment` use case  
- `networks` → `ListNetworks` use case

**Key Changes:**
- Use cases return structured results (not strings)
- Commands create appropriate renderer
- Renderer handles all formatting/display

**Example Migration Pattern:**
```go
// Old: cmd/list.go
func listCommand() {
    deployments := getDeployments()
    // Direct formatting in command
    fmt.Printf("Found %d deployments\n", len(deployments))
}

// New: cli/list.go
func listCommand() {
    result := app.ListDeployments.Run(ctx, params)
    renderer := render.NewTableRenderer(cmd.OutOrStdout())
    return renderer.RenderDeploymentList(result)
}
```

**Group 2 - Project Management:**
- `init` → `InitProject` use case
- `config` → `ManageConfig` use case

**Group 3 - Deployment Operations:**
- `run` → `RunScript` use case
- `generate` → `GenerateDeployment` use case

**Group 4 - Post-Deployment:**
- `verify` → `VerifyDeployment` use case
- `tag` → `TagDeployment` use case
- `sync` → `SyncRegistry` use case

**Backwards Compatibility:**
- Each command updated individually
- Old command code calls new use case
- Presentation layer preserved exactly through TableRenderer

### Phase 3: Presentation Layer Consolidation
**Goal:** Complete separation of presentation from business logic

1. **Renderer Implementation:**
   - Extract all formatting logic to renderers
   - Create renderer factory based on output preferences
   - Ensure zero formatting in use cases

2. **Structured Results:**
   ```go
   // All use cases return structured data
   type VerificationResult struct {
       Deployment   *domain.Deployment
       Verifiers    map[string]VerificationStatus
       Success      bool
       ErrorDetails []string
   }
   ```

3. **Progress Handling:**
   - ProgressRenderer for CLI spinners/bars
   - JSONProgressRenderer for machine-readable progress
   - NopProgressRenderer for quiet mode

4. **Error Presentation:**
   - Structured error types with codes
   - Renderer decides how to display errors
   - Consistent error formatting across commands

**Backwards Compatibility:**
- TableRenderer produces exact same output
- No visible changes to users
- Foundation laid for future --json flag

### Phase 4: Service Consolidation
**Goal:** Remove duplicate code, consolidate services

1. Merge overlapping functionality
2. Remove old `pkg/` implementations
3. Update imports throughout
4. Clean up unused code

### Phase 5: Advanced Features
**Goal:** Leverage new architecture for improvements

1. Add transaction middleware for auditing
2. Implement caching layers
3. Add plugin support
4. Enhanced testing infrastructure

## Integration Testing (Phase 0 Details)

### Output Fixtures Strategy

1. **Golden File Structure:**
   ```
   testdata/fixtures/
   ├── commands/
   │   ├── list/
   │   │   ├── default.golden
   │   │   ├── with_filters.golden
   │   │   └── empty_result.golden
   │   ├── run/
   │   │   ├── successful_deployment.golden
   │   │   ├── dry_run.golden
   │   │   └── with_errors.golden
   │   └── show/
   │       ├── single_deployment.golden
   │       └── with_verification.golden
   └── workflows/
       ├── full_deployment_flow.golden
       └── init_and_deploy.golden
   ```

2. **Test Framework:**
   ```go
   type CLITestCase struct {
       Name        string
       Command     []string
       Env         map[string]string
       WorkDir     string
       Setup       func(t *testing.T) error
       Stdin       string
       GoldenFile  string
       // For dynamic content
       Normalizers []OutputNormalizer
   }

   type OutputNormalizer interface {
       Normalize(output string) string
   }

   // Example normalizers
   type TimestampNormalizer struct{}
   type AddressNormalizer struct{}
   type HashNormalizer struct{}
   ```

3. **Output Verification Framework:**
   ```go
   func RunCLITest(t *testing.T, tc CLITestCase) {
       // Set up test environment
       output := captureOutput(tc.Command, tc.Env)
       
       // Normalize dynamic content
       for _, n := range tc.Normalizers {
           output = n.Normalize(output)
       }
       
       // Compare with golden file
       golden := readGoldenFile(tc.GoldenFile)
       if diff := cmp.Diff(golden, output); diff != "" {
           t.Errorf("Output mismatch (-want +got):\n%s", diff)
       }
   }
   ```

4. **Fixture Management:**
   ```go
   // Update golden files when needed
   func UpdateGoldenFiles() {
       if os.Getenv("UPDATE_GOLDEN") != "true" {
           return
       }
       // Re-run tests and save outputs as new golden files
   }
   ```

### Backwards Compatibility Testing

1. **Regression Test Suite:**
   - Record current CLI output for all commands
   - Run after each phase to ensure compatibility
   - Flag any output differences

2. **Feature Toggle Testing:**
   - Test both old and new code paths
   - Ensure consistent behavior
   - Gradual migration validation

3. **Integration Test Scenarios:**
   - Full deployment workflow
   - Multi-command sequences
   - Error handling paths
   - Interactive mode testing

## Key Considerations

### Critical Areas for Backwards Compatibility

1. **Registry File Format**
   - Must maintain exact JSON structure
   - Version migrations if needed
   - Backup before modifications

2. **CLI Output Format**
   - Table layouts must be preserved
   - Color codes and styling
   - Progress indicators
   - Error message format

3. **Script Execution**
   - Environment variable handling
   - Parameter resolution
   - Forge command construction
   - Output parsing

4. **Configuration Files**
   - foundry.toml reading
   - .env file handling
   - Context management

### Risk Mitigation

1. **Parallel Implementation:**
   - New code alongside old
   - Feature flags for switching
   - Gradual rollout

2. **Comprehensive Testing:**
   - Unit tests for each layer
   - Integration tests for workflows
   - E2E tests with real contracts

3. **Rollback Strategy:**
   - Git tags at each phase
   - Quick revert procedures
   - User communication plan

## Benefits of New Architecture

1. **Testability:**
   - Pure use cases without I/O
   - Mockable interfaces
   - Isolated components
   - Comprehensive golden file testing

2. **Maintainability:**
   - Clear separation of concerns
   - Consistent patterns
   - Reduced coupling
   - Decoupled presentation layer

3. **Extensibility:**
   - Easy to add new commands
   - Plugin architecture ready
   - Multiple output formats (future --json flag)
   - New renderer types without touching business logic

4. **Performance:**
   - Potential for caching
   - Concurrent operations
   - Optimized data access

## Presentation Layer Benefits

The decoupled presentation layer enables:

1. **Future JSON Support:**
   ```go
   // Adding --json flag becomes trivial
   func newListCmd() *cobra.Command {
       cmd := &cobra.Command{
           RunE: func(cmd *cobra.Command, args []string) error {
               result := app.ListDeployments.Run(ctx, params)
               
               // Future: renderer selection
               var renderer Renderer
               if jsonFlag {
                   renderer = render.NewJSONRenderer(cmd.OutOrStdout())
               } else {
                   renderer = render.NewTableRenderer(cmd.OutOrStdout())
               }
               
               return renderer.RenderDeploymentList(result)
           },
       }
   }
   ```

2. **Machine-Readable Output:**
   - Structured JSON for scripting
   - Consistent schema across commands
   - Progress events as JSON streams

3. **Alternative UIs:**
   - Web UI renderer
   - TUI (Terminal UI) renderer
   - IDE plugin renderers

4. **Testing Benefits:**
   - Test business logic without UI concerns
   - Test rendering separately
   - Mock renderers for unit tests
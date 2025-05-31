# Package Audit: Next Steps Proposal

Based on the comprehensive package audit, this document outlines a prioritized action plan to address the most critical issues discovered in the treb-cli codebase.

## Priority 1: Critical Dead Code Removal (1-2 days)

### 1.1 Remove or Integrate Dead Packages

**Issue**: Two entire packages are completely unused despite providing valuable functionality.

**Action Items**:
1. **forge package** - Choose one:
   - **Option A (Recommended)**: Integrate the forge package
     - Replace all `exec.Command("forge", ...)` calls with `forge.NewForge()`
     - Update: `script/executor.go`, `verification/manager.go`, `config/foundry.go`
     - Benefit: Centralized error handling, consistent execution
   - **Option B**: Delete the forge package entirely
     - Remove `cli/pkg/forge/` directory
     - Document why direct exec.Command is preferred

2. **generator package** - Choose one:
   - **Option A**: Complete the refactoring
     - Update `cmd/generate.go` to use `generator.NewGenerator()`
     - Move script generation logic from `contracts` to `generator`
     - Benefit: Better separation of concerns
   - **Option B (Recommended for now)**: Delete the generator package
     - Remove `cli/pkg/generator/` directory
     - Keep generation in `contracts` package until clearer requirements

**Timeline**: 1 day

## Priority 2: Fix Architectural Issues (2-3 days)

### 2.1 Integrate Resolvers Package Everywhere

**Issue**: The resolvers package is designed for universal contract/deployment resolution but only used in one command.

**Action Items**:
1. Update commands to use resolvers:
   ```go
   // In cmd/show.go, cmd/verify.go, cmd/tag.go
   ctx := resolvers.NewContext(".", isInteractive)
   deployment, err := ctx.ResolveDeployment(identifier, manager, chainID, namespace)
   ```

2. Remove duplicate resolution logic from:
   - `cmd/show.go` - Custom deployment resolution
   - `cmd/verify.go` - Direct use of PickDeployment
   - `cmd/tag.go` - Direct use of PickDeployment

**Timeline**: 1 day

### 2.2 Stop Event Package Re-exports

**Issue**: Massive re-exporting between script and events packages creates confusion.

**Action Items**:
1. Update all imports to use events package directly:
   ```go
   // Change from:
   import "github.com/trebuchet-org/treb-cli/cli/pkg/script"
   // To:
   import "github.com/trebuchet-org/treb-cli/cli/pkg/events"
   ```

2. Remove re-exports from:
   - `script/events.go` - Keep only event topic constants
   - `script/proxy_tracker.go` - Remove ProxyRelationship re-export

3. Update affected files:
   - `cmd/run.go`
   - Any other files using script package for event types

**Timeline**: 1 day

## Priority 3: Create Shared Infrastructure (1-2 days)

### 3.1 Create Constants Package

**Issue**: CreateX address is hardcoded in multiple places.

**Action Items**:
1. Create `cli/pkg/constants/constants.go`:
   ```go
   package constants

   const (
       CreateXAddress = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed"
   )
   ```

2. Update imports in:
   - `dev/anvil.go`
   - `registry/script_updater.go`
   - `treb-sol` (Solidity constants)

### 3.2 Create RPC Types Package

**Issue**: RPC types are defined inline in multiple packages.

**Action Items**:
1. Create `cli/pkg/rpc/types.go`:
   ```go
   package rpc

   type Request struct {
       JSONRPC string      `json:"jsonrpc"`
       Method  string      `json:"method"`
       Params  interface{} `json:"params"`
       ID      int         `json:"id"`
   }

   type Response struct {
       JSONRPC string          `json:"jsonrpc"`
       Result  json.RawMessage `json:"result"`
       Error   *Error          `json:"error"`
       ID      int             `json:"id"`
   }

   type Error struct {
       Code    int    `json:"code"`
       Message string `json:"message"`
   }
   ```

2. Update packages to use shared types:
   - `dev/anvil.go`
   - `network/resolver.go`

**Timeline**: 1 day total for both tasks

## Priority 4: Complete Type Migration (2 days)

### 4.1 Remove Legacy Types

**Issue**: Legacy DeploymentEntry type is still used in some places.

**Action Items**:
1. Update to use new types.Deployment:
   - `cmd/dev.go` - Uses DeploymentEntry
   - `script/proxy_tracker.go` - References DeploymentEntry

2. Once all usages are migrated:
   - Delete `types/deployment.go`
   - Update imports to use only `types/registry.go`

**Timeline**: 2 days

## Priority 5: Code Organization (Ongoing)

### 5.1 Remove Dead Code Within Packages

**Issue**: Many unused exports identified in the audit.

**Quick Wins** (can be done incrementally):
1. **abi package**:
   - Make `FormatTokenAmount` unexported

2. **broadcast package**:
   - Delete `BundleTransactionInfo` type
   - Delete `MatchBundleTransactions` function

3. **contracts package**:
   - Delete `AllFilter()` function
   - Delete `ResetGlobalIndexer()` function
   - Delete `LibraryRequirement` type
   - Make template types unexported

4. **config package**:
   - Delete entire `deploy.go` file (legacy)
   - Make `LoadEnvFile`/`LoadEnvFiles` unexported

## Implementation Plan

### Week 1: Critical Issues
- **Day 1**: Dead package removal/integration (Priority 1)
- **Day 2-3**: Resolver integration (Priority 2.1)
- **Day 4**: Stop re-exports (Priority 2.2)
- **Day 5**: Create shared infrastructure (Priority 3)

### Week 2: Type Migration and Cleanup
- **Day 1-2**: Complete type migration (Priority 4)
- **Day 3-5**: Incremental dead code removal (Priority 5)

## Success Metrics

1. **Zero dead packages** - All packages have at least one consumer
2. **Single source of truth** - No duplicate types or re-exports
3. **Consistent patterns** - All commands use resolvers for lookups
4. **Reduced code size** - Measurable reduction in LOC from dead code removal
5. **Better type safety** - No string-based lookups where resolvers should be used

## Notes

- Each priority can be implemented independently
- Tests should be updated alongside code changes
- Consider feature flags for gradual rollout of architectural changes
- Document decisions in code comments for future maintainers

## Alternative Approach: Gradual Migration

If a big-bang approach is too risky, consider:

1. **Phase 1**: Dead code removal only (low risk)
2. **Phase 2**: Add shared infrastructure without changing existing code
3. **Phase 3**: Gradually migrate to new patterns with feature flags
4. **Phase 4**: Remove old code once new patterns are stable

This approach takes longer but reduces risk of breaking changes.
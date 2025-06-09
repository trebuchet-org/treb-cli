# Treb CLI Code Improvement Proposals

Based on the comprehensive code audit, here are proposals for improving the treb CLI codebase architecture, reducing duplication, and enhancing maintainability.

## Executive Summary

The treb CLI codebase is well-structured with clear separation of concerns. However, there are opportunities to:
1. Consolidate duplicate functionality
2. Simplify inter-package dependencies
3. Improve type consistency
4. Enhance error handling patterns
5. Reduce configuration complexity

## Major Issues Identified

### 1. Duplicate Type Definitions

**Problem**: The `types` package has both legacy (V1) and current (V2) deployment structures.
- `DeploymentEntry` (legacy) vs `Deployment` (current)
- Duplicate enums for deployment types and strategies
- Multiple verification status representations

**Proposal**:
- Complete migration to V2 types
- Remove legacy types after migration
- Create migration utilities if needed for backward compatibility

### 2. Configuration Complexity

**Problem**: Multiple configuration sources with overlapping responsibilities:
- `config/deploy.go` - Deployment configuration
- `config/treb.go` - Treb-specific configuration in foundry.toml
- `config/foundry.go` - Foundry configuration
- Similar structures (`SenderConfig`) defined in multiple places

**Proposal**:
- Unify configuration under a single structure
- Create a `ConfigLoader` that handles all sources
- Define `SenderConfig` once and reuse
- Consider using a configuration schema validator

### 3. Circular Dependency Risks

**Problem**: Complex inter-package dependencies that could lead to circular imports:
- `script` depends on `registry` which uses types from `script`
- `abi` package has interfaces that mirror other packages

**Proposal**:
- Create an `interfaces` package for shared interfaces
- Move event types to a dedicated `events/types` package
- Use dependency injection pattern more consistently

### 4. Code Duplication

**Problem**: Similar functionality implemented in multiple places:
- Proxy tracking in both `events` and `script` packages
- Contract resolution logic scattered across packages
- Multiple implementations of forge command execution

**Proposal**:
- Consolidate proxy tracking into `events` package only
- Create a single `forge.Executor` that all packages use
- Centralize contract resolution in `resolvers` package

## Detailed Improvement Proposals

### 1. Package Restructuring

```
cli/pkg/
├── core/           # Core business logic
│   ├── types/      # All type definitions
│   ├── events/     # Event definitions and parsing
│   └── errors/     # Common error types
├── infra/          # Infrastructure concerns
│   ├── forge/      # Forge interaction
│   ├── network/    # Network management
│   ├── storage/    # Registry persistence
│   └── config/     # Configuration loading
├── features/       # Feature-specific logic
│   ├── deploy/     # Deployment orchestration
│   ├── verify/     # Verification logic
│   ├── generate/   # Code generation
│   └── safe/       # Safe integration
└── ui/             # User interaction
    ├── display/    # Output formatting
    ├── prompt/     # User prompts
    └── progress/   # Progress tracking
```

### 2. Interface Segregation

Create focused interfaces in a dedicated package:

```go
// pkg/interfaces/contracts.go
type ContractResolver interface {
    Resolve(identifier string) (*ContractInfo, error)
}

type ContractIndexer interface {
    Index() error
    GetByName(name string) (*ContractInfo, error)
    GetByPath(path string) (*ContractInfo, error)
}

// pkg/interfaces/registry.go
type RegistryReader interface {
    GetDeployment(id string) (*Deployment, error)
    GetByAddress(chainID uint64, address string) (*Deployment, error)
}

type RegistryWriter interface {
    AddDeployment(deployment *Deployment) error
    UpdateVerification(id string, status VerificationStatus) error
}
```

### 3. Unified Configuration System

```go
// pkg/config/unified.go
type Config struct {
    Local    LocalConfig    // From .treb/config.local.json
    Deploy   DeployConfig   // From foundry.toml
    Foundry  FoundryConfig  // From foundry.toml
    Env      EnvConfig      // From .env files
}

type ConfigLoader struct {
    projectRoot string
    cache       *Config
}

func (cl *ConfigLoader) Load() (*Config, error) {
    // Load all config sources
    // Merge with precedence
    // Validate
    // Cache result
}
```

### 4. Error Handling Improvements

Create typed errors for better error handling:

```go
// pkg/core/errors/errors.go
type ErrorCode string

const (
    ErrContractNotFound     ErrorCode = "CONTRACT_NOT_FOUND"
    ErrDeploymentFailed     ErrorCode = "DEPLOYMENT_FAILED"
    ErrInvalidConfiguration ErrorCode = "INVALID_CONFIG"
)

type TrebError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *TrebError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

### 5. Event System Consolidation

Unify event handling across the system:

```go
// pkg/core/events/bus.go
type EventBus struct {
    handlers map[EventType][]EventHandler
}

type EventHandler func(event Event) error

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) error {
    for _, handler := range eb.handlers[event.Type()] {
        if err := handler(event); err != nil {
            return err
        }
    }
    return nil
}
```

### 6. Dependency Injection Container

Implement a simple DI container to manage dependencies:

```go
// pkg/core/container/container.go
type Container struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

func (c *Container) Register(name string, service interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.services[name] = service
}

func (c *Container) Get(name string) (interface{}, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    service, ok := c.services[name]
    if !ok {
        return nil, fmt.Errorf("service %s not found", name)
    }
    return service, nil
}
```

### 7. Testing Improvements

**Current Issues**:
- Limited test coverage
- Tests scattered across packages
- No integration test suite

**Proposals**:
- Add comprehensive unit tests for each package
- Create integration test suite
- Add contract test fixtures
- Implement test helpers package

### 8. Documentation Improvements

**Proposals**:
- Add godoc comments to all public functions
- Create architecture decision records (ADRs)
- Add package-level documentation
- Create developer guide

## Implementation Priority

### Phase 1: Foundation (High Priority)
1. Migrate to V2 types completely
2. Create interfaces package
3. Implement unified configuration
4. Add comprehensive error types

### Phase 2: Consolidation (Medium Priority)
1. Consolidate proxy tracking
2. Unify forge execution
3. Centralize contract resolution
4. Implement event bus

### Phase 3: Enhancement (Lower Priority)
1. Add dependency injection
2. Restructure packages
3. Improve test coverage
4. Enhanced documentation

## Specific Code Smells to Address

1. **Long Methods**: Break down methods over 50 lines
2. **Deep Nesting**: Refactor code with nesting > 3 levels
3. **Magic Strings**: Replace with constants
4. **Duplicate Constants**: Centralize in a constants package
5. **Complex Conditionals**: Extract to well-named functions

## Performance Optimizations

1. **Contract Indexing**: Cache indexing results
2. **Network Calls**: Implement request pooling
3. **File Operations**: Batch registry updates
4. **Parallel Processing**: Use goroutines for independent operations

## Security Enhancements

1. **Input Validation**: Add validation for all user inputs
2. **Path Traversal**: Sanitize file paths
3. **Sensitive Data**: Never log private keys or mnemonics
4. **Network Security**: Validate RPC endpoints

## Conclusion

The treb CLI has a solid foundation but would benefit from:
- Type consolidation and migration
- Simplified configuration management
- Clearer package boundaries
- Enhanced error handling
- Better test coverage

These improvements would make the codebase more maintainable, reduce bugs, and improve developer experience.
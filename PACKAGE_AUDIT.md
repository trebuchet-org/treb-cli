# Package Audit

This document tracks all exported methods in each package and their usage across the codebase.

## Audit Status
- [x] abi
- [x] broadcast  
- [x] config
- [x] contracts
- [x] dev
- [x] forge
- [x] generator
- [x] interactive
- [x] network
- [x] project
- [x] registry
- [x] resolvers
- [x] safe
- [x] script
- [x] types
- [x] events (discovered during audit)
- [x] verification

---

## Package: `abi`

### Exported Items

**Types:**
- `ABIInput` - Represents a constructor/function input parameter
- `ABIConstructor` - Represents the constructor function in an ABI
- `Method` - Represents a function in the ABI  
- `ContractABI` - Represents the parsed ABI of a contract
- `Parser` - Main struct for handling ABI parsing from Foundry artifacts

**Functions:**
- `NewParser(projectRoot string) *Parser` - Creates a new ABI parser
- `ParseContractABI(contractName string) (*ContractABI, error)` - Parses ABI from contract artifact
- `GenerateConstructorArgs(abi *ContractABI) (string, string)` - Generates Solidity constructor argument code
- `FindInitializeMethod(abi *ContractABI) *Method` - Finds initialize method in ABI
- `GenerateInitializerArgs(method *Method) (string, string)` - Generates Solidity initializer argument code
- `DecodeConstructorArgs(contractName string, constructorArgs []byte) (string, error)` - Decodes constructor arguments
- `FormatTokenAmount(amount *big.Int) string` - Formats big.Int as human-readable token amount

### Usage Analysis

1. **Used by `contracts/generator.go`**:
   - `NewParser`, `ParseContractABI`, `GenerateConstructorArgs`, `FindInitializeMethod`, `GenerateInitializerArgs`
   - For generating deployment scripts with proper constructor/initializer handling

2. **Used by `script/display.go`**:
   - `NewParser`, `DecodeConstructorArgs`
   - For displaying decoded constructor arguments in deployment events

3. **Used by `generator/generator.go`**:
   - `NewParser`, `ParseContractABI`, `FindInitializeMethod`
   - For providing user feedback about constructor/initializer detection

### Issues Found

1. **`FormatTokenAmount` should be unexported** - Only used internally within `DecodeConstructorArgs`
2. **Mixed concerns** - Package handles parsing, code generation, and formatting
3. **Hardcoded patterns** - Contract-specific logic (token patterns) embedded in parser

### Recommendations

1. Make `FormatTokenAmount` internal (unexported)
2. Consider splitting into: `abi/types`, `abi/codegen`, `abi/decode`
3. Extract hardcoded token patterns to configuration

---

## Package: `broadcast`

### Exported Items

**Types in parser.go:**
- `BroadcastFile` - Main struct representing a Foundry broadcast file
- `Transaction` - Transaction data structure
- `TxData` - Transaction data details
- `Receipt` - Transaction receipt structure
- `Log` - Event log structure
- `AdditionalContract` - Additional contracts deployed in a transaction
- `Parser` - Parser struct for handling broadcast files

**Functions in parser.go:**
- `NewParser(projectRoot string) *Parser` - Creates a new parser instance
- `ParseBroadcastFile(file string) (*BroadcastFile, error)` - Parses a specific broadcast file
- `ParseLatestBroadcast(scriptName, chainID) (*BroadcastFile, error)` - Parses the latest broadcast file
- `GetAllBroadcastFiles(scriptName, chainID) ([]string, error)` - Gets all broadcast files for a script
- `GetTransactionHashForAddress(address) (hash, blockNum, error)` - Method on BroadcastFile

**Types in tx_matcher.go:**
- `TransactionInfo` - Contains transaction details from broadcast files
- `BundleTransactionInfo` - Groups transaction info with bundle context

**Functions in tx_matcher.go:**
- `MatchBundleTransactions(bundleID, txInfos, sender) *BundleTransactionInfo` - Matches broadcast transactions

### Usage Analysis

1. **Used by `registry/broadcast_enricher.go`**:
   - `BroadcastFile`, `NewParser`, `ParseLatestBroadcast`
   - For enriching registry data with broadcast information

2. **Used by `script/display.go`**:
   - `TransactionInfo`, `NewParser`, `ParseLatestBroadcast`
   - For displaying transaction information

3. **Used by `script/utils.go`**:
   - `BroadcastFile`, `TransactionInfo`
   - For converting broadcast data

### Issues Found

1. **Dead code**:
   - `BundleTransactionInfo` type - never used
   - `MatchBundleTransactions` function - never used

2. **Naming conflict**:
   - `TransactionInfo` type conflicts with same-named type in `script/display.go`

3. **Unexported utility**:
   - `parseHexToUint64` could be useful elsewhere (duplicated in broadcast_enricher.go)

4. **Misplaced code**:
   - Transaction matching logic seems Safe-specific, might belong in `safe` package

### Recommendations

1. Remove dead code (`BundleTransactionInfo`, `MatchBundleTransactions`)
2. Rename `TransactionInfo` in tx_matcher.go to avoid confusion (e.g., `BroadcastTransaction`)
3. Export `parseHexToUint64` as utility function
4. Consider moving Safe-specific matching logic to `safe` package

---

## Package: `config`

### Exported Items

**config.go:**
- `Config` struct - Project/context configuration
- `Load() (*Config, error)` - Loads .treb config
- `Save() error` - Saves config
- `GetCurrentContext() (name, projectRoot)` - Gets current context
- `SetContext(name, path) error` - Sets context
- `RemoveContext(name) error` - Removes context
- `ListContexts() []ContextInfo` - Lists all contexts

**deploy.go (legacy):**
- `DeployConfig` struct - Legacy deployment configuration
- `LoadDeployConfig() (*DeployConfig, error)` - Loads deploy config
- `SenderType` constants (PrivateKey, HardwareWallet, Safe)
- Various legacy sender config types

**env.go:**
- `LoadEnvFile(path) error` - Loads single .env file
- `LoadEnvFiles(paths) error` - Loads multiple .env files

**foundry.go:**
- `FoundryConfig` struct - Parsed foundry.toml
- `FoundryManager` struct - Manages Foundry configuration
- `NewFoundryManager(root) *FoundryManager` - Creates manager
- `GetConfig() *FoundryConfig` - Gets parsed config
- `GetRemappings() []string` - Gets remapping paths
- Various library management functions

**ledger.go:**
- `GetAddressFromPrivateKey(key) (string, error)` - Derives address from private key
- `GetLedgerAddress(derivationPath) (string, error)` - Gets Ledger hardware wallet address

**treb.go:**
- `TrebConfig` struct - New treb configuration format
- `GetSenderFromEnv(types) (interface{}, error)` - Gets sender config from env
- Various sender config types (matching deploy.go)

### Usage Analysis

1. **Used by cmd layer**:
   - `config.go`: Used by `cmd/context.go` and `cmd/run.go` for context management
   - `treb.go`: Used by `cmd/run.go` for sender configuration

2. **Used by pkg layer**:
   - `foundry.go`: Used by `contracts/indexer.go` for getting remappings
   - `treb.go`: Used extensively by `script` package for sender configs

3. **Internal usage only**:
   - `env.go`: LoadEnvFile/LoadEnvFiles only used within config package
   - `ledger.go`: Wallet functions only used within config package

### Issues Found

1. **Major duplication**: deploy.go and treb.go have nearly identical sender configuration code
2. **Dead code**: 
   - All exported functions in deploy.go (replaced by treb.go)
   - Library management functions in foundry.go
   - LoadEnvFile/LoadEnvFiles (should be unexported)
3. **Misplaced code**: 
   - Wallet/crypto utilities in ledger.go belong in separate package
4. **Confusing naming**: Multiple "Config" types with different purposes
5. **Exposed internals**: Several helper functions that should be unexported

### Recommendations

1. **Delete deploy.go entirely** - Legacy code replaced by treb.go
2. **Move wallet utilities** from ledger.go to new `wallet` or `crypto` package
3. **Unexport internal functions**:
   - LoadEnvFile/LoadEnvFiles
   - ParseRemapping/ParseLibraryEntry
4. **Remove unused library functions** from foundry.go
5. **Rename types for clarity**:
   - `Config` → `ContextConfig` 
   - Keep `TrebConfig` and `FoundryConfig` as-is
6. **Consider merging**: foundry.go functionality might belong in `forge` package

---

## Package: `contracts`

### Exported Items

**generator.go:**
- `DeployStrategy` type - CREATE2/CREATE3 enum
- `ProxyType` type - Proxy pattern types
- `ScriptTemplate` struct - Deploy script template data (only used internally)
- `ProxyScriptTemplate` struct - Proxy script template data (only used internally)
- `Generator` struct - Handles deploy script generation
- `NewGenerator(projectRoot) *Generator` - Creates generator
- `GenerateDeployScript(contract, strategy) error` - Generates deploy script
- `GenerateProxyScript(impl, proxy, proxyType) error` - Generates proxy script
- `ValidateStrategy(string) (DeployStrategy, error)` - Validates strategy string
- Strategy constants: `StrategyCreate2`, `StrategyCreate3`
- Proxy constants: `ProxyTypeOZTransparent`, `ProxyTypeOZUUPS`, `ProxyTypeCustom`

**indexer.go:**
- `ContractInfo` struct - Contract discovery information
- `Artifact` struct - Foundry compilation artifact
- `ArtifactMetadata` struct - Artifact metadata (only used internally)
- `BytecodeObject` struct - Bytecode info (only used internally)
- `LinkRef` struct - Library link reference (never used)
- `QueryFilter` struct - Contract query filtering
- `LibraryRequirement` struct - Library dependency (never used)
- `Indexer` struct - Contract discovery and indexing
- `NewIndexer(projectRoot) *Indexer` - Creates indexer
- `GetGlobalIndexer(projectRoot) (*Indexer, error)` - Gets singleton indexer
- `ResetGlobalIndexer()` - Resets global indexer (never used)
- `DefaultFilter()` - Returns default query filter
- `ProjectFilter()` - Returns project-only filter  
- `AllFilter()` - Returns all-inclusive filter (never used)
- Various indexer methods for contract discovery

**generator_paths.go:**
- Path utility methods (mostly used internally)

### Usage Analysis

1. **Used by cmd layer**:
   - `cmd/generate.go`: Uses Generator, strategies, ValidateStrategy, DefaultFilter
   - `cmd/run.go`: Uses indexer for contract lookup

2. **Used by pkg layer**:
   - `interactive/contract_picker.go`: Uses ContractInfo for UI
   - `resolvers/contracts.go`: Uses ContractInfo, QueryFilter, filters
   - `script/display.go`: Uses Indexer type
   - `script/registry_update.go`: Uses Indexer
   - `generator/generator.go`: Uses Generator, ContractInfo, types
   - `registry/script_updater.go`: Uses GetContractByBytecodeHash

### Issues Found

1. **Exported internal types**:
   - `ScriptTemplate`, `ProxyScriptTemplate` - Only used in generator.go
   - `ArtifactMetadata`, `BytecodeObject`, `LinkRef` - Implementation details
   - Proxy type constants - Only used internally

2. **Dead code**:
   - `AllFilter()` - Never used
   - `ResetGlobalIndexer()` - Never used
   - `LibraryRequirement` type - Never used
   - `GetRequiredLibraries()` method - Never used
   - `GetSourceHash()` method - Never used

3. **Misplaced functionality**:
   - Generator could arguably belong in `generator` package
   - Path utilities in generator_paths.go are mostly internal

### Recommendations

1. **Unexport internal types**:
   - Make `ScriptTemplate`, `ProxyScriptTemplate` internal
   - Make `ArtifactMetadata`, `BytecodeObject`, `LinkRef` internal
   - Make proxy constants internal (only used in generator.go)

2. **Remove dead code**:
   - Delete `AllFilter()`, `ResetGlobalIndexer()`
   - Delete `LibraryRequirement` type and related methods
   - Delete `GetSourceHash()` if truly unused

3. **Keep current structure**:
   - Generator is tightly coupled with ContractInfo, makes sense here
   - Indexer is core functionality and belongs here

---

## Package: `dev`

### Exported Items

**anvil.go:**
- `AnvilPidFile` const = "/tmp/treb-anvil-pid" - PID file location
- `AnvilLogFile` const = "/tmp/treb-anvil.log" - Log file location
- `AnvilPort` const = "8545" - Default anvil port
- `CreateXAddress` const = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed" - CreateX factory address
- `RPCRequest` struct - JSON-RPC request structure
- `RPCResponse` struct - JSON-RPC response structure
- `RPCError` struct - JSON-RPC error structure
- `StartAnvil()` - Starts local anvil node with CreateX
- `StopAnvil()` - Stops anvil node
- `RestartAnvil()` - Restarts anvil node
- `ShowAnvilLogs()` - Shows anvil logs
- `ShowAnvilStatus()` - Shows anvil status

### Usage Analysis

1. **Used by cmd layer**:
   - `cmd/dev.go`: Uses all anvil management functions (StartAnvil, StopAnvil, etc.)

2. **Used by tests**:
   - `test/setup_test.go`: Uses RestartAnvil() and StopAnvil() for integration tests

### Issues Found

1. **Duplicated constant**:
   - `CreateXAddress` is hardcoded in both `dev/anvil.go` and `registry/script_updater.go`
   - Should be in a shared constants package

2. **Generic RPC types**:
   - `RPCRequest`, `RPCResponse`, `RPCError` are generic JSON-RPC types
   - `network/resolver.go` defines its own inline RPC response struct
   - Could be moved to shared `rpc` or `jsonrpc` package

3. **Build tag protection**: 
   - Package is only included with `dev` build tag, which is appropriate

### Recommendations

1. **Extract shared constants**:
   - Move `CreateXAddress` to a shared constants package
   - Update both `dev/anvil.go` and `registry/script_updater.go` to use it

2. **Create RPC utilities package**:
   - Move RPC types to shared `rpc` or `jsonrpc` package
   - Reuse in both `dev` and `network` packages

3. **Keep anvil functionality here**:
   - Anvil management is development-specific and belongs in dev package

---

## Package: `forge`

### Exported Items

**Types:**
- `Forge` - Main struct for handling Forge command execution

**Functions:**
- `NewForge(projectRoot string) *Forge` - Creates new Forge executor

**Methods:**
- `Build() error` - Runs forge build with output handling
- `CheckInstallation() error` - Verifies forge is installed
- `RunScript(scriptPath, flags, envVars) (string, error)` - Runs forge script
- `RunScriptWithArgs(scriptPath, flags, envVars, functionArgs) (string, error)` - Runs forge script with function args

### Usage Analysis

**CRITICAL FINDING: This package is completely unused!**

Despite providing a clean abstraction for forge commands, the codebase bypasses it entirely:

1. **Direct exec.Command usage instead:**
   - `verification/manager.go`: `exec.Command("forge", "verify-contract", ...)`
   - `script/executor.go`: `exec.Command("forge", "script", ...)`
   - `config/foundry.go`: `exec.Command("forge", "remappings")`
   - `test/setup_test.go`: `exec.Command("forge", "build")`

2. **No imports found** - grep shows zero imports of the forge package

### Issues Found

1. **Complete dead code** - Entire package is unused
2. **Missed abstraction** - Multiple packages reimplementing forge execution
3. **Inconsistent error handling** - Each package handles forge errors differently
4. **Duplication** - Error parsing logic could be centralized

### Recommendations

Either:
1. **Integrate the forge package**:
   - Replace all direct `exec.Command("forge", ...)` calls
   - Use `forge.NewForge()` for consistent execution
   - Benefit from centralized error handling
   
2. **Delete the forge package**:
   - If there's a reason it's not being used
   - Avoid maintaining dead code

The forge package provides value (error parsing, debug output, consistent interface) that should be leveraged throughout the codebase.

---

## Package: `generator`

### Exported Items

**Types:**
- `Generator` - Wrapper struct for script generation with resolved contracts

**Functions:**
- `NewGenerator(projectRoot string) *Generator` - Creates generator instance
- `GenerateDeployScript(contractInfo, strategy) error` - Generates deploy script
- `GenerateProxyScript(implInfo, proxyInfo, strategy, proxyType) error` - Generates proxy script

### Usage Analysis

**CRITICAL FINDING: This package is completely unused!**

The codebase is using `contracts.Generator` directly instead of this wrapper:
- `cmd/generate.go`: Uses `contracts.NewGenerator(".")` directly

### Issues Found

1. **Another dead package** - Entire generator package is unused
2. **Duplicated functionality** - Wraps `contracts.Generator` without adding significant value
3. **Confusing architecture** - Two generators with similar names but different packages
4. **Incomplete refactoring** - Appears to be part of an unfinished refactoring effort

### Purpose Analysis

Looking at the code, this package appears to:
- Add user-friendly output (constructor info, next steps)
- Provide a cleaner interface for resolved contracts
- Separate concerns between contract discovery and script generation

### Recommendations

1. **Complete the refactoring**:
   - Update `cmd/generate.go` to use this generator
   - Remove script generation from `contracts` package
   - Clear separation: `contracts` for discovery, `generator` for generation

2. **Or delete it**:
   - If the refactoring was abandoned
   - Avoid confusion from duplicate functionality

The wrapper adds value (better UX, cleaner separation) but needs to be integrated or removed.

---

## Package: `interactive`

### Exported Items

**selector.go:**
- `Selector` struct - Core interactive selection functionality
- `NewSelector() *Selector` - Creates selector instance
- `SelectOption(prompt, options) (int, string, error)` - Interactive selection
- `SimpleSelect(prompt, options) (int, string, error)` - Same as SelectOption
- `PromptString(prompt, defaultValue) (string, error)` - Text input
- `PromptConfirm(prompt) (bool, error)` - Yes/no confirmation

**contract_picker.go:**
- `SelectContract(contracts, isInteractive) (*ContractInfo, error)` - Contract selection

**deployment_picker.go:**
- `PickDeployment(deployments, isInteractive) (*Deployment, error)` - Deployment selection

### Usage Analysis

1. **Used by cmd layer:**
   - `cmd/generate.go`: Uses `SelectContract()` for contract selection
   - `cmd/verify.go`: Uses `PickDeployment()` for deployment selection
   - `cmd/tag.go`: Uses `PickDeployment()` for deployment selection

2. **Used by pkg layer:**
   - `resolvers/contracts.go`: Uses `SelectContract()` for disambiguation

3. **Unused exports:**
   - `Selector` struct and all its methods (`NewSelector`, `SelectOption`, etc.)
   - Only the high-level functions are actually used

### Issues Found

1. **Dead code**:
   - `Selector` struct and all its methods are never used
   - `SimpleSelect` duplicates `SelectOption` functionality
   - Low-level selection primitives unused in favor of high-level functions

2. **Inconsistent naming**:
   - `SelectContract` vs `PickDeployment` - should be consistent
   - `SimpleSelect` vs `SelectOption` - confusing duplicates

3. **Missing abstraction**:
   - Contract and deployment pickers have similar patterns
   - Could be generalized to a generic picker

### Recommendations

1. **Remove unused Selector**:
   - Delete `Selector` struct and its methods
   - Keep only `SelectContract` and `PickDeployment` functions
   - Simplify to just use promptui directly

2. **Standardize naming**:
   - Rename to consistent pattern: `SelectContract` and `SelectDeployment`
   - Or use `PickContract` and `PickDeployment`

3. **Consider generic picker**:
   - Both functions format options similarly
   - Could create generic `SelectFromList` with formatter function

The package serves its purpose well for interactive selection, but has accumulated unused code that should be cleaned up.

---

## Package: `network`

### Exported Items

**resolver.go:**
- `NetworkInfo` struct - Contains resolved network details (Name, RpcUrl, ChainID)
- `Resolver` struct - Handles network resolution from foundry.toml
- `NewResolver(projectRoot string) *Resolver` - Creates resolver, loads .env
- `ResolveNetwork(network string) (*NetworkInfo, error)` - Resolves network to RPC URL and chain ID

### Usage Analysis

1. **Used by cmd layer:**
   - `cmd/run.go`: Creates resolver, calls ResolveNetwork for script execution
   - `cmd/verify.go`: Creates resolver for verification manager

2. **Used by pkg layer:**
   - `script/executor.go`: Uses NetworkInfo for RPC URL and broadcast file location
   - `verification/manager.go`: Uses Resolver to get chain ID for explorer URLs
   - `types/deployment.go`: Uses NetworkInfo in DeploymentEntry struct

### Issues Found

1. **Missing network utilities**:
   - Chain ID to network name mapping exists in `verification/manager.go` as private function
   - Should be in network package as exported function

2. **Inline RPC types**:
   - Defines anonymous RPC response struct inline
   - Could use shared RPC types from proposed `rpc` package

3. **Limited functionality**:
   - Only handles network resolution
   - Could include more network utilities (chain lists, explorer URLs, etc.)

### Features

- Environment variable substitution in RPC URLs
- Chain ID caching to avoid repeated RPC calls
- Automatic .env file loading
- Clear error messages for missing configs

### Recommendations

1. **Move network utilities here**:
   - Move `getNetworkName` from verification package and export it
   - Add common chain constants and mappings
   - Add explorer URL generation

2. **Use shared RPC types**:
   - When `rpc` package is created, use its types
   - Remove inline struct definition

3. **Expand functionality**:
   - Add network validation helpers
   - Add testnet/mainnet detection
   - Add gas price estimation utilities

The package provides essential network resolution but could be expanded to be a comprehensive network utilities package.

---

## Package: `project`

### Exported Items

**initializer.go:**
- `Initializer` struct - Handles project setup and initialization
- `NewInitializer() *Initializer` - Creates new initializer
- `Initialize() error` - Sets up treb in existing Foundry project

### Usage Analysis

**Single usage:**
- `cmd/init.go`: Uses NewInitializer() and Initialize() to set up treb

### Functionality

The initializer performs these steps:
1. Validates Foundry project (checks foundry.toml, src/, script/)
2. Checks treb-sol library is installed
3. Creates .treb registry directory structure
4. Creates .env.example file
5. Prints helpful next steps

### Issues Found

1. **Limited scope**:
   - Only handles initialization
   - Could include more project management features

2. **No project validation utilities**:
   - Project validation logic is private
   - Other commands might benefit from these checks

3. **Single-use package**:
   - Entire package exists for one command
   - Could be part of cmd or expanded with more features

### Recommendations

1. **Export validation utilities**:
   - Export `ValidateFoundryProject()` for use by other commands
   - Export `CheckTrebSolLibrary()` for dependency checking

2. **Expand functionality**:
   - Add project configuration management
   - Add project health checks
   - Add upgrade/migration utilities

3. **Or simplify**:
   - If keeping limited scope, consider moving to cmd/init.go
   - Reduces package count for single-use functionality

The package works well for its purpose but could either be expanded to a full project management package or simplified by moving to the command layer.

---

## Package: `registry`

### Exported Items

**manager.go:**
- Constants: `TrebDir`, `DeploymentsFile`, `TransactionsFile`, `SafeTransactionsFile`, `SolidityRegistryFile`
- `Manager` struct - Core registry management
- `NewManager(rootDir) (*Manager, error)` - Creates manager
- Deployment methods: Add, Get (by ID/address/namespace), GetAll, Save, Update
- Transaction methods: Add, Get, GetAll
- Safe transaction methods: Add, Get, GetAll, Update
- Tag methods: AddTag, RemoveTag
- Verification: UpdateDeploymentVerification

**registry_update.go:**
- `RegistryUpdate` struct - Batch registry updates
- `NewRegistryUpdate(namespace, chainID, network, script) *RegistryUpdate`
- Add methods: AddDeployment, AddTransaction, AddSafeTransaction
- `EnrichFromBroadcast(internalTxID, enrichment) error`
- `Apply(manager) error` - Applies updates to registry
- Supporting types: DeploymentUpdate, TransactionUpdate, SafeTransactionUpdate, BroadcastEnrichment

**broadcast_enricher.go:**
- `BroadcastEnricher` struct - Enriches registry with broadcast data
- `NewBroadcastEnricher() *BroadcastEnricher`
- `EnrichFromBroadcastFile(update, broadcastPath) error`
- `EnrichFromBroadcastParser(update, script, chainID) error`

**script_updater.go:**
- `ScriptUpdater` struct - Builds registry updates from script events
- `NewScriptUpdater(indexer) *ScriptUpdater`
- `BuildRegistryUpdate(events, namespace, chainID, network, script) *RegistryUpdate`

### Usage Analysis

1. **Used by cmd layer** (all use `NewManager(".")`):
   - `cmd/list.go`: GetAllDeployments, GetAllDeploymentsHydrated
   - `cmd/show.go`: GetDeployment
   - `cmd/verify.go`: UpdateDeploymentVerification
   - `cmd/sync.go`: GetAllSafeTransactions, UpdateSafeTransaction
   - `cmd/tag.go`: AddTag, RemoveTag
   - `cmd/dev.go`: GetDeploymentByAddress
   - `cmd/init.go`: References registry constants

2. **Used by pkg layer:**
   - `script/registry_update.go`: Uses ScriptUpdater, BroadcastEnricher
   - `resolvers/deployments.go`: Uses Manager for deployment queries
   - `verification/manager.go`: Uses Manager for verification updates
   - `project/initializer.go`: References SolidityRegistryFile

### Issues Found

1. **Hardcoded CreateX address**:
   - `script_updater.go` hardcodes CreateX address
   - Should use shared constant (also in dev package)

2. **Complex update flow**:
   - RegistryUpdate pattern is powerful but complex
   - Could benefit from clearer documentation

3. **Hydration pattern**:
   - GetAllDeploymentsHydrated vs GetAllDeployments confusing
   - Should clarify or simplify

4. **Missing utilities**:
   - No deployment search/filter capabilities
   - No bulk operations support

### Recommendations

1. **Extract constants**:
   - Move CreateX address to shared constants
   - Consider registry file paths as config

2. **Simplify API**:
   - Consider builder pattern for complex queries
   - Add search/filter methods

3. **Improve documentation**:
   - Document the update flow clearly
   - Add examples for common patterns

4. **Add utilities**:
   - Deployment search by contract name
   - Bulk tag operations
   - Registry migration tools

The registry package is well-designed as the central data store but could benefit from API simplification and better constant management.

---

## Package: `resolvers`

### Exported Items

**context.go:**
- `Context` struct - Resolution context with interactive mode support
- `NewContext(projectRoot, interactive) *Context` - Creates context
- `IsInteractive() bool` - Check interactive mode
- `ProjectRoot() string` - Get project root

**contracts.go:**
- `ResolveContract(nameOrPath, filter) (*ContractInfo, error)` - Main contract resolver
- `ResolveContractForImplementation(nameOrPath) (*ContractInfo, error)` - For implementations
- `ResolveContractForProxy(nameOrPath) (*ContractInfo, error)` - For proxies
- `ResolveContractForLibrary(nameOrPath) (*ContractInfo, error)` - For libraries
- `MustResolveContract(nameOrPath, filter) *ContractInfo` - Panics on error
- `ResolveProxyContracts() ([]*ContractInfo, error)` - Get all proxies
- `SelectProxyContract() (*ContractInfo, error)` - Interactive proxy selection

**deployments.go:**
- `ResolveDeployment(identifier, manager, chainID, namespace) (*Deployment, error)` - Main deployment resolver

### Usage Analysis

**CRITICAL FINDING: Severely underutilized package!**

Only one usage found:
- `cmd/generate.go`: Uses `NewContext` and `ResolveContract`

Commands that should use resolvers but don't:
- `cmd/show.go` - Implements own deployment resolution
- `cmd/verify.go` - Uses PickDeployment directly
- `cmd/tag.go` - Uses PickDeployment directly
- Other commands with contract/deployment resolution needs

### Features

1. **Contract Resolution**:
   - Multiple identifier formats (name, path:name)
   - Interactive disambiguation
   - Specialized methods for contract types
   - Integration with contract indexer

2. **Deployment Resolution**:
   - Flexible identifier formats:
     - "Counter" (name)
     - "Counter:v2" (name:label)
     - "staging/Counter" (namespace/name)
     - "11155111/Counter" (chain/name)
     - Full deployment ID
     - Address (requires chainID)
   - Namespace and chain filtering
   - Interactive selection support

### Issues Found

1. **Massive underutilization**:
   - Package designed for widespread use but only used once
   - Commands reimplement resolution logic
   - Duplication of identifier parsing across commands

2. **Missing integration**:
   - Show, verify, tag commands should use this
   - Any command dealing with contracts/deployments needs this

3. **Incomplete API**:
   - No batch resolution methods
   - No caching of resolved items

### Recommendations

1. **Immediate integration needed**:
   - Update `cmd/show.go` to use `ResolveDeployment`
   - Update `cmd/verify.go` to use `ResolveDeployment`
   - Update `cmd/tag.go` to use `ResolveDeployment`
   - Any other deployment/contract resolution

2. **Expand API**:
   - Add batch resolution methods
   - Add resolution caching
   - Add more specialized resolvers

3. **Standardize resolution**:
   - All identifier parsing should go through resolvers
   - Remove duplicate logic from commands
   - Document identifier formats clearly

This package is well-designed but critically underused. It should be the single source of truth for all contract and deployment resolution in the CLI.

---

## Package: `safe`

### Exported Items

**client.go:**
- `Client` struct - Safe Transaction Service API client
- `MultisigTransaction` struct - Safe transaction details from API
- `Confirmation` struct - Transaction confirmation details
- `TransactionServiceURLs` map[uint64]string - Chain ID to API URL mapping
- `NewClient(chainID uint64) *Client` - Creates Safe client
- `SetDebug(debug bool)` - Enable debug output
- `GetTransaction(safeTxHash) (*MultisigTransaction, error)` - Get transaction details
- `GetPendingTransactions(safeAddress) ([]*MultisigTransaction, error)` - Get pending txs
- `IsTransactionExecuted(safeTxHash) (bool, error)` - Check execution status

### Usage Analysis

**Limited usage - only in sync command:**
- `cmd/sync.go`: Uses all Safe client methods for syncing transaction status

**Related but separate:**
- `types/registry.go`: Defines internal `SafeTransaction` type (different from API type)
- `registry/manager.go`: Manages Safe transactions in registry
- `script/display.go`: Displays Safe transaction events

### Features

1. **API Integration**:
   - Clean interface to Safe Transaction Service
   - Support for multiple chains (mainnet, sepolia, base, etc.)
   - Proper error handling and debug output

2. **Transaction Management**:
   - Query transaction details and confirmations
   - Check execution status
   - Get pending transactions for a Safe

### Issues Found

1. **Type duplication**:
   - `safe.MultisigTransaction` vs `types.SafeTransaction`
   - Similar fields but different purposes (API vs storage)
   - Could be confusing

2. **Limited functionality**:
   - Only read operations, no write/submit
   - No transaction building helpers
   - No gas estimation

3. **Hardcoded URLs**:
   - `TransactionServiceURLs` hardcoded in package
   - Could be configuration

### Recommendations

1. **Clarify type names**:
   - Rename to distinguish API types from storage types
   - e.g., `SafeAPITransaction` vs `SafeStoredTransaction`

2. **Expand functionality**:
   - Add transaction creation/submission
   - Add signature collection helpers
   - Add gas estimation

3. **Configuration**:
   - Move service URLs to configuration
   - Allow custom Safe service endpoints

4. **Better integration**:
   - Could be used by script execution for Safe deployments
   - Could provide utilities for Safe deployment scripts

The package serves its purpose well as a Safe Transaction Service client but has room for expansion to support more Safe-related functionality.

---

## Package: `script`

### Exported Items

**executor.go:**
- `Executor` struct - Runs Foundry scripts
- `RunOptions` struct - Script execution options
- `RunResult` struct - Execution results
- `NewExecutor(network, projectRoot) *Executor` - Creates executor
- `Run(scriptPath, options) (*RunResult, error)` - Runs script

**parser.go:**
- `ForgeScriptOutput` struct - Raw forge JSON output
- `ParsedForgeOutput` struct - Complete parsed output
- `ParseForgeOutput(output) (*ParsedForgeOutput, error)` - Parse JSON
- `ParseAllEvents(jsonOutput) ([]ParsedEvent, error)` - Extract events

**events.go:** (re-exports from events package)
- All event types and parsing functions
- `EventType`, `ParsedEvent`, various event structs

**display.go:**
- Display constants (colors, icons)
- `ReportTransactions(txInfos) error` - Display transaction report
- `PrintDeploymentBanner()`, `PrintSuccessMessage()`, etc. - UI utilities
- `GetEventIcon(eventType) string` - Event type icons
- `FormatDeploymentSummary()` - Format deployment info
- `TransactionInfo` struct - Groups related events

**proxy_tracker.go:**
- `ProxyTracker` struct - Tracks proxy relationships
- `NewProxyTracker() *ProxyTracker` - Creates tracker
- `ProxyRelationship` struct (re-exported)

**sender_configs.go:**
- `SenderInitConfig` struct - Sender configuration
- `SenderConfigs` type - Array of configs
- `BuildSenderConfigs(trebConfig) (SenderConfigs, error)` - Build configs
- `EncodeSenderConfigs(configs) (string, error)` - Encode for env

**registry_update.go:**
- `UpdateRegistryFromEvents(events, namespace, chainID, network, script, broadcast) error` - Main registry update function

**utils.go:**
- Utility functions (not many exports here)

### Usage Analysis

**Primary usage in cmd:**
- `cmd/run.go`: Uses almost everything - Executor, display functions, UpdateRegistryFromEvents, ProxyTracker
- `cmd/generate.go`: Only references in comments

**Issues Found**

1. **Massive re-exports**:
   - Most event types are re-exported from events package
   - Creates confusion about source of truth
   - Backward compatibility concern?

2. **Mixed responsibilities**:
   - Package handles execution, parsing, display, and registry updates
   - Could be split into focused packages

3. **Duplicate TransactionInfo**:
   - Defined in display.go
   - Similar type in broadcast package
   - Creates confusion

4. **Direct exec.Command usage**:
   - Executor uses exec.Command directly
   - Should use forge package instead

### Recommendations

1. **Stop re-exporting**:
   - Import events package directly where needed
   - Cleaner separation of concerns

2. **Split package**:
   - `script/executor` - Script execution
   - `script/display` - UI/formatting
   - `script/parser` - Output parsing
   - Keep registry update separate

3. **Use forge package**:
   - Replace direct exec.Command with forge.RunScript

4. **Consolidate types**:
   - Single TransactionInfo type
   - Clear ownership of types

The script package is central to treb's functionality but has grown too large with mixed concerns. It would benefit from splitting into focused sub-packages.

---

## Package: `types`

### Exported Items

**deployment.go (legacy, being phased out):**
- Constants: `Status` (Executed, Queued, Unknown)
- Constants: `DeploymentType` (Singleton, Proxy, Library, Unknown)
- Constants: `DeployStrategy` (Create2, Create3, Unknown)
- `DeploymentEntry` struct - Legacy deployment structure
- `Verification` struct - Legacy verification info
- `DeploymentInfo` struct - Transaction/block info
- `ContractMetadata` struct - Compiler info
- `VerifierStatus` struct - Per-verifier tracking
- Parsing functions: `ParseStatus()`, `ParseDeploymentType()`
- Display methods: `GetDisplayName()`, `GetColoredDisplayName()`

**registry.go (current architecture):**
- Constants: `DeploymentMethod` (CREATE, CREATE2, CREATE3)
- Constants: `TransactionStatus` (PENDING, EXECUTED, FAILED)
- Constants: `VerificationStatus` (UNVERIFIED, PENDING, VERIFIED, FAILED, PARTIAL)
- `Deployment` struct - New comprehensive deployment structure
- `DeploymentStrategy` struct - Deployment method details
- `ProxyInfo` struct - Proxy relationships and history
- `ArtifactInfo` struct - Contract compilation details
- `VerificationInfo` struct - Enhanced verification tracking
- `Transaction` struct - Transaction records
- `SafeTransaction` struct - Safe multisig batches
- `LookupIndexes` struct - Efficient registry lookups
- `SolidityRegistry` struct - Simplified for Solidity
- `RegistryFiles` struct - Complete registry container
- Methods: `GetDisplayName()`, `GetShortID()`

### Usage Analysis

1. **Commands using types:**
   - `cmd/list.go`: Uses Deployment, status constants
   - `cmd/show.go`: Uses Deployment, Transaction
   - `cmd/verify.go`: Uses Deployment, VerificationStatus
   - `cmd/dev.go`: Uses DeploymentEntry (legacy)
   - `cmd/sync.go`: Uses SafeTransaction
   - `cmd/tag.go`: Uses Deployment

2. **Registry package:**
   - Uses all new types extensively
   - Central data model for registry operations

3. **Other packages:**
   - `verification/manager.go`: Uses Deployment, VerifierStatus
   - `script/proxy_tracker.go`: Uses DeploymentEntry (legacy)
   - `interactive/deployment_picker.go`: Uses Deployment
   - `resolvers/deployments.go`: Uses Deployment

### Issues Found

1. **Legacy types still in use**:
   - `DeploymentEntry` used in dev command and proxy tracker
   - Migration incomplete

2. **Confusing naming**:
   - `DeploymentType` vs `DeploymentMethod`
   - `DeployStrategy` vs `DeploymentStrategy`
   - Legacy vs new types not clearly distinguished

3. **Type sprawl**:
   - Many small struct types
   - Could benefit from consolidation

4. **Missing common types**:
   - Address type (using strings)
   - ChainID type (using uint64)
   - Could improve type safety

### Recommendations

1. **Complete migration**:
   - Remove all DeploymentEntry usage
   - Delete deployment.go once migration complete

2. **Rename for clarity**:
   - Prefix legacy types with "Legacy" during transition
   - Make naming consistent between old/new

3. **Add common types**:
   - `type Address string` with validation
   - `type ChainID uint64` with network methods
   - `type TxHash string` with validation

4. **Document transition**:
   - Add migration guide in types package
   - Mark deprecated types clearly

The types package is well-structured for the new registry architecture but needs to complete the migration from legacy types and could benefit from stronger typing for common values.

---

## Package: `events`

### Exported Items

**types.go:**
- Event type constants: `EventTypeContractDeployed`, `EventTypeSafeTransactionQueued`, etc.
- `ParsedEvent` interface - Base interface for all events
- Event structs: `ContractDeployedEvent`, `DeployingContractEvent`, `SafeTransactionQueuedEvent`, `TransactionSimulatedEvent`, `TransactionFailedEvent`, `TransactionBroadcastEvent`, `AdminChangedEvent`, `BeaconUpgradedEvent`, `UpgradedEvent`, `ProxyDeployedEvent`, `SenderDeployerConfiguredEvent`, `UnknownEvent`
- Data structures: `EventDeployment`, `Transaction`, `RichTransaction`, `DeploymentData`, `Log`
- `ProxyRelationship` struct and related types

**proxy_tracker.go:**
- `ProxyTracker` struct - Tracks proxy relationships from events
- `NewProxyTracker() *ProxyTracker` - Creates tracker
- `ProcessEvents(events []ParsedEvent)` - Process events to extract relationships
- `GetRelationshipForProxy(address) (*ProxyRelationship, bool)` - Get proxy info
- `GetProxiesForImplementation(address) []common.Address` - Find proxies
- `PrintProxyRelationships()` - Display relationships

### Usage Analysis

1. **Heavy re-exporting:**
   - `script/events.go`: Re-exports ALL types from events package
   - `script/proxy_tracker.go`: Re-exports ProxyRelationship

2. **Direct usage:**
   - `script/parser.go`: Uses events.Log type
   - `script/registry_update.go`: Imports events for ParsedEvent interface
   - `registry/script_updater.go`: Uses events package directly with own ProxyTracker

3. **Event parsing:**
   - Script package contains parsing functions that convert logs to events
   - Uses event topic constants (keccak256 hashes)

### Issues Found

1. **Confusing architecture**:
   - Events package defines types
   - Script package re-exports everything
   - Script package also contains parsing logic
   - Should parsing be in events package?

2. **Duplicate ProxyTracker**:
   - Events package has ProxyTracker
   - Script package wraps it with additional functionality
   - Registry package uses events.ProxyTracker directly
   - Confusing which to use

3. **Backward compatibility debt**:
   - Massive re-exporting suggests refactoring debt
   - Makes it unclear where types originate

### Recommendations

1. **Move parsing to events package**:
   - Event parsing logic belongs with event definitions
   - Script package should just use events package

2. **Stop re-exporting**:
   - Remove all re-exports from script package
   - Update imports to use events package directly
   - Cleaner architecture

3. **Consolidate ProxyTracker**:
   - Single ProxyTracker implementation
   - Put additional functionality in events package
   - Remove wrapper in script package

4. **Clear separation**:
   - Events: Types and parsing
   - Script: Execution and orchestration
   - Registry: Storage and updates

The events package is well-designed but its relationship with the script package needs clarification. The re-exporting pattern creates confusion and should be eliminated.

---

## Package: `verification`

### Exported Items

**manager.go:**
- `Manager` struct - Handles contract verification
- `NewManager(registryManager, networkResolver) *Manager` - Creates manager
- `VerifyDeployment(deployment) error` - Verify single deployment
- `VerifyDeploymentWithDebug(deployment, debug) error` - Verify with debug output

### Usage Analysis

**Single usage point:**
- `cmd/verify.go`: Uses all verification methods

**Related usage:**
- `cmd/show.go`: Displays verification status
- `cmd/list.go`: Shows verification icons
- `registry/manager.go`: UpdateDeploymentVerification method

### Features

1. **Multi-verifier support**:
   - Etherscan integration
   - Sourcify integration
   - Tracks individual verifier status

2. **Network awareness**:
   - Uses network resolver for chain info
   - Builds appropriate explorer URLs

3. **Registry integration**:
   - Updates verification status in registry
   - Persists verifier-specific results

### Issues Found

1. **Direct forge execution**:
   - Uses exec.Command("forge", "verify-contract")
   - Should use forge package

2. **Hardcoded verifier list**:
   - Verifiers hardcoded in manager
   - Could be configurable

3. **Private network utilities**:
   - `getNetworkName()` is private but useful
   - Should be in network package

4. **Limited error handling**:
   - Some verifier failures silent
   - Could provide better diagnostics

### Recommendations

1. **Use forge package**:
   - Replace exec.Command with forge.RunScript
   - Consistent error handling

2. **Move utilities**:
   - Export getNetworkName to network package
   - Other packages need this functionality

3. **Configurable verifiers**:
   - Allow custom verifier configuration
   - Support private explorers

4. **Better error reporting**:
   - Aggregate verifier errors
   - Provide actionable feedback

The verification package is well-focused but could benefit from using the forge abstraction and sharing network utilities with other packages.

---

## Summary and Key Findings

### Critical Issues

1. **Dead Packages**:
   - `forge` - Completely unused despite providing valuable abstraction
   - `generator` - Unused wrapper around contracts.Generator

2. **Severely Underutilized**:
   - `resolvers` - Only used by generate command, should be used everywhere
   - `project` - Single-use package for init command

3. **Architectural Issues**:
   - Massive re-exporting between script and events packages
   - Direct exec.Command usage bypassing forge package
   - Duplicate types (TransactionInfo in multiple packages)
   - Legacy types (DeploymentEntry) still in use

### Recommendations

1. **Immediate Actions**:
   - Integrate forge package or delete it
   - Use resolvers package in all commands needing deployment/contract resolution
   - Stop re-exporting from events package
   - Complete migration from legacy types

2. **Shared Infrastructure**:
   - Create constants package for CreateX address
   - Create rpc package for JSON-RPC types
   - Move network utilities to network package
   - Create common types (Address, ChainID, TxHash)

3. **Package Consolidation**:
   - Split large script package into focused sub-packages
   - Consider merging single-use packages
   - Clear separation of concerns

4. **Code Quality**:
   - Remove all dead code identified
   - Standardize naming conventions
   - Document package purposes clearly

The codebase shows signs of ongoing refactoring with some packages created but not integrated. Completing these refactoring efforts would significantly improve code organization and maintainability.

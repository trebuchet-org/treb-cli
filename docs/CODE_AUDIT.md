# Treb CLI Code Audit

This document provides a comprehensive analysis of the `cli/pkg` packages in the treb CLI codebase. For each package, we document:
- Exposed structs and interfaces
- Public functions
- Dependencies (both internal and external)
- Where and how the package is used
- Inter-struct dependencies
- Initialization patterns

## Table of Contents

1. [abi Package](#abi-package)
2. [broadcast Package](#broadcast-package)
3. [config Package](#config-package)
4. [contracts Package](#contracts-package)
5. [dev Package](#dev-package)
6. [events Package](#events-package)
7. [forge Package](#forge-package)
8. [generator Package](#generator-package)
9. [interactive Package](#interactive-package)
10. [network Package](#network-package)
11. [project Package](#project-package)
12. [registry Package](#registry-package)
13. [resolvers Package](#resolvers-package)
14. [safe Package](#safe-package)
15. [script Package](#script-package)
16. [types Package](#types-package)
17. [verification Package](#verification-package)

---

## abi Package

**Purpose**: Provides ABI parsing, resolution, and transaction decoding functionality.

### Interfaces

#### ABIResolver
```go
type ABIResolver interface {
    ResolveABI(address common.Address) (contractName string, abiJSON string, isProxy bool, implAddress *common.Address)
}
```
- **Used by**: `TransactionDecoder` for on-demand ABI loading
- **Implemented by**: `RegistryABIResolver`

#### DeploymentLookup
```go
type DeploymentLookup interface {
    GetDeploymentByAddress(chainID uint64, address string) (*types.Deployment, error)
}
```
- **Used by**: `RegistryABIResolver`
- **Implemented by**: Registry manager (in registry package)

#### ContractLookup
```go
type ContractLookup interface {
    GetContractByArtifact(artifact string) ContractInfo
}
```
- **Used by**: `RegistryABIResolver`
- **Implemented by**: Contracts indexer (in contracts package)

#### ContractInfo
```go
type ContractInfo interface {
    GetArtifactPath() string
}
```
- **Used by**: `RegistryABIResolver`
- **Implemented by**: Contract types in contracts package

### Structs

#### RegistryABIResolver
- **Purpose**: Resolves ABIs using deployment registry and contracts indexer
- **Initialization**: `NewRegistryABIResolver(projectRoot string, chainID uint64, deploymentLookup DeploymentLookup, contractLookup ContractLookup)`
- **Dependencies**: 
  - DeploymentLookup interface
  - ContractLookup interface
  - Parser (internal)
- **Used by**: Script executor, transaction decoder

#### TransactionDecoder
- **Purpose**: Decodes transaction calldata and return data
- **Initialization**: `NewTransactionDecoder()`
- **Dependencies**: 
  - ABIResolver interface (optional, set via `SetABIResolver`)
  - ethereum ABI package
- **Used by**: Script enhanced display, broadcast parser

#### Parser
- **Purpose**: Parses contract ABIs from Foundry artifacts
- **Initialization**: `NewParser()`
- **Dependencies**: None (uses standard library and ethereum packages)
- **Used by**: RegistryABIResolver, script parameter resolver

#### ContractABI
- **Purpose**: Represents parsed ABI with constructor and methods
- **Fields**: Constructor, Initializer, Methods, ABI
- **Used by**: Parser (returned from ParseContractABI)

#### DecodedTransaction
- **Purpose**: Human-readable representation of a transaction
- **Fields**: To, From, Method, Args, IsError, ErrorName, ReturnValue
- **Used by**: TransactionDecoder (returned from DecodeTransaction)

### External Dependencies
- `github.com/ethereum/go-ethereum/accounts/abi`
- `github.com/ethereum/go-ethereum/accounts/abi/bind/v2`
- `github.com/ethereum/go-ethereum/common`
- `github.com/ethereum/go-ethereum/common/hexutil`
- `github.com/ethereum/go-ethereum/core/types`

### Internal Dependencies
- `github.com/trebuchet-org/treb-cli/cli/pkg/types`

### Key Usage Patterns

1. **ABI Resolution Flow**:
   ```
   Script Executor → RegistryABIResolver → DeploymentLookup/ContractLookup → Parser → ABI
   ```

2. **Transaction Decoding Flow**:
   ```
   Broadcast Parser → TransactionDecoder → ABIResolver (if set) → DecodedTransaction
   ```

3. **Constructor Handling**:
   ```
   Script Parameter Resolver → Parser → GenerateConstructorArgs/DecodeConstructorArgs
   ```

### Well-Known Addresses
- CreateX: `0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed`
- MultiSend: `0x40A2aCCbd92BCA938b02010E17A5b8929b49130D`
- ProxyFactory: `0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67`

---

## broadcast Package

**Purpose**: Handles parsing and processing of Foundry broadcast files containing deployment information.

### Structs

#### BroadcastFile
- **Purpose**: Main structure representing a Foundry broadcast file
- **Fields**:
  - `Transactions []Transaction` - List of executed transactions
  - `Receipts []Receipt` - Transaction receipts with execution results
  - `Libraries []string` - Deployed libraries
  - `Pending []string` - Pending operations
  - `Returns interface{}` - Script return values
  - `Timestamp int64` - Execution timestamp
  - `Chain int64` - Chain ID
  - `Multi bool` - Multi-transaction flag
  - `Commit string` - Git commit hash
- **Methods**:
  - `GetTransactionHashForAddress(address common.Address) (common.Hash, uint64, error)` - Finds deployment transaction

#### Transaction
- **Purpose**: Individual transaction details from broadcast
- **Fields**: ContractName, ContractAddress, Function, Arguments, Transaction (TxData), AdditionalContracts
- **Used by**: BroadcastFile

#### TxData
- **Purpose**: Low-level transaction data
- **Fields**: Type, From, Gas, Value, Data, Nonce, AccessList, MaxFeePerGas, MaxPriorityFeePerGas
- **Used by**: Transaction

#### Receipt
- **Purpose**: Transaction receipt information
- **Fields**: TransactionHash, TransactionIndex, BlockHash, BlockNumber, Status, CumulativeGasUsed, Logs, LogsBloom, EffectiveGasPrice
- **Used by**: BroadcastFile

#### AdditionalContract
- **Purpose**: Tracks contracts deployed via CREATE2/CREATE3 (CreateX)
- **Fields**: Address, TransactionHash, InitCode
- **Used by**: Transaction

#### Parser
- **Purpose**: Parses Foundry broadcast files
- **Initialization**: `NewParser(projectRoot string)`
- **Dependencies**: None (uses file system and JSON parsing)
- **Used by**: Registry broadcast enricher, script executor

#### TransactionInfo (tx_matcher.go)
- **Purpose**: Simplified transaction details for matching
- **Fields**: Hash, Sender, Target, Calldata
- **Used by**: Bundle matching logic

#### BundleTransactionInfo (tx_matcher.go)
- **Purpose**: Groups transactions by bundle
- **Fields**: BundleID, Transactions, IsSafe
- **Used by**: Registry update logic for Safe deployments

### Public Functions

1. **parser.go**
   - `NewParser(projectRoot string) *Parser` - Creates parser instance
   - `ParseBroadcastFile(file string) (*BroadcastFile, error)` - Parses specific file
   - `ParseLatestBroadcast(scriptName string, chainID uint64) (*BroadcastFile, error)` - Parses latest broadcast
   - `GetAllBroadcastFiles(scriptName string, chainID uint64) ([]string, error)` - Lists all broadcasts

2. **tx_matcher.go**
   - `MatchBundleTransactions(bundleID common.Hash, txInfos []TransactionInfo, senderAddr common.Address) *BundleTransactionInfo` - Matches transactions to bundles

### External Dependencies
- `github.com/ethereum/go-ethereum/common` - Ethereum types

### Internal Dependencies
None

### Key Usage Patterns

1. **Broadcast Parsing Flow**:
   ```
   Script Executor → Parser → ParseLatestBroadcast → BroadcastFile
   Registry Enricher → Parser → GetTransactionHashForAddress → Deployment Info
   ```

2. **Bundle Matching Flow**:
   ```
   Registry Update → MatchBundleTransactions → BundleTransactionInfo → Safe Detection
   ```

3. **CreateX Deployment Tracking**:
   ```
   Transaction → AdditionalContracts → Address Mapping
   ```

### File Structure Convention
- Broadcast files: `broadcast/<script>/<chainId>/run-*.json`
- Latest symlink: `broadcast/<script>/<chainId>/run-latest.json`

---

## config Package

**Purpose**: Provides comprehensive configuration management for treb CLI, handling local config, deployment settings, environment variables, and Foundry integration.

### Structs

#### Config (config.go)
- **Purpose**: Main treb configuration
- **Fields**: Namespace, Network
- **Used by**: CLI commands for default values

#### Manager (config.go)
- **Purpose**: Manages `.treb/config.local.json` file
- **Initialization**: `NewManager(projectRoot string)`
- **Dependencies**: None
- **Used by**: CLI commands for persistent settings

#### DeployConfig (deploy.go)
- **Purpose**: Deployment configuration from foundry.toml
- **Fields**: Profile (map of ProfileConfig)
- **Methods**:
  - `ResolveSenderName(address string) string`
  - `GetProfileConfig(profile string) (*ProfileConfig, error)`
  - `GetSender(profile, senderName string) (*SenderConfig, error)`
  - `Validate(namespace string) error`
  - `GenerateEnvVars(namespace string) (map[string]string, error)`

#### ProfileConfig (deploy.go)
- **Purpose**: Configuration for a deployment profile
- **Fields**: Senders (map of SenderConfig), LibraryDeployer
- **Used by**: DeployConfig

#### SenderConfig (deploy.go, treb.go)
- **Purpose**: Configuration for deployment sender
- **Fields**: Type, Address, PrivateKey, Safe, Signer, DerivationPath
- **Used by**: ProfileConfig, TrebConfig, script package

#### FoundryConfig (foundry.go)
- **Purpose**: Full foundry.toml configuration
- **Fields**: Profile, RpcEndpoints
- **Used by**: FoundryManager

#### ProfileFoundryConfig (foundry.go)
- **Purpose**: Foundry build and test settings per profile
- **Fields**: Sender, Libraries, SrcPath, OutPath, LibPaths, TestPath, ScriptPath, Remappings, SolcVersion, Optimizer, OptimizerRuns
- **Used by**: FoundryConfig

#### FoundryManager (foundry.go)
- **Purpose**: Manages foundry.toml file operations
- **Initialization**: `NewFoundryManager(projectRoot string)`
- **Dependencies**: TOML parser
- **Used by**: Registry manager for library updates

#### TrebConfig (treb.go)
- **Purpose**: Treb-specific configuration in foundry.toml
- **Fields**: Senders, LibraryDeployer
- **Methods**: `GetSenderNameByAddress(address string) (string, error)`
- **Used by**: Script executor for sender resolution

#### FoundryProfileConfig (treb.go)
- **Purpose**: Complete profile config including treb section
- **Fields**: Treb (TrebConfig), embedded toml.Primitive
- **Used by**: FoundryFullConfig

#### FoundryFullConfig (treb.go)
- **Purpose**: Complete foundry.toml structure with treb sections
- **Fields**: Profile (map of FoundryProfileConfig)
- **Methods**: `GetProfileTrebConfig(profileName string) (*TrebConfig, error)`
- **Used by**: Script package for loading sender configs

### Public Functions

1. **config.go**
   - `DefaultConfig() *Config` - Returns default configuration
   - Manager methods for Load, Save, Set, Get, List, Exists, GetPath

2. **deploy.go**
   - `LoadDeployConfig(projectPath string) (*DeployConfig, error)` - Loads deploy config

3. **env.go**
   - `LoadEnvFile(filePath string) error` - Loads single .env file
   - `LoadEnvFiles(filePaths ...string) error` - Loads multiple .env files

4. **foundry.go**
   - `LoadFoundryConfig(projectRoot string) (*FoundryConfig, error)` - Helper to load config
   - FoundryManager methods for AddLibrary, UpdateLibraryAddress, GetRemappings, etc.
   - `ParseRemapping(remapping string) (string, string, error)` - Parses import remapping
   - `ParseLibraryEntry(entry string) (path, name, address string, err error)` - Parses library

5. **ledger.go**
   - `GetLedgerAddress(derivationPath string) (string, error)` - Gets Ledger address
   - `GetAddressFromPrivateKey(privateKeyHex string) (string, error)` - Derives address

6. **treb.go**
   - `LoadTrebConfig(projectPath string) (*FoundryFullConfig, error)` - Loads treb config

### External Dependencies
- `github.com/BurntSushi/toml` - TOML parsing
- `github.com/ethereum/go-ethereum/crypto` - Ethereum crypto functions

### Internal Dependencies
None

### Key Usage Patterns

1. **Configuration Loading Flow**:
   ```
   CLI Command → Manager → Load config.local.json
   Script Executor → LoadTrebConfig → Parse foundry.toml → Sender configs
   ```

2. **Environment Variable Expansion**:
   ```
   Config value "${VAR}" → os.Getenv("VAR") → Expanded value
   ```

3. **Library Management Flow**:
   ```
   Registry Update → FoundryManager → AddLibrary → Update foundry.toml
   ```

4. **Sender Resolution**:
   ```
   Address → TrebConfig → GetSenderNameByAddress → Sender name
   ```

### Configuration Hierarchy
1. `.treb/config.local.json` - Local treb settings (namespace, network)
2. `foundry.toml` - Build settings and deployment configurations
3. `.env` files - Environment variables
4. Command-line flags - Override all above

### Sender Types
- `private_key` - Direct private key deployment
- `safe` - Gnosis Safe multisig deployment
- `ledger` - Hardware wallet deployment

---

## contracts Package

**Purpose**: Discovers, indexes, and generates deployment scripts for contracts.

### Enums/Types

- `DeployStrategy` - CREATE2, CREATE3
- `ProxyType` - TransparentUpgradeable, UUPSUpgradeable, Custom

### Structs

#### ContractInfo
- **Purpose**: Information about a discovered contract
- **Fields**: Name, Path, ArtifactPath, IsLibrary, IsInterface, IsAbstract, LibraryRequirements
- **Used by**: Generator, interactive package, resolvers

#### Artifact
- **Purpose**: Foundry compilation artifact
- **Fields**: ABI, Bytecode, DeployedBytecode, Metadata, MethodIdentifiers
- **Used by**: ContractInfo, ABI parser

#### Indexer
- **Purpose**: Discovers and indexes contracts
- **Initialization**: `NewIndexer(projectRoot string)`
- **Dependencies**: forge package, abi package
- **Used by**: Registry ABIResolver, resolvers, global indexer

#### Generator
- **Purpose**: Generates deployment scripts
- **Initialization**: `NewGenerator(projectRoot string)`
- **Dependencies**: abi parser, config manager
- **Used by**: generate command, proxy deployment

#### QueryFilter
- **Purpose**: Filtering options for contract queries
- **Fields**: ExcludeLibraries, ExcludeInterfaces, ExcludeAbstract, IncludeNodeModules, MatchPattern
- **Used by**: Indexer queries

### Public Functions

1. **indexer.go**
   - `GetGlobalIndexer(projectRoot string) (*Indexer, error)` - Global singleton
   - `ResetGlobalIndexer()` - Resets global instance
   - `DefaultFilter()` - Standard filter
   - `ProjectFilter()` - Excludes node_modules
   - `ScriptFilter()` - Only scripts
   - `AllFilter()` - No filtering
   - Indexer methods: Discover, GetContract, GetContracts, GetScripts, etc.

2. **generator.go & generator_paths.go**
   - `GenerateDeployScript(contractInfo *ContractInfo, strategy DeployStrategy) error`
   - `GenerateProxyDeployScript(...) error`
   - `GetDeployScriptPath(contractInfo *ContractInfo) string`
   - `GetProxyScriptPath(contractInfo *ContractInfo) string`
   - `ValidateStrategy(strategy string) (DeployStrategy, error)`

### External Dependencies
- `github.com/fatih/color` - Colored output

### Internal Dependencies
- `abi` - For parsing artifacts
- `config` - For foundry config
- `forge` - For build operations

---

## dev Package

**Purpose**: Manages local development environment (Anvil node).

### Constants
- `AnvilPidFile` = ".anvil.pid"
- `AnvilLogFile` = "anvil.log"
- `AnvilPort` = 8545
- `CreateXAddress` = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed"

### Structs

#### RPCRequest/Response/Error
- **Purpose**: JSON-RPC communication structures
- **Used by**: Anvil interaction functions

### Public Functions
- `StartAnvil() error` - Starts Anvil node
- `StopAnvil() error` - Stops Anvil node
- `RestartAnvil() error` - Restarts Anvil
- `ShowAnvilLogs() error` - Shows logs
- `ShowAnvilStatus() error` - Shows status

### External Dependencies
- `github.com/fatih/color` - Colored output

### Internal Dependencies
None

---

## events Package

**Purpose**: Parses and tracks proxy relationships from events.

### Interfaces

#### ParsedEvent
- **Methods**: EventType() EventType
- **Implemented by**: All event types

### Enums

- `EventType` - AdminChanged, BeaconUpgraded, Upgraded, Unknown
- `ProxyRelationshipType` - MINIMAL, UUPS, TRANSPARENT, BEACON

### Structs

#### ProxyTracker
- **Purpose**: Tracks proxy relationships
- **Initialization**: `NewProxyTracker()`
- **Dependencies**: None
- **Used by**: Script executor, enhanced display

#### ProxyRelationship
- **Purpose**: Stores proxy-implementation relationship
- **Fields**: ProxyAddress, ImplementationAddress, Type, InitialDeployment
- **Used by**: ProxyTracker

#### Event Types
- `AdminChangedEvent` - Admin change events
- `BeaconUpgradedEvent` - Beacon upgrade events
- `UpgradedEvent` - Implementation upgrade events
- `ProxyDeployedEvent` - Initial deployment events

### Public Functions
- ProxyTracker methods: ProcessEvents, GetRelationshipForProxy, GetProxiesForImplementation, etc.

### External Dependencies
- `github.com/ethereum/go-ethereum/common`
- `github.com/fatih/color`

### Internal Dependencies
None

---

## forge Package

**Purpose**: Wrapper for Forge command execution.

### Structs

#### Forge
- **Purpose**: Handles Forge commands
- **Initialization**: `NewForge(projectRoot string)`
- **Dependencies**: None
- **Used by**: Contracts indexer, script executor

### Public Functions
- `Build() error` - Runs forge build
- `CheckInstallation() error` - Checks forge installation
- `RunScript(scriptPath string, flags []string, envVars map[string]string) (string, error)`
- `RunScriptWithArgs(...) (string, error)` - With function arguments

### External Dependencies
None (uses os/exec)

### Internal Dependencies
None

---

## generator Package

**Purpose**: High-level orchestration of script generation.

### Structs

#### Generator
- **Purpose**: Handles script generation with resolved contracts
- **Initialization**: `NewGenerator(projectRoot string)`
- **Dependencies**: contracts generator, abi parser
- **Used by**: generate command

### Public Functions
- `GenerateDeployScript(contractInfo *contracts.ContractInfo, strategy contracts.DeployStrategy) error`
- `GenerateProxyScript(...) error`

### External Dependencies
None

### Internal Dependencies
- `abi` - For parsing artifacts
- `contracts` - For contract info and generation

---

## interactive Package

**Purpose**: User interaction utilities for CLI.

### Structs

#### Selector
- **Purpose**: Handles interactive selection
- **Initialization**: `NewSelector()`
- **Dependencies**: promptui
- **Used by**: Resolvers, parameter prompter

#### FuzzySearcher
- **Purpose**: Advanced fuzzy search
- **Initialization**: `NewFuzzySearcher(items []string)`
- **Dependencies**: fuzzy library
- **Used by**: Contract/deployment pickers

### Public Functions

1. **selector.go**
   - `SelectOption(prompt string, options []string, defaultIndex int) (string, int, error)`
   - `SimpleSelect(...) (string, int, error)` - Without search
   - `PromptString(prompt string, defaultValue string) (string, error)`
   - `PromptConfirm(prompt string, defaultValue bool) (bool, error)`

2. **contract_picker.go**
   - `SelectContract(matches []*contracts.ContractInfo, prompt string) (*contracts.ContractInfo, error)`

3. **deployment_picker.go**
   - `PickDeployment(matches []*types.Deployment, prompt string) (*types.Deployment, error)`

4. **fuzzy_search.go**
   - `FuzzySearchFunc(items []string) func(input string, index int) bool`

### External Dependencies
- `github.com/fatih/color`
- `github.com/manifoldco/promptui`
- `github.com/sahilm/fuzzy`

### Internal Dependencies
- `contracts` - For ContractInfo
- `types` - For Deployment

---

## network Package

**Purpose**: Network resolution and chain ID extraction.

### Structs

#### NetworkInfo
- **Purpose**: Resolved network details
- **Fields**: Name, RpcUrl, ChainID
- **Used by**: Commands needing network info

#### Resolver
- **Purpose**: Resolves networks from config
- **Initialization**: `NewResolver(projectRoot string)`
- **Dependencies**: None
- **Used by**: Script executor, verification manager

### Public Functions
- `ResolveNetwork(network string) (*NetworkInfo, error)` - Resolves network to RPC and chain ID

### External Dependencies
- `github.com/joho/godotenv` - .env file loading

### Internal Dependencies
None

---

## project Package

**Purpose**: Project initialization and setup.

### Structs

#### Initializer
- **Purpose**: Sets up treb in projects
- **Initialization**: `NewInitializer()`
- **Dependencies**: None
- **Used by**: init command

### Public Functions
- `Initialize() error` - Sets up treb structure

### External Dependencies
None

### Internal Dependencies
None

---

## registry Package

**Purpose**: Manages deployment tracking and registry operations with thread-safe JSON storage.

### Structs

#### Manager
- **Purpose**: Main registry manager with thread-safe operations
- **Initialization**: `NewManager(rootDir string) (*Manager, error)`
- **Dependencies**: None (uses mutex for thread safety)
- **Used by**: All commands that read/write deployments

#### BroadcastEnricher
- **Purpose**: Enriches registry updates with broadcast data
- **Fields**: projectRoot, broadcastParser
- **Used by**: Registry update process

#### RegistryUpdate
- **Purpose**: Represents all changes to be applied atomically
- **Fields**: Deployments, Transactions, SafeTransactions, Metadata
- **Methods**: Apply, GetSummary, various setters
- **Used by**: Script executor to update registry

#### ScriptUpdater
- **Purpose**: Builds registry updates from script events
- **Fields**: contractIndexer, proxyTracker
- **Used by**: Script executor

### Public Functions

1. **manager.go**
   - Manager methods: AddDeployment, AddTransaction, AddSafeTransaction
   - Query methods: GetDeployment, GetDeploymentByAddress, ListDeployments
   - Update methods: UpdateDeploymentVerification, AddTag
   - Export methods: ExportToSolidity

2. **broadcast_enricher.go**
   - `EnrichFromBroadcastFile(update *RegistryUpdate, broadcastPath string) error`
   - `EnrichFromBroadcastParser(update *RegistryUpdate, scriptName string, chainID uint64) error`

3. **registry_update.go**
   - `NewRegistryUpdate(namespace string, chainID uint64, networkName string, scriptPath string) *RegistryUpdate`

4. **script_updater.go**
   - `BuildRegistryUpdate(scriptEvents []interface{}, ...) *RegistryUpdate`

### External Dependencies
- `github.com/ethereum/go-ethereum/common`

### Internal Dependencies
- `types` - Core data structures
- `broadcast` - For broadcast file parsing
- `abi/treb` - For event types
- `contracts` - For contract info
- `events` - For proxy tracking

### Key Features
- Thread-safe operations with mutex
- Atomic updates via RegistryUpdate
- Multiple indexes for efficient lookups
- Broadcast file enrichment
- Safe transaction tracking

---

## resolvers Package

**Purpose**: Resolution logic for contracts and deployments with interactive support.

### Structs

#### Context
- **Purpose**: Resolver configuration
- **Fields**: projectRoot, interactive
- **Initialization**: `NewContext(projectRoot string, interactive bool)`
- **Used by**: All resolver functions

### Public Functions

1. **context.go**
   - `IsInteractive() bool`
   - `ProjectRoot() string`

2. **contracts.go**
   - `ResolveContract(nameOrPath string, filter contracts.QueryFilter) (*contracts.ContractInfo, error)`
   - `ResolveContractForImplementation(nameOrPath string) (*contracts.ContractInfo, error)`
   - `ResolveContractForProxy(nameOrPath string) (*contracts.ContractInfo, error)`
   - `ResolveContractForLibrary(nameOrPath string) (*contracts.ContractInfo, error)`
   - `ResolveProxyContracts() ([]*contracts.ContractInfo, error)`
   - `SelectProxyContract() (*contracts.ContractInfo, error)`

3. **deployments.go**
   - `ResolveDeployment(identifier string, manager *registry.Manager, chainID uint64, namespace string) (*types.Deployment, error)`

### External Dependencies
- `github.com/manifoldco/promptui`
- `github.com/fatih/color`

### Internal Dependencies
- `contracts` - For contract discovery
- `interactive` - For user interaction
- `registry` - For deployment lookups
- `types` - For data structures

### Resolution Patterns
- By exact name match
- By path match
- By deployment ID/address
- Interactive selection when multiple matches

---

## safe Package

**Purpose**: Interacts with Safe Transaction Service API.

### Structs

#### MultisigTransaction
- **Purpose**: Safe multisig transaction
- **Fields**: SafeTxHash, Safe, To, Value, Data, Nonce, ExecutionDate, etc.
- **Used by**: Registry for tracking Safe transactions

#### Client
- **Purpose**: Safe Transaction Service client
- **Initialization**: `NewClient(chainID uint64) (*Client, error)`
- **Dependencies**: HTTP client
- **Used by**: Script executor for Safe status checks

### Public Functions
- `SetDebug(debug bool)` - Enable debug logging
- `GetTransaction(safeTxHash common.Hash) (*MultisigTransaction, error)`
- `GetPendingTransactions(safeAddress common.Address) ([]MultisigTransaction, error)`
- `IsTransactionExecuted(safeTxHash common.Hash) (bool, *common.Hash, error)`

### Pre-configured Networks
- Mainnet, Sepolia, Arbitrum, Optimism, Polygon, BSC, Gnosis, Avalanche, Base, Celo

### External Dependencies
- `github.com/ethereum/go-ethereum/common`

### Internal Dependencies
None

---

## script Package

**Purpose**: Handles Foundry script execution, event parsing, and deployment display.

### Major Components

#### Executor (executor.go)
- **Purpose**: Runs Foundry scripts
- **Structs**: Executor, RunOptions, RunResult
- **Key Functions**: Run, buildEnvironment, parseOutput
- **Dependencies**: config, network, abi/treb

#### ParameterResolver (parameter_resolver.go)
- **Purpose**: Resolves meta types to values
- **Key Functions**: ResolveValue, ResolveAll
- **Meta Types**: sender:ID, deployment:ID, artifact:path
- **Dependencies**: config, registry, contracts, resolvers

#### Display Components (display.go, enhanced_display.go)
- **Purpose**: Shows deployment progress
- **Structs**: TransactionInfo, EnhancedEventDisplay
- **Features**: Phase tracking, ABI decoding, proxy detection
- **Dependencies**: abi, registry, events

#### SenderConfigs (sender_configs.go)
- **Purpose**: Builds sender configurations
- **Functions**: BuildSenderConfigs, EncodeSenderConfigs
- **Dependencies**: config, ethereum crypto

#### Parser (parser.go)
- **Purpose**: Parses forge output
- **Structs**: ForgeScriptOutput, RawLog, ParsedForgeOutput
- **Functions**: ParseForgeOutput, ParseAllEvents

#### EventParser (event_parser.go)
- **Purpose**: Uses ABI bindings for events
- **Functions**: ParseEvent
- **Dependencies**: abi/treb, events

#### Parameters (parameters.go, parameter_prompt.go)
- **Purpose**: Script parameter handling
- **Structs**: Parameter, ParameterParser, ParameterPrompter
- **Types**: string, address, uint256, bool, bytes32, meta types
- **Dependencies**: contracts, interactive

#### Proxy Tracking (proxy_events.go, proxy_tracker.go)
- **Purpose**: Tracks proxy relationships
- **Uses**: events.ProxyTracker
- **Dependencies**: abi/treb, events

### External Dependencies
- `github.com/ethereum/go-ethereum`
- `github.com/fatih/color`
- `github.com/manifoldco/promptui`

### Internal Dependencies
- Most internal packages used

---

## types Package

**Purpose**: Core data structures for the entire system.

### Key Types

#### deployment.go (Legacy V1)
- `Status` - EXECUTED, PENDING_SAFE, UNKNOWN
- `DeploymentType` - SINGLETON, PROXY, LIBRARY
- `DeployStrategy` - CREATE2, CREATE3
- `DeploymentEntry` - Legacy deployment structure
- `Verification` - Verification status

#### registry.go (Current V2)
- `Deployment` - Enhanced deployment record
- `DeploymentStrategy` - Method details
- `ProxyInfo` - Proxy-specific data
- `ArtifactInfo` - Contract artifact info
- `Transaction` - Blockchain transaction
- `SafeTransaction` - Safe multisig batch
- `Registry` - Complete registry structure
- `SolidityRegistry` - Simplified for Solidity

### Public Functions
- Various parsing and display functions
- ID generation utilities
- Enum conversions

### External Dependencies
- `github.com/ethereum/go-ethereum/common`
- `github.com/fatih/color`

### Internal Dependencies
- `network` - For network names in legacy types

---

## verification Package

**Purpose**: Contract verification on Etherscan and Sourcify.

### Structs

#### Manager
- **Purpose**: Handles verification operations
- **Initialization**: `NewManager(registryManager *registry.Manager, networkResolver *network.Resolver)`
- **Dependencies**: registry, network
- **Used by**: verify command

### Public Functions
- `VerifyDeployment(deployment *types.Deployment) error`
- `VerifyDeploymentWithDebug(deployment *types.Deployment, debug bool) error`

### Verification Process
1. Checks if already verified
2. Runs forge verify-contract
3. Updates registry with status
4. Builds explorer URLs

### External Dependencies
None (uses forge command)

### Internal Dependencies
- `registry` - For updating verification status
- `network` - For network resolution
- `types` - For deployment structures

---

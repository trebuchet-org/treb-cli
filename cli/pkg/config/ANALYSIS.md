# Config Package Analysis

## Exported Items and Usage

### config.go - Context/Project Configuration

**Exported Types:**
- `Config` - Project-level configuration (namespace, network, sender)
- `Manager` - Manages .treb/config.local.json file

**Exported Functions:**
- `DefaultConfig()` - Returns default config
- `NewManager(projectRoot string)` - Creates config manager
- `(*Manager) Load()` - Loads config from file
- `(*Manager) Save(config *Config)` - Saves config to file
- `(*Manager) Set(key, value string)` - Sets a config value
- `(*Manager) Get(key string)` - Gets a config value
- `(*Manager) List()` - Returns all config values
- `(*Manager) Exists()` - Checks if config file exists
- `(*Manager) GetPath()` - Returns config file path

**Usage:**
- Used in `cmd/context.go` for managing project context (namespace, network, sender defaults)
- Used in `cmd/run.go` to load default namespace from context

### deploy.go - Legacy Deploy Configuration

**Exported Types:**
- `DeployConfig` - Legacy deploy configuration structure
- `ProfileConfig` - Profile-specific deploy config
- `SenderConfig` - Sender configuration (shared with treb.go)

**Exported Functions:**
- `LoadDeployConfig(projectPath string)` - Loads deploy config from foundry.toml
- `(*DeployConfig) ResolveSenderName(address string)` - Finds sender name by address
- `(*DeployConfig) GetProfileConfig(profile string)` - Gets profile config
- `(*DeployConfig) GetSender(profile, senderName string)` - Gets specific sender
- `(*DeployConfig) Validate(namespace string)` - Validates deploy config
- `(*DeployConfig) ValidateSender(profile, senderName string)` - Validates sender config
- `(*DeployConfig) GenerateEnvVars(namespace string)` - Generates env vars
- `(*DeployConfig) GenerateSenderEnvVars(profile, senderName string)` - Generates sender env vars

**Usage:**
- Used in `cmd/dev.go` for development commands
- Used in `pkg/script/sender_configs.go` (but with TrebConfig instead)
- **NOTE**: This appears to be legacy code that's being replaced by treb.go

### env.go - Environment Variable Loading

**Exported Functions:**
- `LoadEnvFile(filePath string)` - Loads single .env file
- `LoadEnvFiles(filePaths ...string)` - Loads multiple .env files

**Usage:**
- Called internally by deploy.go and treb.go
- **NOT used directly by any other packages**

### foundry.go - Foundry Configuration Management

**Exported Types:**
- `FoundryConfig` - Full foundry.toml structure
- `ProfileFoundryConfig` - Profile-specific foundry config
- `FoundryManager` - Manages foundry.toml file

**Exported Functions:**
- `NewFoundryManager(projectRoot string)` - Creates foundry manager
- `LoadFoundryConfig(projectRoot string)` - Helper to load foundry config
- `(*FoundryManager) Load()` - Loads foundry config
- `(*FoundryManager) Save()` - **DEPRECATED** - should not be used
- `(*FoundryManager) AddLibrary(profile, libraryPath, libraryName, address string)` - Adds library
- `(*FoundryManager) UpdateLibraryAddress(profile, libraryName, newAddress string)` - Updates library
- `(*FoundryManager) GetRemappings()` - Gets forge remappings
- `ParseRemapping(remapping string)` - Parses remapping string
- `(*FoundryManager) GetLibraries(profile string)` - Gets libraries for profile
- `(*FoundryManager) RemoveLibrary(profile, libraryName string)` - Removes library
- `ParseLibraryEntry(entry string)` - Parses library entry
- `(*FoundryManager) AddLibraryAuto(profile, libraryName, address string)` - Auto-adds library

**Usage:**
- Used in `pkg/contracts/indexer.go` for getting remappings
- Library management functions appear to be unused

### ledger.go - Hardware Wallet Support

**Exported Functions:**
- `GetLedgerAddress(derivationPath string)` - Gets address from Ledger
- `GetAddressFromPrivateKey(privateKeyHex string)` - Derives address from private key

**Usage:**
- Called internally by deploy.go for sender address resolution
- **NOT used directly by any other packages**

### treb.go - New Treb Configuration

**Exported Types:**
- `TrebConfig` - Treb-specific configuration within profile
- `FoundryProfileConfig` - Profile config including treb section
- `FoundryFullConfig` - Complete foundry.toml with treb sections

**Exported Functions:**
- `LoadTrebConfig(projectPath string)` - Loads treb config from foundry.toml
- `(*FoundryFullConfig) GetProfileTrebConfig(profileName string)` - Gets treb config for profile
- `(*TrebConfig) GetSenderNameByAddress(address string)` - Finds sender by address

**Usage:**
- Used in `pkg/script/executor.go` for loading sender configurations
- Used in `pkg/script/display.go` for resolving sender names
- Used in `pkg/script/sender_configs.go` for building sender configs
- Used in `cmd/run.go` for configuration management

## Issues and Recommendations

### 1. Duplicate/Overlapping Code
- **deploy.go vs treb.go**: Both handle sender configurations from foundry.toml
  - `DeployConfig` vs `TrebConfig` - duplicate structures
  - `LoadDeployConfig` vs `LoadTrebConfig` - duplicate loading
  - `SenderConfig` is defined in deploy.go but used by both
  - **Recommendation**: Remove deploy.go entirely and migrate cmd/dev.go to use treb.go

### 2. Dead/Unused Code
- `LoadEnvFile` and `LoadEnvFiles` in env.go are only called internally
  - **Recommendation**: Make these unexported (lowercase) internal functions
- Library management functions in foundry.go appear unused:
  - `AddLibrary`, `UpdateLibraryAddress`, `RemoveLibrary`, `AddLibraryAuto`
  - **Recommendation**: Remove if not needed, or document intended usage
- `ParseRemapping` and `ParseLibraryEntry` are unused
  - **Recommendation**: Remove or make internal if only used internally

### 3. Code That Should Move
- `GetAddressFromPrivateKey` and `GetLedgerAddress` in ledger.go
  - These are wallet/crypto utilities, not config management
  - **Recommendation**: Move to a `pkg/wallet` or `pkg/crypto` package
  
### 4. Confusing Naming
- Package has multiple "Config" types that serve different purposes:
  - `Config` - project context settings
  - `DeployConfig` - legacy deployment config
  - `FoundryConfig` - foundry.toml settings
  - `TrebConfig` - new treb-specific config
  - **Recommendation**: Rename types for clarity:
    - `Config` → `ContextConfig` or `ProjectContext`
    - Remove `DeployConfig` (legacy)
    - Keep `FoundryConfig` and `TrebConfig` as-is

### 5. Internal Functions Exposed
- `expandEnvVar` in treb.go should be unexported
- `expandEnvVars`, `expandString` in deploy.go should be unexported
- `findLibrarySourcePath` in foundry.go should be unexported

### 6. Missing Functionality
- No way to validate complete treb configuration (only individual senders)
- No helper to get all configured senders across all profiles
- No way to check if a profile exists without error

## Migration Path

1. **Phase 1**: Clean up obvious issues
   - Make internal functions unexported
   - Remove unused exported functions
   - Fix duplicate SenderConfig definition

2. **Phase 2**: Consolidate sender configuration
   - Migrate cmd/dev.go from DeployConfig to TrebConfig
   - Remove deploy.go entirely
   - Update any remaining references

3. **Phase 3**: Reorganize packages
   - Move wallet utilities to separate package
   - Consider splitting config package into subpackages:
     - `config/context` - project context management
     - `config/foundry` - foundry.toml management
     - `config/treb` - treb-specific configuration
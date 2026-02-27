# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **treb** (Trebuchet) - a Go CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. The architecture follows a "Go orchestrates, Solidity executes" pattern where Go handles configuration, planning, and registry management while all chain interactions happen through proven Foundry scripts.

## Architecture

### Core Components
- **Go CLI** (`internal/`): Orchestration layer with commands for init, run, generate, list, show, verify, and registry management
- **Foundry Library** (`treb-sol/`): Git submodule containing Solidity base contracts and utilities
- **Registry System**: JSON-based deployment tracking with enhanced metadata (versions, verification status, salt/address tracking)
- **CreateX Integration**: Deterministic deployments across chains using CreateX factory

### Key Patterns
- **CreateXOperation/Deployment/Executor**: Extended versions of proven Operation/Deployment/Executor pattern with CreateX integration
- **Salt-based Determinism**: Multi-component salt generation (contract name + environment + label) for consistent addresses
- **Enhanced Registry**: Comprehensive tracking of deployments, verification status, metadata, and broadcast files
- **Script Orchestration**: Go executes `forge script` commands and parses results rather than direct chain interaction

## Development Commands

### Essential Makefile Commands
```bash
# Building and Installation
make build         # Build treb CLI binary to bin/treb
make install       # Install treb globally (copies to /usr/local/bin)
make clean         # Clean build artifacts

# Testing
make unit-test       # Run all unit tests
make integration-test # Run integration tests (builds binaries & contracts first)

# Development Tools
make fmt           # Format Go code with gofumpt
make lint          # Run golangci-lint checks
make dev-setup     # Install required development tools (gofumpt, golangci-lint)
make watch         # Auto-rebuild on file changes (requires fswatch)
make bindings      # Generate Go bindings from treb-sol ABIs

# IMPORTANT: Always run before committing
make fmt             # Format code
make lint            # Check for linting issues
make unit-test       # Run unit tests
make integration-test # Run integration tests
```

### Integration Testing for Feature Development

**IMPORTANT**: When developing features, use integration tests instead of manually running commands. The integration test framework provides a controlled environment with proper setup/teardown, golden file testing, and parallel execution.

#### Why Use Integration Tests?

1. **Isolated Environment**: Each test runs in a temporary directory with fresh state
2. **Reproducible**: Tests use local Anvil chains, no external dependencies
3. **Golden Files**: Expected output is automatically compared and updated
4. **Fast Feedback**: Tests run in parallel with isolated contexts
5. **Debug Support**: Full command output available with debug flag

#### Running Integration Tests

```bash
# Run all integration tests
make integration-test

# Run specific test suite
cd test/integration && go test -run TestListCommand -v

# Run specific test case
go test -run TestListCommand/list_with_all_categories -v

# Run with debug output to see actual command execution
go test -run TestListCommand/list_with_all_categories -v -treb.debug

# Update golden files when output changes are expected
go test -run TestListCommand -v -treb.updategolden

# Keep test artifacts for debugging (normally cleaned up)
go test -run TestListCommand -v -treb.skipcleanup

# Run tests in parallel (faster)
go test -run TestListCommand -v -parallel=10
```

#### Creating Integration Tests

Integration tests are located in `test/integration/`. Each test follows this pattern:

```go
{
    Name: "test_name",
    SetupCmds: [][]string{
        // Commands run before test to set up state
        s("config set network anvil-31337"),
        {"gen", "deploy", "Counter"},
        {"run", "script/deploy/DeployCounter.s.sol"},
    },
    TestCmds: [][]string{
        // Commands to test and capture output
        {"list"},
        {"show", "Counter", "--json"},
    },
    // Optional: Custom normalizers for test output
    Normalizers: []helpers.Normalizer{
        helpers.LegacySolidityNormalizer{}, // For old Solidity versions
    },
    // Optional: Additional files to capture
    OutputArtifacts: []string{".treb/deployments.json"},
    // Optional: Expect command to fail
    ExpectErr: true,
}
```

#### Integration Test Framework Features

- **Test Context**: Each test gets an isolated `TestContext` with:
  - Temporary working directory
  - Fresh Anvil instances (ports 30000-30015)
  - Clean project copy from `test/testdata/project`
  - Isolated configuration

- **Golden Files**: Located in `test/testdata/golden/integration/`
  - Automatically created/updated with `-treb.updategolden`
  - Normalizers remove dynamic content (timestamps, hashes, etc.)
  - Separate files for commands output and artifacts

- **Helper Functions**:
  - `s()`: Marks commands that affect shared state
  - `RunCommands()`: Execute commands and capture output
  - `ReadArtifact()`: Read files from test directory

#### Common Test Patterns

```go
// Deploy a library
{"gen", "deploy", "src/StringUtils.sol:StringUtils"},
{"run", "script/deploy/DeployStringUtils.s.sol"},

// Deploy a proxy with implementation
{"gen", "deploy", "UpgradeableCounter", "--proxy", "--proxy-contract", "ERC1967Proxy.sol:ERC1967Proxy"},
{"run", "DeployUpgradeableCounterProxy"},

// Deploy with environment variables
{"run", "script/deploy/DeployCounter.s.sol", "--env", "LABEL=v1"},

// Test with different namespaces
{"run", "script/deploy/DeployCounter.s.sol", "--namespace", "production"},

// Test JSON output
{"list", "--json"},

// Test error cases
{
    Name: "deploy_already_exists",
    SetupCmds: [][]string{
        {"run", "script/deploy/DeployCounter.s.sol"},
    },
    TestCmds: [][]string{
        {"run", "script/deploy/DeployCounter.s.sol"}, // Should fail
    },
    ExpectErr: true,
}
```

#### Debugging Failed Tests

1. **Run with Debug Flag**: `go test -run TestName -v -treb.debug`
   - Shows full command execution
   - Displays stdout/stderr
   - Shows working directory

2. **Keep Test Artifacts**: `go test -run TestName -v -treb.skipcleanup`
   - Preserves temporary directory
   - Allows inspection of generated files

3. **Check Golden Files**: Compare in `test/testdata/golden/`
   - Look at the diff output
   - Update with `-treb.updategolden` if changes are expected

4. **Add Custom Normalizers**: For platform-specific differences
   - `LegacySolidityNormalizer`: For Solidity <0.8 bytecode
   - `TimestampNormalizer`: Replace timestamps
   - `AddressNormalizer`: Replace addresses

#### Example: Testing a New Feature

```go
// test/integration/myfeature_test.go
func TestMyFeature(t *testing.T) {
    tests := []IntegrationTest{
        {
            Name: "my_feature_basic",
            SetupCmds: [][]string{
                s("config set network anvil-31337"),
                {"gen", "deploy", "MyContract"},
            },
            TestCmds: [][]string{
                {"mycommand", "--flag"},
                {"list"},
            },
        },
    }
    
    RunIntegrationTestSuite(t, tests)
}
```

### Foundry Library (treb-sol)
```bash
# Build library (in treb-sol/)
cd treb-sol && forge build

# Run library tests
cd treb-sol && forge test -vvv

# Test address prediction
cd treb-sol && forge script script/PredictAddress.s.sol --sig "predict(string,string)" "MyContract" "staging"
```

### Project Setup Workflow
```bash
# 1. Create Foundry project
forge init my-project && cd my-project

# 2. Initialize treb
treb init my-project  

# 3. Install treb-sol
forge install trebuchet-org/treb-sol

# 4. Configure environment
cp .env.example .env && edit .env
```

## Key Design Decisions

1. **No Direct Chain Interaction in Go**: All chain operations go through Foundry scripts to maintain proven patterns
2. **Deterministic Addresses**: CreateX + salt components ensure same addresses across chains
3. **Enhanced Registry**: JSON registry tracks deployment metadata, verification status, and provides audit trail
4. **Environment-Aware**: Support for staging/prod deployments with different salt components
5. **Safe Integration**: Support for Safe multisig deployments with transaction tracking

## Deployment Workflow

The `treb run` command follows this flow:

1. **Script Resolution**: Resolve the script reference to a contract artifact
2. **Parameter Resolution**: Extract parameters from natspec annotations and resolve values
3. **Network Resolution**: Load network config from configuration context or CLI flags
4. **Sender Resolution**: Build sender configuration from foundry.toml profiles
5. **Script Execution**: Run forge script with proper environment variables and parameters
6. **Result Hydration**: Parse script output and broadcast files into domain models
7. **Registry Update**: Record deployments and transactions in `.treb/` registry files

## Registry Schema

The registry (`.treb/deployments.json`) is a comprehensive JSON structure tracking:
- Project metadata (name, version, commit, timestamp)
- Per-network deployments with:
  - Contract addresses and deployment type (implementation/proxy)
  - Salt and init code hash for deterministic deployments
  - Verification status and explorer URLs
  - Deployment info (tx hash, block number, broadcast file path)
  - Contract metadata (compiler version, source hash)
  - Support for tags and labels for deployment categorization

## Configuration

### treb.toml Sender Configuration (Recommended)
```toml
# treb.toml — Treb sender configuration
# Each [ns.<name>] section defines a namespace with sender configs.
# The optional 'profile' field maps to a foundry.toml profile (defaults to namespace name).

[ns.default]
profile = "default"

[ns.default.senders.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

[ns.production]
profile = "production"

# Production namespace with Safe multisig sender
[ns.production.senders.safe]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
signer = "proposer"

# Hardware wallet proposer
[ns.production.senders.proposer]
type = "ledger"
derivation_path = "${PROD_PROPOSER_DERIVATION_PATH}"
```

### foundry.toml Sender Profiles (Legacy)
```toml
# NOTE: This format is deprecated. Run `treb migrate-config` to migrate to treb.toml.

# Default profile with private key sender
[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

# Production profile with Safe multisig sender
[profile.production.treb.senders.safe]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
signer = "proposer"

# Hardware wallet proposer
[profile.production.treb.senders.proposer]
type = "ledger"
derivation_path = "${PROD_PROPOSER_DERIVATION_PATH}"
```

### Environment Variables
```bash
DEPLOYER_PRIVATE_KEY=0x...           # Deployment private key
PROD_PROPOSER_DERIVATION_PATH=...    # Ledger derivation path
ETHERSCAN_API_KEY=...                # For contract verification
RPC_URL=...                          # Network RPC endpoint
```

### Run Command Options
```bash
# Run a deployment script
treb run script/deploy/DeployCounter.s.sol

# Run with environment variables
treb run DeployCounter --env LABEL=v1

# Run with specific network and namespace
treb run DeployCounter --network sepolia --namespace production

# Dry run (simulate without broadcasting)
treb run DeployCounter --dry-run
```

## CLI Commands

### Main Commands
- `treb init`: Initialize a new treb project with deployment configuration
- `treb run <script>`: Run a Foundry script with treb infrastructure
- `treb gen deploy <contract>`: Generate a deployment script for a contract
- `treb list`: List all deployments in the registry
- `treb show <contract>`: Show detailed deployment information for a contract
- `treb verify <contract>`: Verify deployed contracts on block explorers
- `treb compose`: Execute orchestrated deployments from a YAML configuration

### Management Commands
- `treb config`: Manage treb local configuration (`set`, `remove` subcommands)
- `treb sync`: Sync deployment registry with on-chain state
- `treb tag <contract> <tag>`: Tag a deployment with a version or label
- `treb register`: Register an existing contract deployment in the registry
- `treb networks`: List available networks from foundry.toml
- `treb prune`: Prune registry entries that no longer exist on-chain
- `treb reset`: Reset all registry entries for the current namespace and network
- `treb dev`: Development utilities (`dev anvil start/stop/restart/status/logs`)
- `treb version`: Show treb version information

## Important File Locations

### Go CLI Structure
- `cli/`: CLI entrypoint
- `internal/cli/`: Cobra command implementations (run.go, list.go, show.go, etc.)
- `internal/cli/render/`: Presentation/rendering layer
- `internal/cli/interactive/`: Interactive selection components
- `internal/usecase/`: Use case implementations (business logic)
- `internal/usecase/ports.go`: Port interfaces for adapters
- `internal/domain/`: Domain models, filters, and configuration types
- `internal/adapters/`: Adapter implementations (forge, blockchain, repository, etc.)
- `internal/app/`: Application wiring (DI with Wire)
- `internal/config/`: Configuration resolution and network management
- `pkg/safe/`: Safe multisig API client

### Solidity Base Contracts (treb-sol/)
- `src/Deployment.sol`: Base deployment contract with structured logging
- `src/ProxyDeployment.sol`: Proxy deployment patterns
- `src/LibraryDeployment.sol`: Library deployment patterns
- `src/internal/`: Internal utilities and registry contracts

### Configuration Files
- `treb.toml`: Treb sender configuration with namespace-based sender profiles (preferred)
- `foundry.toml`: Foundry configuration with RPC endpoints (legacy sender profiles deprecated)
- `.treb/deployments.json`: Deployment registry tracking all deployments
- `.treb/transactions.json`: Transaction registry tracking script executions
- `.treb/config.json`: Local project configuration (namespace, network)
- `.env`: Environment variables for RPC URLs, keys, etc.
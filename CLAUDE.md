# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **treb** (Trebuchet) - a Go CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. The architecture follows a "Go orchestrates, Solidity executes" pattern where Go handles configuration, planning, and registry management while all chain interactions happen through proven Foundry scripts.

## Architecture

### Core Components
- **Go CLI** (`cli/`): Orchestration layer with commands for init, deploy, predict, verify, and registry management
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
make test          # Run all unit tests
make integration   # Run integration tests (builds binaries & contracts first)
make test-all      # Run both unit and integration tests
make golden-test   # Update golden test files (UPDATE_GOLDEN=true make golden-test)

# Development Tools
make fmt           # Format Go code with gofumpt
make lint          # Run golangci-lint checks
make dev-setup     # Install required development tools (gofumpt, golangci-lint)
make watch         # Auto-rebuild on file changes (requires watchman)

# Running treb
make run ARGS="list"                    # Run treb with arguments
make run ARGS="deploy Counter --network sepolia"
make run ARGS="show Counter --json"

# Foundry Integration
make install-forge # Install Foundry if not present
make bindings      # Generate Foundry bindings from treb-sol

# Other Commands
make example       # Create an example project in /tmp
make coverage      # Generate test coverage report

# IMPORTANT: Always run before committing
make fmt           # Format code
make lint          # Check for linting issues  
make test-all      # Run all tests including integration
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
make integration

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

The `treb deploy` command follows this flow:

1. **Validation**: Check deploy configuration in `foundry.toml` under `[deploy.contracts]`
2. **Build**: Execute `forge build` to compile contracts
3. **Script Generation**: Check/generate deploy script in `script/deploy/`
4. **Network Resolution**: Load network config from environment or CLI flags
5. **Script Execution**: Run forge script with proper environment variables
6. **Broadcast Parsing**: Extract deployment results from broadcast files
7. **Registry Update**: Record deployment in `deployments.json` with full metadata

## Registry Schema

The registry (`deployments.json`) is a comprehensive JSON structure tracking:
- Project metadata (name, version, commit, timestamp)
- Per-network deployments with:
  - Contract addresses and deployment type (implementation/proxy)
  - Salt and init code hash for deterministic deployments
  - Verification status and explorer URLs
  - Deployment info (tx hash, block number, broadcast file path)
  - Contract metadata (compiler version, source hash)
  - Support for tags and labels for deployment categorization

## Configuration

### foundry.toml Deploy Profiles
```toml
# Default profile with private key deployment
[profile.default.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

# Staging profile with Safe multisig
[profile.staging.deployer]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
proposer = { type = "private_key", private_key = "${DEPLOYER_PRIVATE_KEY}" }

# Production profile with hardware wallet
[profile.production.deployer]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
proposer = { type = "ledger", derivation_path = "${PROD_PROPOSER_DERIVATION_PATH}" }
```

### Environment Variables
```bash
DEPLOYER_PRIVATE_KEY=0x...           # Deployment private key
PROD_PROPOSER_DERIVATION_PATH=...    # Ledger derivation path
ETHERSCAN_API_KEY=...                # For contract verification
RPC_URL=...                          # Network RPC endpoint
```

### Deploy Command Options
```bash
# Deploy with specific profile
treb deploy Counter --profile staging

# Deploy with custom label (affects salt/address)
treb deploy Counter --label v2

# Deploy with version tag (metadata only)
treb deploy Counter --tag 1.0.0
```

## CLI Commands

### Main Commands
- `treb init <project-name>`: Initialize a new treb project with deployment configuration
- `treb deploy <contract>`: Deploy a contract using its deployment script
- `treb list`: List all deployments in the registry
- `treb show <contract>`: Show detailed deployment information for a contract
- `treb verify <contract>`: Verify deployed contracts on block explorers
- `treb generate <contract>`: Generate deployment script for a contract

### Management Commands  
- `treb tag <contract> <tag>`: Tag a deployment with a version or label
- `treb sync`: Sync deployment registry with on-chain state
- `treb config`: Manage deployment configuration

### Additional Commands
- `treb debug <command>`: Debug deployment issues
- `treb version`: Show treb version information

## Important File Locations

### Go CLI Structure
- `cli/cmd/`: Command implementations (deploy.go, init.go, etc.)
- `cli/pkg/deployment/`: Core deployment logic and execution
- `cli/pkg/forge/`: Forge command execution and integration
- `cli/pkg/registry/`: Deployment registry management
- `cli/pkg/broadcast/`: Broadcast file parsing
- `cli/pkg/verification/`: Contract verification logic
- `cli/pkg/safe/`: Safe multisig integration

### Solidity Base Contracts (treb-sol/)
- `src/Deployment.sol`: Base deployment contract with structured logging
- `src/ProxyDeployment.sol`: Proxy deployment patterns
- `src/LibraryDeployment.sol`: Library deployment patterns
- `src/internal/`: Internal utilities and registry contracts

### Configuration Files
- `foundry.toml`: Foundry configuration with deploy profiles
- `deployments.json`: Deployment registry tracking all deployments
- `.env`: Environment variables for RPC URLs, keys, etc.
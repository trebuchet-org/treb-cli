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

### Go CLI Development
```bash
# Build treb CLI
make build

# Run tests
make test

# Install globally  
make install

# Run locally with arguments
make run ARGS="deploy Counter --network sepolia"

# Create example project
make example

# Setup development environment
make dev-setup

# Clean build artifacts
make clean

# Watch for changes and rebuild automatically
make watch

# Install Foundry if not present
make install-forge

# IMPORTANT: Always run before committing
make fmt      # Format code
make lint     # Check for linting issues
make test     # Run all tests
```

### Foundry Library (treb-sol)
```bash
# Build library (in treb-sol/)
cd treb-sol && forge build

# Run library tests
cd treb-sol && forge test

# Test address prediction
cd treb-sol && forge script script/PredictAddress.s.sol --sig "predict(string,string)" "MyContract" "staging"
```

### Testing Individual Components
```bash
# Run specific Go test
go test -v ./cli/pkg/safe/

# Run Foundry tests with verbose output
cd treb-sol && forge test -vvv

# Test deployment locally with fixture project
cd fixture && treb deploy Counter --network local
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
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **fdeploy** (Forge Deploy) - a Go CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. The architecture follows a "Go orchestrates, Solidity executes" pattern where Go handles configuration, planning, and registry management while all chain interactions happen through proven Foundry scripts.

## Architecture

### Core Components
- **Go CLI** (`cli/`): Orchestration layer with commands for init, deploy, predict, verify, and registry management
- **Foundry Library** (`forge-deploy-lib/`): Git submodule containing Solidity base contracts and utilities
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
# Build fdeploy CLI
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
```

### Foundry Library (forge-deploy-lib)
```bash
# Build library (in forge-deploy-lib/)
cd forge-deploy-lib && forge build

# Run library tests
cd forge-deploy-lib && forge test

# Test address prediction
cd forge-deploy-lib && forge script script/PredictAddress.s.sol --sig "predict(string,string)" "MyContract" "staging"
```

### Project Setup Workflow
```bash
# 1. Create Foundry project
forge init my-project && cd my-project

# 2. Initialize fdeploy
fdeploy init my-project  

# 3. Install forge-deploy-lib
forge install fdeploy-org/forge-deploy

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

The `fdeploy deploy` command follows this flow:

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
fdeploy deploy Counter --profile staging

# Deploy with custom label (affects salt/address)
fdeploy deploy Counter --label v2

# Deploy with version tag (metadata only)
fdeploy deploy Counter --tag 1.0.0
```
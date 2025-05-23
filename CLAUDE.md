# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **fdeploy** (Forge Deploy) - a Go CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. The architecture follows a "Go orchestrates, Solidity executes" pattern where Go handles configuration, planning, and registry management while all chain interactions happen through proven Foundry scripts.

## Architecture

### Core Components
- **Go CLI** (`cli/`): Orchestration layer with commands for init, deploy, predict, verify, and registry management
- **Foundry Library** (`lib/forge-deploy-lib`): Git submodule containing Solidity base contracts and utilities
- **Registry System**: JSON-based deployment tracking with enhanced metadata (versions, verification status, salt/address tracking)
- **CreateX Integration**: Deterministic deployments across chains using CreateX factory

### Key Patterns
- **CreateXOperation/Deployment/Executor**: Extended versions of proven Operation/Deployment/Executor pattern with CreateX integration
- **Salt-based Determinism**: Multi-component salt generation (contract name + version + environment) for consistent addresses
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

# Create example project
make example
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

## Registry Schema

The registry is a comprehensive JSON structure tracking:
- Project metadata (version, commit, timestamp)
- Per-network deployments with addresses, salts, verification status
- Contract metadata (version, source commit, compiler)
- Deployment info (tx hashes, broadcast files, block numbers)
- Support for both implementation and proxy deployments
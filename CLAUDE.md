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

Since this is a planning/design phase project with only a plan.md file, there are no build/test commands yet. When implemented, expect:

```bash
# Go CLI commands
go build -o fdeploy ./cli/cmd
go test ./...

# Foundry library commands (in lib/ subdirectory)
forge build
forge test
forge script script/PredictAddress.s.sol
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
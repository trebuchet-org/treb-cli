![Developers deploying code with a trebuchet](./docs/treb-sol.png)

# treb

🏰 **Trebuchet** - A powerful CLI for orchestrating deterministic smart contract deployments across chains. Because sometimes you need perfect ballistics for your contract launches.

[![Go Report Card](https://goreportcard.com/badge/github.com/trebuchet-org/treb-cli)](https://goreportcard.com/report/github.com/trebuchet-org/treb-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

**treb** is a Go CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. It follows a "Go orchestrates, Solidity executes" pattern where Go handles configuration, planning, and registry management while all chain interactions happen through proven Foundry scripts.

Paired with [treb-sol](https://github.com/trebuchet-org/treb-sol), the Solidity library for deployment scripts, treb provides a complete framework for managing complex multi-chain deployments with various wallet types including hardware wallets and Safe multisigs.

## ✨ Key Features

- 🎯 **Deterministic Deployments**: CreateX-based deployments with predictable addresses across all chains
- 📝 **Script Parameters**: Define and validate script parameters using natspec annotations
- 🔄 **Multi-Sender Support**: EOA, hardware wallets (Ledger/Trezor), and Safe multisig
- 📚 **Registry System**: Comprehensive deployment tracking with metadata and verification status
- 🔍 **Fuzzy Search**: Interactive pickers with fzf-like search for contracts and deployments
- 🛡️ **Type Safety**: Automatic parameter validation and type checking
- 🎨 **Beautiful Output**: Color-coded deployment events and parameter status
- ⚡ **Fast Iteration**: Hot-reload compatible with `forge script --watch`

## Installation

### Using trebup (Installer Script)

```bash
curl -L https://raw.githubusercontent.com/trebuchet-org/treb-cli/main/trebup/install | bash
```

### From Source

```bash
git clone https://github.com/trebuchet-org/treb-cli
cd treb-cli
make install
```

## Quick Start

### 1. Initialize a New Project

```bash
# Create a new Foundry project
forge init my-project && cd my-project

# Initialize treb
treb init

# Install the Solidity library
forge install trebuchet-org/treb-sol
```

### 2. Configure Your Deployment

Edit `foundry.toml`:

```toml
[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

# Safe multisig deployment
[profile.production.treb.senders.deployer]
type = "safe"
safe = "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD40"
proposer = "deployer"

# Hardware wallet deployment
[profile.production.treb.senders.proposer]  
type = "ledger"
derivation_path = "m/44'/60'/0'/0/0"
```

### 3. Write a Deployment Script

```solidity
// script/Deploy.s.sol
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";

contract Deploy is TrebScript {
    using Deployer for Senders.Sender;

    /**
     * @custom:env {uint256} INITIAL_SUPPLY Initial Supply for the contracg
     * @custom:env {string} LABEL Deployment label
     */
    function run() public broadcast {
        // Parameters are automatically validated and prompted if missing
        string memory label = vm.envString("LABEL");
        uint256 initialSupply = vm.envUint("INITIAL_SUPPLY");

        // Deploy with deterministic address
        address token = sender("deployer")
            .create3("src/Token.sol:Token")
            .setLabel(label)
            .deploy(abi.encode("Token", "TK", initialSupply));
    }
}
```

### 4. Deploy Your Contracts

```bash
# Deploy with parameters
treb run Deploy --env label=v1

# Interactive mode - prompts for missing parameters
treb run Deploy

# Deploy to different networks
treb run Deploy --network sepolia --namespace production

# View deployments
treb list

# Show detailed deployment info
treb show Token
```

## 📋 Commands

### Main Commands

- `treb init` - Initialize a new treb project
- `treb run <script>` - Run a Foundry script with treb infrastructure
- `treb gen deploy <contract>` - Generate a deployment script for a contract
- `treb list` - List all deployments in the registry
- `treb show <contract>` - Show detailed deployment information
- `treb verify <contract>` - Verify contracts on block explorers
- `treb compose` - Execute orchestrated deployments from a YAML configuration

### Management Commands

- `treb config` - Manage treb local configuration
- `treb sync` - Sync registry with on-chain state
- `treb tag <contract> <tag>` - Tag a deployment version
- `treb register` - Register an existing contract deployment in the registry
- `treb networks` - List available networks from foundry.toml
- `treb prune` - Prune registry entries that no longer exist on-chain
- `treb reset` - Reset all registry entries for the current namespace and network
- `treb dev` - Development utilities (anvil management)
- `treb version` - Show version information

## 🎯 Script Parameters

treb supports defining script parameters using natspec annotations:

```solidity
/**
 * @custom:env {string} name Parameter description
 * @custom:env {address} owner Owner address  
 * @custom:env {uint256:optional} amount Optional amount
 * @custom:env {sender} deployer Sender to use for deployment
 * @custom:env {deployment} token Reference to existing deployment
 * @custom:env {artifact} implementation Contract artifact to deploy
 */
function run() public {
    string memory name = vm.envString("name");
    address owner = vm.envAddress("owner");
    uint256 amount = vm.envOr("amount", uint256(0));
    // ...
}
```

### Supported Types

**Base Types:**
- `string`, `address`, `uint256`, `int256`, `bytes32`, `bytes`

**Meta Types:**
- `sender` - References a configured sender
- `deployment` - References an existing deployment (e.g., "Token:v1")
- `artifact` - References a contract artifact to deploy

### Parameter Features

- ✅ Automatic validation
- 🔍 Interactive prompts for missing values
- 📝 Optional parameters with `{type:optional}`
- 🚀 Fuzzy search for deployments and artifacts

## 🏗️ Architecture

treb follows a clear separation of concerns:

- **Go CLI**: Orchestration, configuration, registry management
- **Solidity Scripts**: All chain interactions via [treb-sol](https://github.com/trebuchet-org/treb-sol)
- **Registry**: JSON-based deployment tracking with comprehensive metadata
- **CreateX**: Deterministic deployments using the CreateX factory

## 📚 Registry System

The deployment registry (`.treb/deployments.json`) tracks:

- Contract addresses and deployment metadata
- Verification status and explorer links
- Salt and init code hash for deterministic deployments
- Transaction details and gas costs
- Contract metadata (compiler version, optimization settings)

## 🔧 Configuration

### Foundry Profile Configuration

```toml
# Reference by private key in-line for anvil well known accounts
[profile.production.treb.senders.anvil] 
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

# Reference by private key with env var:
[profile.production.treb.senders.local]
type = "private_key"
private_key = "${LOCAL_PK}"

# Hardware wallet
[profile.production.treb.senders.proposer]
type = "ledger"
derivation_path = "${LEDGER_DERIVATION_PATH}"
# Proposer uses hardware wallet

# Safe sender
[profile.production.treb.senders.safe]
type = "safe"
address = 0x....
```

## 🤝 Integration with treb-sol

treb works seamlessly with [treb-sol](https://github.com/trebuchet-org/treb-sol), the Solidity library that provides:

- Base contracts for deployment scripts
- Sender abstraction for multiple wallet types
- Registry integration for cross-contract lookups
- Harness system for secure contract interaction

## 📖 Examples

### Multi-Contract Deployment

```solidity
contract DeploySystem is TrebScript {
    using Deployer for Senders.Sender;

    function run() public broadcast {
        // Deploy Token
        address token = sender("deployer")
            .create3("src/Token.sol:Token")
            .deploy();

        // Deploy Vault using Token address
        address vault = sender("deployer")
            .create3("src/Vault.sol:Vault")
            .deploy(abi.encode(token));

        // Deploy Factory using both
        sender("deployer")
            .create3("src/Factory.sol:Factory")
            .deploy(abi.encode(token, vault));
    }
}
```

### Reference Existing Deployments

```solidity
contract UpgradeSystem is TrebScript {
    /**
     * @custom:env {deployment} oldImpl Current implementation
     * @custom:env {artifact} newImpl New implementation to deploy
     */
    function run() public broadcast {
        address oldImpl = vm.envAddress("oldImpl");
        string memory newImplArtifact = vm.envString("newImpl");

        // Deploy new implementation
        address newImpl = sender("deployer")
            .create3(newImplArtifact)
            .deploy();

        // Upgrade proxy (example)
        IProxy(proxy).upgradeTo(newImpl);
    }
}
```

## 🛠️ Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/trebuchet-org/treb-cli
cd treb-cli

# Build
make build

# Run tests
make test

# Run integration tests
make integration-test

# Install locally
make install
```

### Development Commands

```bash
# Watch for changes and rebuild
make watch

# Run linter
make lint

# Clean build artifacts
make clean
```

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🔗 Links

- [treb-sol](https://github.com/trebuchet-org/treb-sol) - Solidity library for deployment scripts
- [Documentation](https://docs.trebuchet.org) - Full documentation (coming soon)
- [Discord](https://discord.gg/trebuchet) - Join our community (coming soon)

---

Built with ❤️ by the Trebuchet team. Because every smart contract deserves a perfect launch trajectory. 🚀
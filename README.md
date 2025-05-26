# treb

**Trebuchet** - Foundry Script Orchestration with CreateX for deterministic smart contract deployments.

## Overview

treb is a CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. It follows a "Go orchestrates, Solidity executes" pattern where:

- **Go CLI** handles configuration, planning, registry management, and library resolution
- **Foundry scripts** handle all chain interactions using proven patterns
- **CreateX** provides deterministic addresses across chains
- **Automatic library injection** resolves and deploys dependencies on demand

## Features

- 🎯 **Deterministic Deployments**: Same addresses across chains using CreateX
- 📚 **Automatic Library Management**: Detects, deploys, and links libraries automatically
- 📊 **Enhanced Registry**: Comprehensive deployment tracking with metadata
- 🔍 **Address Prediction**: Predict addresses before deployment
- ✅ **Verification Management**: Automated contract verification
- 🛡️ **Safe Integration**: Support for multisig deployments
- ⚙️ **Configuration Management**: Simple config with `treb config`
- 🌐 **Multi-environment**: Deploy across different environments (staging, prod)

## Installation

### Easy Installation (Recommended)

```bash
# Install trebup (treb version manager)
curl -L https://raw.githubusercontent.com/trebuchet-org/treb-cli/main/trebup/install | bash

# Restart your terminal or source your shell config
source ~/.bashrc  # or ~/.zshenv

# Install latest treb
trebup
```

### Manual Installation

```bash
# Requires Go 1.21+ and Foundry
git clone https://github.com/trebuchet-org/treb-cli
cd treb-cli
make install
```

## Quick Start

### 1. Initialize Foundry project

```bash
forge init my-protocol
cd my-protocol
```

### 2. Initialize treb

```bash
treb init my-protocol
```

### 3. Install treb-sol

```bash
# Install the deployment library
forge install trebuchet-org/treb-sol
```

### 4. Set up configuration

```bash
# Set default network and environment
treb config set network sepolia
treb config set environment staging

# Set up your .env file with RPC URLs and keys
echo 'SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY' >> .env
echo 'DEPLOYER_PRIVATE_KEY=0x...' >> .env
echo 'ETHERSCAN_API_KEY=...' >> .env
```

### 5. Generate deployment script

```bash
# Generate deployment script automatically
treb gen deploy MyToken
```

This creates a deployment script like:

```solidity
// script/deploy/DeployMyToken.s.sol
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployMyToken
 * @notice Deployment script for MyToken contract
 * @dev Generated automatically by treb
 */
contract DeployMyToken is Deployment {
    constructor() Deployment(
        "src/MyToken.sol:MyToken",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure override returns (bytes memory) {
        // Constructor arguments detected from ABI
        string memory _name = "My Token";
        string memory _symbol = "MTK";
        uint256 _totalSupply = 1000000e18;
        return abi.encode(_name, _symbol, _totalSupply);
    }
}
```

### 6. Deploy contracts

```bash
# Predict address before deployment
treb deploy MyToken --predict

# Deploy to configured network
treb deploy MyToken

# Deploy to specific network with verification
treb deploy MyToken --network mainnet --verify

# Deploy with custom environment and label
treb deploy MyToken --env production --label v1.0
```

## Commands

### Project Management
```bash
treb init <project-name>        # Initialize treb in existing Foundry project
```

### Configuration
```bash
treb config                     # Show current configuration
treb config set <key> <value>   # Set configuration value
treb config init                # Initialize configuration file
```

### Deployment
```bash
treb deploy <contract>          # Deploy contract via Foundry script
treb deploy <contract> --predict # Predict deployment address
treb verify <contract>          # Verify contracts on explorers
```

### Registry & Information
```bash
treb list                       # List all deployments
treb list --libraries           # List deployed libraries  
treb show <contract>            # Show deployment details
treb tag <contract>             # Tag a deployment with version
```

### Version Management
```bash
treb version                    # Show treb version
trebup                          # Install/update treb (if using trebup)
trebup --list                   # List installed treb versions
```

## Library Management

treb automatically handles library dependencies:

```bash
# Libraries are detected and deployed automatically
treb deploy MyContract  # Deploys any required libraries first

# View deployed libraries
treb list --libraries

# Libraries are reused across deployments on the same chain
```

Example with a contract that uses libraries:
```solidity
import {StringUtils} from "./StringUtils.sol";

contract MyContract {
    using StringUtils for string;
    
    function process(string memory input) public pure returns (string memory) {
        return input.toUpperCase();  // Uses StringUtils library
    }
}
```

When deploying `MyContract`, treb will automatically:
1. Detect that `StringUtils` is required
2. Check if it's already deployed on the target chain
3. Deploy it if missing (or reuse existing deployment)
4. Link it when deploying `MyContract`

## Project Structure

```
your-project/
├── src/                        # Smart contracts
│   ├── MyToken.sol
│   └── libraries/
│       └── StringUtils.sol
├── script/
│   └── deploy/                 # Deployment scripts
│       ├── DeployMyToken.s.sol
│       └── DeployStringUtils.s.sol  # Auto-generated if needed
├── test/                       # Tests
├── lib/
│   ├── forge-std/              # Foundry standard library  
│   └── treb-sol/               # Treb deployment library
├── deployments.json            # Central deployment registry
├── foundry.toml               # Foundry & treb configuration
├── .treb                      # Treb configuration (optional)
├── remappings.txt             # Import remappings
└── .env                       # Environment variables
```

## Configuration

### Foundry Configuration (`foundry.toml`)

```toml
# Deployment configuration
[deploy.contracts]
MyToken = { script = "script/deploy/DeployMyToken.s.sol" }

# Profile-specific deployer configuration
[profile.default.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

[profile.production.deployer]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
proposer = { type = "private_key", private_key = "${DEPLOYER_PRIVATE_KEY}" }
```

### Treb Configuration (`.treb`)

```json
{
  "environment": "staging",
  "network": "sepolia",
  "verify": true
}
```

## Deterministic Deployments

treb uses CreateX with salt components for deterministic addresses:

- **Contract name**: Ensures different contracts get different addresses
- **Environment**: Separates staging/production deployments  
- **Label**: Optional versioning (e.g., "v1.0", "beta")

Same salt = same address across all chains.

## Development

```bash
# Setup development environment
make dev-setup

# Build CLI
make build

# Run tests
make test

# Create example project for testing
make example

# Run linting
make lint

# Clean artifacts
make clean
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make test` and `make lint`
6. Submit a pull request

## Status

🔧 **Active Development** - Core features are stable and usable. New features and improvements are continuously being added.

## License

MIT
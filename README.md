# fdeploy

**Forge Deploy** - Foundry Script Orchestration with CreateX for deterministic smart contract deployments.

## Overview

fdeploy is a CLI tool that orchestrates Foundry script execution for deterministic smart contract deployments using CreateX. It follows a "Go orchestrates, Solidity executes" pattern where:

- **Go** handles configuration, planning, and registry management
- **Foundry scripts** handle all chain interactions using proven patterns
- **CreateX** provides deterministic addresses across chains

## Features

- 🎯 **Deterministic Deployments**: Same addresses across chains using CreateX
- 📊 **Enhanced Registry**: Comprehensive deployment tracking with metadata
- 🔍 **Address Prediction**: Predict addresses before deployment
- ✅ **Verification Management**: Automated contract verification
- 🛡️ **Safe Integration**: Support for multisig deployments
- 🔄 **Multi-chain**: Deploy to multiple networks simultaneously

## Installation

### Prerequisites

- [Go 1.21+](https://golang.org/doc/install)
- [Foundry](https://book.getfoundry.sh/getting-started/installation)

### Install fdeploy

```bash
# Clone the repository
git clone https://github.com/bogdan/fdeploy
cd fdeploy

# Build and install
make install
```

## Quick Start

### 1. Initialize Foundry project

```bash
forge init my-protocol
cd my-protocol
```

### 2. Initialize fdeploy

```bash
fdeploy init my-protocol
```

### 3. Install forge-deploy-lib

```bash
# Install the deployment library
forge install your-org/forge-deploy-lib
```

### 4. Set up environment

```bash
# Copy and configure environment
cp .env.example .env
# Edit .env with your RPC URLs, private keys, and API keys
```

### 5. Create deployment script

```solidity
// script/DeployMyToken.s.sol
pragma solidity ^0.8.0;

import "forge-deploy-lib/base/CreateXDeployment.sol";
import "../src/MyToken.sol";

contract DeployMyToken is CreateXDeployment {
    constructor() CreateXDeployment(
        "MyToken",
        "v1.0.0", 
        _buildSaltComponents()
    ) {}
    
    function _buildSaltComponents() private pure returns (string[] memory) {
        string[] memory components = new string[](3);
        components[0] = "MyToken";
        components[1] = "v1.0.0";
        components[2] = vm.envString("DEPLOYMENT_ENV");
        return components;
    }
    
    function deployContract() internal override returns (address) {
        return address(new MyToken("My Token", "MTK", 1000000e18));
    }
    
    function getInitCode() internal pure override returns (bytes memory) {
        return abi.encodePacked(
            type(MyToken).creationCode,
            abi.encode("My Token", "MTK", 1000000e18)
        );
    }
}
```

### 6. Deploy contracts

```bash
# Predict address
fdeploy predict MyToken --env staging

# Deploy to staging
fdeploy deploy MyToken --env staging --verify

# Deploy to production across multiple chains
fdeploy deploy MyToken --env prod --networks mainnet,polygon,arbitrum --verify
```

## Commands

### Project Management
```bash
fdeploy init <project-name>     # Initialize fdeploy in existing Foundry project
```

### Deployment
```bash
fdeploy predict <contract>      # Predict deployment address
fdeploy deploy <contract>       # Deploy contract via Foundry script
fdeploy verify                  # Verify contracts on explorers
```

### Registry Management
```bash
fdeploy registry show <contract>   # Show deployment info
fdeploy registry sync              # Sync from broadcast files
```

## Registry Structure

The registry (`deployments.json`) tracks comprehensive deployment information:

```json
{
  "project": {
    "name": "my-protocol",
    "version": "1.0.0",
    "commit": "abc123",
    "timestamp": "2025-05-23T10:30:00Z"
  },
  "networks": {
    "1": {
      "name": "mainnet", 
      "deployments": {
        "MyToken_prod": {
          "address": "0x1234...abcd",
          "type": "implementation",
          "salt": "0xabcd...1234",
          "verification": {
            "status": "verified",
            "explorerUrl": "https://etherscan.io/address/0x1234...abcd#code"
          },
          "deployment": {
            "txHash": "0x789a...bcde",
            "blockNumber": 12345678,
            "broadcastFile": "broadcast/DeployMyToken.s.sol/1/run-latest.json"
          }
        }
      }
    }
  }
}
```

## Architecture

### Project Structure
```
your-project/
├── src/                    # Smart contracts
├── script/                 # Deployment scripts  
├── test/                   # Tests
├── lib/
│   ├── forge-std/          # Foundry standard library
│   └── forge-deploy-lib/   # fdeploy library
├── deployments/            # Per-chain deployment files
├── deployments.json        # Central registry
├── foundry.toml           # Foundry config
├── remappings.txt         # Import remappings
└── .env                   # Environment variables
```

### Salt Components

Deterministic addresses are generated using salt components:
- Contract name (e.g., "MyToken")
- Version (e.g., "v1.0.0") 
- Environment (e.g., "staging", "prod")

This ensures consistent addresses across chains while allowing environment separation.

## Development

```bash
# Setup development environment
make dev-setup

# Build
make build

# Run tests
make test

# Create example project
make example
```

## Status

🚧 **Early Development** - This project is in active development. Core functionality is being implemented.

## License

MIT
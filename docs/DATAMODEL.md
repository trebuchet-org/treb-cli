# Treb Data Model

This document describes the data model for Treb's deployment registry system.

## Overview

The registry tracks smart contract deployments across multiple chains and environments (namespaces). Data is stored in multiple JSON files within the `.treb/` directory for separation of concerns and efficient access patterns.

## File Structure

```
.treb/
├── deployments.json   # Deployment records
├── transactions.json  # Transaction records
├── safe-txs.json     # Safe transaction batches
├── lookup.json       # Indexes and lookups
└── registry.json     # Simplified registry for Solidity
```

## Deployment ID Format

Deployment IDs follow a hierarchical format that ensures uniqueness across parallel deployments:

```
<namespace>/<chain-id>/<contract-name>:<label>(#<tx-prefix>)
```

Examples:
- `production/1/Counter:v1` - First deployment with label "v1"
- `staging/1/Counter:v2#ab12` - Second deployment with same label (includes tx prefix)
- `test/31337/Token:usdc` - Test deployment on local chain

The transaction prefix (`#ab12`) is only added when needed for uniqueness (multiple deployments with same namespace/chain/contract/label combination). It uses the first 4 bytes of the transaction ID.

## Data Models

### 1. Deployment (`deployments.json`)

```json
{
  "production/1/Counter:v1": {
    "id": "production/1/Counter:v1",
    "namespace": "production",
    "chainId": 1,
    "contractName": "Counter",
    "label": "v1",
    "address": "0x1234567890123456789012345678901234567890",
    "type": "SINGLETON",
    "transactionId": "tx-0x1234abcd...",
    
    "deploymentStrategy": {
      "method": "CREATE2",
      "salt": "0x...",
      "initCodeHash": "0x...",
      "factory": "0x...",
      "constructorArgs": "0x...",
      "entropy": "production:Counter:v1"
    },
    
    "proxyInfo": null,
    
    "artifact": {
      "path": "src/Counter.sol:Counter",
      "compilerVersion": "0.8.19",
      "sourceHash": "0x...",
      "bytecodeHash": "0x..."
    },
    
    "verification": {
      "status": "VERIFIED",
      "etherscanUrl": "https://etherscan.io/address/0x1234...",
      "verifiedAt": "2024-01-15T10:30:00Z"
    },
    
    "tags": ["v1.0.0", "release"],
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-15T10:30:00Z"
  },
  
  "production/1/CounterProxy:main": {
    "id": "production/1/CounterProxy:main",
    "namespace": "production",
    "chainId": 1,
    "contractName": "CounterProxy",
    "label": "main",
    "address": "0xABCD567890123456789012345678901234567890",
    "type": "PROXY",
    "transactionId": "tx-0x5678efgh...",
    
    "proxyInfo": {
      "type": "ERC1967",
      "implementation": "0x1234567890123456789012345678901234567890",
      "admin": "0x...",
      "history": [
        {
          "implementationId": "production/1/Counter:v1",
          "upgradedAt": "2024-01-15T10:00:00Z",
          "upgradeTxId": "tx-0x5678efgh..."
        }
      ]
    }
  }
}
```

### 2. Transaction (`transactions.json`)

```json
{
  "tx-0x1234abcd...": {
    "id": "tx-0x1234abcd...",
    "chainId": 1,
    "hash": "0x1234abcd...",
    "blockNumber": 18500000,
    "status": "EXECUTED",
    "sender": "0xDeployer...",
    "nonce": 42,
    
    "deployments": ["production/1/Counter:v1"],
    
    "operations": [
      {
        "type": "DEPLOY",
        "target": "0x1234567890123456789012345678901234567890",
        "method": "CREATE2",
        "result": {
          "address": "0x1234567890123456789012345678901234567890",
          "gasUsed": 500000
        }
      }
    ],
    
    "safeContext": null,
    
    "environment": "production",
    "createdAt": "2024-01-15T10:00:00Z"
  },
  
  "tx-0x5678efgh...": {
    "id": "tx-0x5678efgh...",
    "chainId": 1,
    "hash": "0x5678efgh...",
    "blockNumber": 18500100,
    "status": "EXECUTED",
    "sender": "0xSafe...",
    
    "deployments": ["production/1/CounterProxy:main"],
    
    "safeContext": {
      "safeAddress": "0xSafe...",
      "safeTxHash": "0xsafetx123...",
      "batchIndex": 0,
      "proposerAddress": "0xProposer..."
    }
  }
}
```

### 3. Safe Transaction Batch (`safe-txs.json`)

```json
{
  "0xsafetx123...": {
    "safeTxHash": "0xsafetx123...",
    "safeAddress": "0xSafe...",
    "chainId": 1,
    "status": "EXECUTED",
    "nonce": 5,
    
    "transactions": [
      {
        "to": "0xCreateX...",
        "value": "0",
        "data": "0x...",
        "operation": 0
      }
    ],
    
    "transactionIds": ["tx-0x5678efgh..."],
    
    "proposedBy": "0xProposer...",
    "proposedAt": "2024-01-15T09:00:00Z",
    
    "confirmations": [
      {
        "signer": "0xSigner1...",
        "signature": "0x...",
        "confirmedAt": "2024-01-15T09:10:00Z"
      },
      {
        "signer": "0xSigner2...",
        "signature": "0x...",
        "confirmedAt": "2024-01-15T09:15:00Z"
      }
    ],
    
    "executedAt": "2024-01-15T10:00:00Z",
    "executionTxHash": "0x5678efgh..."
  }
}
```

### 4. Lookup Indexes (`lookup.json`)

```json
{
  "version": "1.0.0",
  
  "byAddress": {
    "1": {
      "0x1234567890123456789012345678901234567890": "production/1/Counter:v1",
      "0xABCD567890123456789012345678901234567890": "production/1/CounterProxy:main"
    }
  },
  
  "byNamespace": {
    "production": {
      "1": ["production/1/Counter:v1", "production/1/CounterProxy:main"],
      "137": ["production/137/Counter:v1"]
    },
    "staging": {
      "1": ["staging/1/Counter:test"]
    }
  },
  
  "byContract": {
    "Counter": ["production/1/Counter:v1", "staging/1/Counter:test"],
    "CounterProxy": ["production/1/CounterProxy:main"]
  },
  
  "proxies": {
    "implementations": {
      "production/1/Counter:v1": ["production/1/CounterProxy:main"]
    },
    "proxyToImpl": {
      "production/1/CounterProxy:main": "production/1/Counter:v1"
    }
  },
  
  "pending": {
    "safeTxs": ["0xsafetx456..."]
  }
}
```

### 5. Solidity Registry (`registry.json`)

Simplified format for Solidity contract consumption:

```json
{
  "1": {
    "production": {
      "Counter:v1": "0x1234567890123456789012345678901234567890",
      "CounterProxy:main": "0xABCD567890123456789012345678901234567890"
    },
    "staging": {
      "Counter:test": "0xDEF0123456789012345678901234567890ABCD"
    }
  },
  "137": {
    "production": {
      "Counter:v1": "0x1234567890123456789012345678901234567890"
    }
  }
}
```

## Key Design Principles

1. **Namespace-First**: Namespaces are the top-level organizing principle, allowing staging/production/test deployments across chains.

2. **Deterministic IDs**: IDs are generated from namespace/chain/contract/label, with tx prefix added only for uniqueness.

3. **Separation of Concerns**: Different aspects stored in separate files for efficiency and clarity.

4. **Parallel Deployment Support**: Transaction prefix ensures unique IDs even when multiple users deploy simultaneously.

5. **Solidity Compatibility**: Simplified registry format that can be easily parsed in Solidity contracts.

6. **Rich Metadata**: Full deployment context preserved for debugging and auditing.

## Usage Patterns

### Deployment Lookup
```go
// Direct lookup by ID
deployment := deployments["production/1/Counter:v1"]

// Find by address using lookup index
deploymentId := lookup.byAddress["1"]["0x1234..."]
deployment := deployments[deploymentId]

// Get all deployments in namespace
productionDeployments := lookup.byNamespace["production"]["1"]
```

### Safe Transaction Tracking
```go
// Check pending Safe transactions
for _, safeTxHash := range lookup.pending.safeTxs {
    safeTx := safeTxs[safeTxHash]
    if safeTx.status == "PENDING" {
        // Check with Safe API for updates
    }
}
```

### Solidity Registry Access
```solidity
// In Solidity, read registry.json
address counter = registry[1]["production"]["Counter:v1"];
```

## Migration Strategy

When migrating from the current model:

1. Parse existing `deployments.json`
2. Extract transactions from broadcast files
3. Generate new IDs based on namespace/chain/contract/label
4. Build lookup indexes
5. Create simplified Solidity registry
6. Store in new `.treb/` structure
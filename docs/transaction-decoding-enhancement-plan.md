# Transaction Decoding Enhancement Plan

## UPDATE: Implementation Complete

This enhancement has been implemented using a cleaner approach than originally planned. Instead of pre-loading all ABIs, the solution implements on-demand ABI resolution directly in the transaction decoder.

## Problem Statement

The `treb run` command currently only decodes transactions for contracts deployed during the script execution. It cannot decode transactions to/from previously deployed contracts that are registered in the deployments registry. This limits the utility of the transaction display, as interactions with existing contracts show raw hex data instead of human-readable function calls and parameters.

## Current State Analysis

### Transaction Decoder Flow
1. **Initialization**: Creates empty maps for contract ABIs and proxy relationships
2. **Runtime Registration**: During script execution, when `ContractDeployed` events occur:
   - Extracts artifact name from event
   - Uses indexer to find contract info
   - Loads ABI from artifact path
   - Registers ABI with decoder
3. **Transaction Decoding**: When decoding a transaction:
   - Looks up ABI by contract address
   - If proxy, follows relationship to implementation
   - Decodes function selector and parameters
   - Returns formatted transaction info

### Limitations
- Only knows about contracts deployed in current execution
- No access to historical deployments from registry
- Cannot decode transactions to existing infrastructure contracts
- Misses important context about contract interactions

## Implemented Solution

The final implementation uses a cleaner architecture than originally proposed:

### Architecture

1. **ABIResolver Interface** (`cli/pkg/abi/decoder.go`):
   - Added an `ABIResolver` interface to the transaction decoder
   - The decoder now accepts an optional resolver for on-demand ABI loading
   - When decoding fails with registered ABIs, it falls back to the resolver

2. **RegistryABIResolver** (`cli/pkg/abi/abi_resolver.go`):
   - Implements the `ABIResolver` interface
   - Uses interfaces to avoid circular dependencies
   - Uses the registry manager to look up deployments by address
   - Uses the contracts indexer to get artifact information
   - Loads ABIs on-demand from artifact files
   - Handles proxy relationships automatically

3. **Adapter Pattern** (`cli/pkg/script/indexer_adapter.go`):
   - Provides adapters to wrap concrete types with the required interfaces
   - Allows the abi package to remain independent of other packages
   - Clean separation of concerns

4. **Integration** (`cli/cmd/run.go`):
   - The run command creates a registry manager
   - Passes it to the enhanced display via `SetRegistryResolver`
   - The enhanced display uses adapters to configure the transaction decoder

### Key Benefits

- **On-Demand Loading**: ABIs are only loaded when needed, not upfront
- **Memory Efficient**: No need to cache all ABIs in memory
- **Clean Separation**: ABI resolution logic is encapsulated in the decoder
- **Reusable**: The ABIResolver interface can be implemented differently if needed
- **No Chain ID Lookups**: Uses the chain ID already available from network resolution

## Original Proposed Solution (Not Implemented)

### High-Level Design
Enhance the transaction decoder initialization to load ABIs from the deployments registry:

1. **On Initialization**:
   - Load deployments.json for current chain
   - For each deployment, load ABI from artifact path
   - Register contract ABIs and proxy relationships
   - Cache loaded ABIs for performance

2. **Integration Points**:
   - Modify `NewEnhancedEventDisplay` to accept registry path
   - Add method to load ABIs from registry deployments
   - Ensure chain-specific filtering
   - Handle missing/invalid artifact files gracefully

### Detailed Implementation

#### 1. Add Registry Loading to Enhanced Display

```go
// In enhanced_display.go
func (d *EnhancedEventDisplay) LoadABIsFromRegistry(registryPath string, chainID uint64) error {
    // Load registry
    registry, err := registry.LoadRegistry(registryPath)
    if err != nil {
        return fmt.Errorf("failed to load registry: %w", err)
    }
    
    // Filter deployments for current chain
    deployments := registry.GetDeploymentsForChain(chainID)
    
    // Load ABI for each deployment
    for _, deployment := range deployments {
        if err := d.loadDeploymentABI(deployment); err != nil {
            // Log warning but continue - don't fail entirely
            if d.verbose {
                fmt.Printf("Warning: Failed to load ABI for %s: %v\n", 
                    deployment.Artifact.Path, err)
            }
        }
    }
    
    return nil
}

func (d *EnhancedEventDisplay) loadDeploymentABI(deployment *types.Deployment) error {
    // Convert artifact path to file path
    artifactPath := filepath.Join("out", deployment.Artifact.Path + ".json")
    
    // Load ABI from artifact file
    abiJSON := d.loadABIFromPath(artifactPath)
    if abiJSON == "" {
        return fmt.Errorf("no ABI found in artifact file")
    }
    
    // Register contract ABI
    contractName := d.extractContractName(deployment.Artifact.Path)
    err := d.transactionDecoder.RegisterContract(
        deployment.Address, 
        contractName, 
        abiJSON,
    )
    if err != nil {
        return fmt.Errorf("failed to register ABI: %w", err)
    }
    
    // Register in known addresses for reconciliation
    d.knownAddresses[deployment.Address] = contractName
    d.deployedContracts[deployment.Address] = contractName
    
    // Handle proxy relationships
    if deployment.Type == types.DeploymentTypeProxy && deployment.Implementation != nil {
        d.transactionDecoder.RegisterProxyRelationship(
            deployment.Address,
            *deployment.Implementation,
        )
    }
    
    return nil
}
```

#### 2. Modify Run Command Integration

```go
// In cmd/run.go
// After creating enhanced display
enhancedDisplay := script.NewEnhancedEventDisplay(indexer)

// Load ABIs from registry if available
if _, err := os.Stat("deployments.json"); err == nil {
    chainID := getChainIDFromNetwork(network) // Helper to get chain ID
    if err := enhancedDisplay.LoadABIsFromRegistry(".", chainID); err != nil {
        // Log warning but don't fail
        fmt.Printf("Warning: Could not load ABIs from registry: %v\n", err)
    }
}
```

#### 3. Add Chain ID Resolution

```go
// Helper function to get chain ID from network config
func getChainIDFromNetwork(network string) uint64 {
    // Try to get from environment or config
    // This needs to integrate with existing network resolution
    // Default chain IDs for common networks:
    switch network {
    case "mainnet":
        return 1
    case "sepolia":
        return 11155111
    case "base":
        return 8453
    case "base-sepolia":
        return 84532
    default:
        // Try to resolve from RPC or config
        return 0
    }
}
```

## Edge Cases and Complexity

### 1. Missing Artifact Files
- **Issue**: Artifact file may be deleted or moved
- **Solution**: Log warning and continue, don't fail entire process
- **Impact**: Some transactions won't be decoded

### 2. Changed ABIs
- **Issue**: Contract may have been upgraded with different ABI
- **Solution**: Registry tracks versions, use latest deployment for address
- **Complexity**: Need to handle multiple deployments at same address

### 3. Large Registries
- **Issue**: Projects with many deployments could slow initialization
- **Solution**: 
  - Lazy loading: Only load ABIs when needed
  - Caching: Store parsed ABIs in memory
  - Parallel loading: Load multiple ABIs concurrently

### 4. Cross-Chain Deployments
- **Issue**: Same contract deployed on multiple chains
- **Solution**: Filter by chain ID when loading from registry
- **Complexity**: Need reliable chain ID detection

### 5. Proxy Patterns
- **Issue**: Various proxy patterns (UUPS, Transparent, Beacon)
- **Solution**: Registry already tracks proxy relationships
- **Complexity**: Ensure all proxy types are handled correctly

### 6. Library Deployments
- **Issue**: Libraries have different calling patterns
- **Solution**: Registry tracks deployment type, handle accordingly
- **Note**: Libraries typically use DELEGATECALL

### 7. Contract Verification
- **Issue**: Unverified contracts may have incomplete metadata
- **Solution**: Gracefully handle missing data
- **Impact**: May affect decoding accuracy

## Performance Considerations

1. **Startup Time**: Loading many ABIs could slow initialization
   - Mitigation: Load in background, use goroutines
   
2. **Memory Usage**: Large ABIs consume memory
   - Mitigation: Only keep frequently used ABIs in memory
   
3. **File I/O**: Reading many artifact files
   - Mitigation: Batch operations, use OS file caching

## Testing Strategy

1. **Unit Tests**:
   - Test ABI loading from various artifact formats
   - Test chain ID filtering
   - Test error handling for missing files

2. **Integration Tests**:
   - Test with real registry and artifact files
   - Test transaction decoding with loaded ABIs
   - Test proxy transaction decoding

3. **E2E Tests**:
   - Run scripts that interact with existing contracts
   - Verify transaction display shows decoded data
   - Test with various network configurations

## Migration Path

1. **Phase 1**: Implement basic registry loading
   - Load ABIs at startup
   - Test with small registries
   
2. **Phase 2**: Add optimizations
   - Implement lazy loading
   - Add caching layer
   
3. **Phase 3**: Enhanced features
   - Support for library calls
   - Better proxy pattern support
   - Performance monitoring

## Alternative Approaches Considered

1. **On-Demand Loading**: Load ABIs only when transaction is encountered
   - Pros: Lower memory usage, faster startup
   - Cons: Slower first decode, more complex

2. **External ABI Service**: Fetch ABIs from Etherscan/Sourcify
   - Pros: Always up-to-date, no local storage
   - Cons: Network dependency, API limits

3. **Pre-compiled ABI Cache**: Generate binary ABI cache
   - Pros: Very fast loading
   - Cons: Complex build process, version management

## Recommended Approach

Start with the basic implementation (load all ABIs at startup) and iterate based on real-world performance. The registry already contains all necessary data, so the implementation is straightforward. Focus on error handling and graceful degradation to ensure the feature doesn't break existing functionality.

## Success Metrics

1. **Functionality**: Can decode transactions to any contract in registry
2. **Performance**: Startup time increase < 100ms for typical projects
3. **Reliability**: Gracefully handles missing/invalid data
4. **User Experience**: Clear, readable transaction displays

## Next Steps

1. Wait for parallel changes to complete
2. Implement basic registry loading
3. Add comprehensive error handling
4. Test with various project sizes
5. Optimize based on performance data
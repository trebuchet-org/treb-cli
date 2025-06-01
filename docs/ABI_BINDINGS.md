# ABI Bindings Migration Plan - Complete Type Replacement

This document outlines a direct migration plan from custom event parsing to auto-generated ABI bindings, completely replacing custom types with generated ones.

## Overview

We will completely remove custom event types and parsing logic in favor of auto-generated ABI bindings. This is a breaking change that provides maximum benefit with minimal long-term maintenance.

## Current vs Target State

### Current State
- Custom event types in `cli/pkg/events/types.go`
- Manual parsing in `cli/pkg/script/events.go`
- ~1000 lines of error-prone parsing code
- Duplicate type definitions

### Target State
- Use only auto-generated types from `cli/pkg/abi/treb/generated.go`
- No custom event types
- Type conversion only at registry boundaries
- Single source of truth for event structures

## JSON to types.Log Conversion

The key to using auto-generated bindings is converting forge's JSON output to go-ethereum's `types.Log`:

```go
// cli/pkg/script/utils.go
func ConvertRawLogToTypesLog(raw RawLog) *types.Log {
    // Convert hex string data to bytes
    data, _ := hex.DecodeString(strings.TrimPrefix(raw.Data, "0x"))
    
    return &types.Log{
        Address: raw.Address,
        Topics:  raw.Topics,
        Data:    data,
        // Blockchain context (not needed for parsing)
        BlockNumber: 0,
        TxHash:      common.Hash{},
        TxIndex:     0,
        BlockHash:   common.Hash{},
        Index:       0,
        Removed:     false,
    }
}
```

## Direct Migration Plan

### Step 1: Replace Event Types (Breaking Change)

**Remove all custom types** from `cli/pkg/events/types.go`:
- `DeployingContractEvent`
- `ContractDeployedEvent`
- `SafeTransactionQueuedEvent`
- `TransactionSimulatedEvent`
- `TransactionFailedEvent`
- `TransactionBroadcastEvent`
- `EventDeployment`
- `Transaction`
- `RichTransaction`

**Use generated types directly**:
```go
// Before
type DeployingContractEvent struct {
    What         string
    Label        string
    InitCodeHash common.Hash
}

// After - use treb.TrebDeployingContract directly
```

### Step 2: Update ParseAllEvents

Replace the entire parsing logic:

```go
// cli/pkg/script/parser.go
func ParseAllEvents(output *ForgeScriptOutput) ([]interface{}, error) {
    trebContract := treb.NewTreb()
    var events []interface{}
    
    for _, rawLog := range output.RawLogs {
        if len(rawLog.Topics) == 0 {
            continue
        }
        
        // Convert to types.Log for unpacking
        typesLog := ConvertRawLogToTypesLog(rawLog)
        
        // Try each event type
        eventSig := rawLog.Topics[0]
        
        // Use the ABI's event IDs directly
        switch eventSig {
        case trebContract.abi.Events["DeployingContract"].ID:
            if event, err := trebContract.UnpackDeployingContractEvent(typesLog); err == nil {
                events = append(events, event)
            }
            
        case trebContract.abi.Events["ContractDeployed"].ID:
            if event, err := trebContract.UnpackContractDeployedEvent(typesLog); err == nil {
                events = append(events, event)
            }
            
        case trebContract.abi.Events["SafeTransactionQueued"].ID:
            if event, err := trebContract.UnpackSafeTransactionQueuedEvent(typesLog); err == nil {
                events = append(events, event)
            }
            
        case trebContract.abi.Events["TransactionSimulated"].ID:
            if event, err := trebContract.UnpackTransactionSimulatedEvent(typesLog); err == nil {
                events = append(events, event)
            }
            
        case trebContract.abi.Events["TransactionFailed"].ID:
            if event, err := trebContract.UnpackTransactionFailedEvent(typesLog); err == nil {
                events = append(events, event)
            }
            
        case trebContract.abi.Events["TransactionBroadcast"].ID:
            if event, err := trebContract.UnpackTransactionBroadcastEvent(typesLog); err == nil {
                events = append(events, event)
            }
        }
    }
    
    return events, nil
}
```

### Step 3: Update Event Consumers

Update all code that consumes events to use type switches:

```go
// Example: registry update
func UpdateRegistryFromEvents(events []interface{}, ...) error {
    for _, event := range events {
        switch e := event.(type) {
        case *treb.TrebContractDeployed:
            // Convert only at registry boundary
            deployment := convertToRegistryDeployment(e)
            registry.RecordDeployment(deployment)
            
        case *treb.TrebSafeTransactionQueued:
            // Handle safe transactions
            handleSafeTransaction(e)
        }
    }
}

// Conversion only happens at registry boundaries
func convertToRegistryDeployment(e *treb.TrebContractDeployed) *types.Deployment {
    return &types.Deployment{
        Address:      e.Location.Hex(),
        Deployer:     e.Deployer.Hex(),
        TransactionID: common.BytesToHash(e.TransactionId[:]).Hex(),
        
        // Convert DeployerEventDeployment to registry format
        ContractName: e.Deployment.Artifact,
        Label:        e.Deployment.Label,
        Salt:         common.BytesToHash(e.Deployment.Salt[:]).Hex(),
        BytecodeHash: common.BytesToHash(e.Deployment.BytecodeHash[:]).Hex(),
        InitCodeHash: common.BytesToHash(e.Deployment.InitCodeHash[:]).Hex(),
        ConstructorArgs: hex.EncodeToString(e.Deployment.ConstructorArgs),
        CreateStrategy: e.Deployment.CreateStrategy,
    }
}
```

### Step 4: Update Display/Output Functions

```go
// cli/pkg/script/display.go
func DisplayEvent(event interface{}) string {
    switch e := event.(type) {
    case *treb.TrebDeployingContract:
        return fmt.Sprintf("🚀 Deploying %s (label: %s)", e.What, e.Label)
        
    case *treb.TrebContractDeployed:
        return fmt.Sprintf("✅ Deployed at %s by %s", 
            e.Location.Hex(), 
            e.Deployer.Hex()[:10])
            
    case *treb.TrebTransactionBroadcast:
        return fmt.Sprintf("📡 Transaction %s: %s → %s", 
            hex.EncodeToString(e.TransactionId[:8]), 
            e.Sender.Hex()[:10], 
            e.To.Hex())
    }
    return "Unknown event"
}
```

### Step 5: Clean Up

1. **Delete files**:
   - `cli/pkg/events/types.go` (entire file)
   - All manual parsing functions in `cli/pkg/script/events.go`

2. **Update imports**:
   ```go
   // Remove
   import "github.com/trebuchet-org/treb-cli/cli/pkg/events"
   
   // Add
   import "github.com/trebuchet-org/treb-cli/cli/pkg/abi/treb"
   ```

3. **Remove re-exports** in `cli/pkg/script/events.go`

## Type Mapping Reference

| Old Type | New Type | Notes |
|----------|----------|-------|
| `events.DeployingContractEvent` | `treb.TrebDeployingContract` | Direct replacement |
| `events.ContractDeployedEvent` | `treb.TrebContractDeployed` | Direct replacement |
| `events.EventDeployment` | `treb.DeployerEventDeployment` | Nested in ContractDeployed |
| `events.Transaction` | `treb.Transaction` | Direct replacement |
| `events.RichTransaction` | `treb.RichTransaction` | Direct replacement |
| `common.Hash` fields | `[32]byte` fields | Use `common.BytesToHash()` when needed |

## Benefits of This Approach

1. **No Duplicate Types**: Single source of truth from Solidity
2. **Automatic Updates**: Regenerate when contracts change
3. **Type Safety**: Compiler catches mismatches
4. **Less Code**: Remove ~1000 lines of parsing logic
5. **Better Performance**: No reflection or manual unpacking

## Implementation Timeline

- **Day 1**: Update ParseAllEvents to use generated unpackers
- **Day 2**: Update all event consumers to use type switches
- **Day 3**: Update registry conversions and display functions
- **Day 4**: Delete old code and test everything
- **Day 5**: Update documentation and merge

## Testing Strategy

```go
// Test that parsing produces expected results
func TestEventParsing(t *testing.T) {
    // Load test fixture with known events
    output := loadTestOutput("testdata/forge-output.json")
    
    // Parse with new system
    events, err := ParseAllEvents(output)
    require.NoError(t, err)
    
    // Verify events
    require.Len(t, events, 3)
    
    // Type assert and check values
    deployed, ok := events[0].(*treb.TrebContractDeployed)
    require.True(t, ok)
    require.Equal(t, "Counter", deployed.Deployment.Artifact)
}
```

## Maintenance

When contracts change:
1. Update ABI JSON file
2. Regenerate bindings: `go generate ./cli/pkg/abi/...`
3. Fix any compilation errors (new fields, changed types)
4. Update registry converters if needed

## Regenerating Bindings

```bash
# Generate new bindings with updated package name
abigen --abi=path/to/abi.json \
       --pkg=treb \
       --type=Treb \
       --out=cli/pkg/abi/treb/generated.go

# Run tests to ensure compatibility
go test ./cli/pkg/script/...
```

## Conclusion

This aggressive approach provides maximum benefit by fully embracing code generation. While it's a breaking change, it results in a cleaner, more maintainable codebase with strong type safety.
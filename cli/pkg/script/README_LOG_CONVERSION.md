# Log Conversion Guide

This document explains how to convert between different log formats in the treb-cli project.

## Log Type Overview

### 1. RawLog (from Forge JSON output)
```go
type RawLog struct {
    Address common.Address `json:"address"`
    Topics  []common.Hash  `json:"topics"`
    Data    string         `json:"data"` // Hex-encoded string with "0x" prefix
}
```

### 2. events.Log (internal format)
```go
type Log struct {
    Address common.Address `json:"address"`
    Topics  []common.Hash  `json:"topics"`
    Data    string         `json:"data"` // Hex-encoded string
}
```

### 3. types.Log (go-ethereum format)
```go
type Log struct {
    Address     common.Address
    Topics      []common.Hash
    Data        []byte         // Raw bytes, not hex string
    
    // Additional blockchain context fields:
    BlockNumber uint64
    TxHash      common.Hash
    TxIndex     uint
    BlockHash   common.Hash
    Index       uint
    Removed     bool
}
```

## Conversion Functions

Two converter functions are available in `cli/pkg/script/utils.go`:

### ConvertRawLogToTypesLog
Converts from Forge's RawLog format to go-ethereum's types.Log:
```go
func ConvertRawLogToTypesLog(rawLog RawLog) (*types.Log, error)
```

### ConvertEventsLogToTypesLog
Converts from internal events.Log format to go-ethereum's types.Log:
```go
func ConvertEventsLogToTypesLog(log events.Log) (*types.Log, error)
```

## Usage with Auto-Generated Unpack Functions

The auto-generated code in `cli/pkg/abi/treb_sol/generated.go` expects `types.Log` format:

```go
// Example: Using converter with auto-generated unpacker
rawLog := RawLog{
    Address: contractAddress,
    Topics:  []common.Hash{eventTopic, ...},
    Data:    "0x...", // hex encoded event data
}

// Convert to types.Log
typesLog, err := ConvertRawLogToTypesLog(rawLog)
if err != nil {
    return err
}

// Use auto-generated unpacker
trebSol := &treb_sol.TrebSol{} // Initialize with ABI
event, err := trebSol.UnpackContractDeployedEvent(typesLog)
if err != nil {
    return err
}

// Access typed event data
fmt.Printf("Contract deployed at: %s\n", event.Location.Hex())
fmt.Printf("Label: %s\n", event.Deployment.Label)
```

## Key Differences

1. **Data Field Format**:
   - RawLog/events.Log: Hex string (e.g., "0x1234...")
   - types.Log: Byte array

2. **Blockchain Context**:
   - RawLog/events.Log: No blockchain context
   - types.Log: Includes block number, tx hash, etc. (set to zero values when converting)

3. **Use Cases**:
   - RawLog: Parsing Forge script output
   - events.Log: Internal event processing
   - types.Log: Integration with go-ethereum tools and auto-generated code

## Migration Path

To use auto-generated unpack functions instead of manual parsing:

1. Convert RawLog to types.Log using `ConvertRawLogToTypesLog`
2. Initialize the auto-generated contract binding (e.g., `treb_sol.TrebSol`)
3. Call the appropriate `Unpack*Event` method
4. Convert the unpacked data to internal event types if needed

This approach provides type safety and reduces manual parsing code while maintaining compatibility with the existing event processing pipeline.
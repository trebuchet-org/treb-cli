# Forge Package Consolidation

This document describes the consolidation of forge command execution and output parsing into the `forge` package.

## Overview

All Foundry forge script execution and parsing logic has been consolidated into the `forge` package to:
- Reduce code duplication
- Centralize forge command handling
- Provide a cleaner API for script execution
- Enable reuse across different parts of the codebase

## New Structure

### forge Package Components

1. **forge.go** - Core forge command execution (existing)
   - `Forge` struct for basic forge operations
   - `Build()`, `RunScript()`, `RunScriptWithArgs()` methods
   - Error parsing utilities

2. **script_runner.go** - Enhanced script execution (new)
   - `ScriptRunner` struct for advanced script execution
   - `ScriptOptions` for configuration
   - `ScriptResult` with parsed output
   - JSON output parsing and debug support

3. **event_parser.go** - Event parsing from logs (new)
   - `EventParser` struct for parsing forge events
   - Support for both Treb events and proxy events
   - Type-safe event parsing using generated bindings

## API Changes

### Old Pattern (script package)
```go
// In script/executor.go
executor := script.NewExecutor(projectPath, network)
result, err := executor.Run(script.RunOptions{...})

// In script/parser.go
output, err := script.ParseForgeOutput(rawOutput)
events, err := script.ParseAllEvents(output)
```

### New Pattern (forge package)
```go
// Option 1: Using ScriptRunner directly
runner := forge.NewScriptRunner(projectPath)
result, err := runner.Run(forge.ScriptOptions{...})

// Option 2: Using ExecutorV2 (maintains compatibility)
executor := script.NewExecutorV2(projectPath, network)
result, err := executor.Run(script.RunOptions{...})

// Event parsing
parser := forge.NewEventParser()
events, err := parser.ParseEvents(result.ParsedOutput.ScriptOutput)
```

## Migration Guide

### For Commands Using Script Execution

1. If you need full compatibility, use `script.ExecutorV2` which wraps the new forge functionality
2. For new code, use `forge.ScriptRunner` directly

### For Event Parsing

1. Replace `script.ParseForgeOutput()` with `forge.ScriptRunner.ParseOutput()`
2. Replace `script.ParseAllEvents()` with `forge.EventParser.ParseEvents()`
3. Use `forge.ExtractDeploymentEvents()` to filter deployment events

### Type Changes

- `script.RawLog` → `forge.EventLog`
- `script.ForgeScriptOutput` → `forge.ScriptOutput`
- `script.ParsedForgeOutput` → `forge.ParsedOutput`

## Benefits

1. **Centralization**: All forge-related logic in one package
2. **Reusability**: Can be used by any package without importing script
3. **Separation of Concerns**: forge package handles forge, script package handles treb-specific logic
4. **Better Testing**: Easier to test forge functionality in isolation
5. **Cleaner API**: More structured options and results

## Future Work

1. Migrate all direct uses of `script.Executor` to use the new API
2. Move more forge-specific utilities (like broadcast parsing) to the forge package
3. Add more forge commands support (test, coverage, etc.)
4. Enhanced error types for better error handling
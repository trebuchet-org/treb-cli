# Forge Output Processing Implementation

## Overview

The forge package now supports two modes of output processing for script execution:

1. **Debug Mode** (`opts.Debug = true` and `opts.JSON = false`)
   - Direct copy of PTY output to stdout
   - Preserves colors and formatting
   - No parsing or processing

2. **Normal Mode** (all other cases)
   - Real-time line-by-line processing
   - Stage tracking with visual spinner
   - Entity parsing and collection via channels
   - Unparsed lines saved to debug directory

## Key Components

### 1. PTY Support
- Uses `github.com/creack/pty` for proper color handling
- Ensures ANSI color codes are preserved in output

### 2. OutputProcessor (`script_output_processor.go`)
- Processes output line-by-line in real-time
- Tracks execution stages with timing
- Shows multi-line spinner with stage progress
- Saves unparsed lines to `.treb-debug/runID/ignored-lineN.txt`

### 3. Stage Tracking
Current stages:
- `Initializing` - Script startup
- `Compiling` - Contract compilation  
- `Simulating` - Transaction simulation
- `Broadcasting` - Transaction broadcast
- `Completed` - Execution finished

Stage display format:
```
✓ Initializing (123ms) → ✓ Compiling (2.5s) → ● Simulating (5s) → ○ Broadcasting
```

### 4. Entity Channel System
Parsed entities are sent through a channel for collection:
- `ScriptOutput` - Main forge output with events
- `GasEstimate` - Gas estimation data
- `StatusOutput` - Status with broadcast path
- `ConsoleLog` - Console log messages
- `UnknownJSON` - Unrecognized JSON objects

### 5. Debug Directory Structure
```
out/.treb-debug/
└── <timestamp>/
    ├── ignored-line1.txt
    ├── ignored-line2.txt
    └── ...
```

## Usage

```go
forge := NewForge(projectRoot)
result, err := forge.Run(ScriptOptions{
    ScriptPath: "script/Deploy.s.sol",
    Debug: false,      // Enable normal mode with processing
    JSON: true,        // Request JSON output from forge
    Broadcast: true,   // Enable broadcasting
})

// result.ParsedOutput contains:
// - ScriptOutput with events
// - GasEstimate 
// - StatusOutput
// - ConsoleLogs array
```

## Stage Detection

Stages are detected in two ways:

1. **Content-based**: Scanning line content for keywords like "compiling", "simulating", "broadcasting"
2. **Entity-based**: When specific JSON entities are parsed, stages are updated accordingly

## Future Improvements

1. **Better Stage Detection**: Add more specific markers for stage transitions
2. **Progress Indicators**: Show percentage complete for each stage
3. **Error Handling**: Better display of errors per stage
4. **Custom Stages**: Allow configuration of additional stages
5. **ANSI Stripping**: Improve the ANSI color code removal for parsing
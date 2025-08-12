# Treb CLI Test Suite

This directory contains the comprehensive test suite for the Treb CLI. The tests are organized into several categories to make them easier to manage and debug.

## Directory Structure

```
test/
├── helpers/           # Common test utilities and helpers
│   ├── context.go     # TrebContext for running commands with different binary versions
│   ├── globals.go     # Global test variables and paths
│   ├── isolation.go   # Test isolation utilities
│   └── anvil_manager.go # Anvil node management
│
├── golden/           # Golden file tests
│   └── framework.go  # Golden test framework with version support
│
├── compatibility/    # v1 to v2 compatibility tests
│   ├── framework.go  # Compatibility test framework
│   ├── list_test.go  # List command compatibility
│   └── show_test.go  # Show command compatibility
│
├── integrations/     # Integration tests
│   └── *.go         # Various integration test files
│
└── testdata/        # Test fixtures and golden files
    ├── project/     # Test Foundry project
    └── golden/      # Golden output files
```

## Binary Version Support

The test suite supports testing against both v1 and v2 binaries:

### TrebContext

The `TrebContext` now includes a `BinaryVersion` field that determines which binary to use:

```go
// Create a v1 context
ctx := helpers.NewTrebContext(t, helpers.BinaryV1)

// Create a v2 context
ctx := helpers.NewTrebContext(t, helpers.BinaryV2)
```

### Environment Variables

- `TREB_TEST_BINARY`: Controls which binary version to use for tests
  - `v1` (default): Use the v1 binary
  - `v2`: Use the v2 binary

- `TREB_TEST_V2_GOLDEN`: When set to "true", golden tests will look for v2-specific golden files
  - Example: `list.golden` → `list.v2.golden`

- `UPDATE_GOLDEN`: When set to "true", updates golden files instead of comparing

- `TREB_TEST_DEBUG`: When set, prints detailed command execution logs

## Test Categories

### Helpers (`helpers/`)

Common utilities used across all test types:
- `TrebContext`: Manages command execution with version support
- `AnvilManager`: Manages multiple Anvil nodes for testing
- `IsolatedTest`: Provides test isolation with snapshots

### Golden Tests (`golden/`)

Tests that compare command output against expected "golden" files:
- Support for v1 and v2 specific golden files
- Automatic output normalization (timestamps, hashes, etc.)
- Update mode for regenerating golden files

### Compatibility Tests (`compatibility/`)

Tests that ensure v1 and v2 behave consistently:
- Can run commands with both versions and compare outputs
- Support for skipping v2 when commands aren't migrated yet
- Dedicated framework for easy compatibility testing

### Integration Tests (`integrations/`)

Full integration tests for various features:
- Deployment workflows
- Multi-chain scenarios
- Proxy relationships
- Registry management

## Running Tests

### Run all tests
```bash
make integration-test
```

### Run specific test categories
```bash
# Golden tests only
go test ./test/golden/...

# Compatibility tests only
go test ./test/compatibility/...

# Integration tests only
go test ./test/integrations/...
```

### Run with specific binary version
```bash
# Run all tests with v2 binary
TREB_TEST_BINARY=v2 go test ./test/...

# Run golden tests with v2-specific golden files
TREB_TEST_BINARY=v2 TREB_TEST_V2_GOLDEN=true go test ./test/golden/...
```

### Update golden files
```bash
UPDATE_GOLDEN=true go test ./test/golden/...
```

### Debug mode
```bash
TREB_TEST_DEBUG=1 go test -v ./test/...
```

## Writing Tests

### Compatibility Test Example

```go
func TestListCommandCompatibility(t *testing.T) {
    helpers.InitGlobals()
    manager := setupAnvilManager(t)
    defer teardownAnvilManager(t, manager)

    // Compare v1 and v2 outputs
    CompareOutputs(t, manager, []string{"list"}, nil)
}
```

### Golden Test Example

```go
func TestListCommands(t *testing.T) {
    manager := setupAnvilManager(t)
    defer teardownAnvilManager(t, manager)

    tests := []GoldenTest{
        {
            Name:       "list_empty",
            Args:       []string{"list"},
            GoldenFile: "commands/list/empty.golden",
        },
    }

    RunGoldenTests(t, manager, tests)
}
```

### Integration Test Example

```go
func TestDeploymentWorkflow(t *testing.T) {
    helpers.InitGlobals()
    manager := helpers.NewAnvilManager(t)
    defer manager.StopAll()

    helpers.IsolatedTest(t, manager, "deploy_counter", func(t *testing.T, ctx *helpers.TrebContext) {
        // Test implementation
        output, err := ctx.Treb("list")
        // ... assertions
    })
}
```
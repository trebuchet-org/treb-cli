# Treb CLI Test Suite

This directory contains the comprehensive test suite for the Treb CLI. The tests are organized into multiple packages to ensure proper isolation, maintainability, and clarity.

## Overview

The test suite is structured to support:
- **Golden file testing** for CLI output verification
- **Integration testing** for end-to-end workflows
- **Compatibility testing** between v1 and v2 binaries
- **Isolated test execution** with automatic cleanup
- **Global resource management** (Anvil nodes)

## Directory Structure

```
test/
├── helpers/              # Shared test utilities and infrastructure
│   ├── anvil_manager.go  # Global Anvil node management
│   ├── context.go        # TrebContext for command execution
│   ├── isolation.go      # Test isolation with snapshots
│   ├── main.go          # Global test setup and binary paths
│   ├── normalizers.go    # Output normalizers for consistent testing
│   └── setup.go         # Test initialization helpers
│
├── golden/              # Golden file tests for CLI output
│   ├── framework.go     # Golden test framework
│   ├── main_test.go     # Test setup
│   └── *_test.go        # Various golden test files
│
├── integration/         # Integration tests for complex workflows
│   ├── main_test.go     # Test setup
│   └── *_test.go        # Various integration test files
│
├── compatibility/       # v1/v2 compatibility tests
│   ├── framework.go     # Compatibility test framework
│   ├── main_test.go     # Test setup
│   └── *_test.go        # Various compatibility test files
│
└── testdata/           # Test fixtures and data
    ├── project/        # Test Foundry project
    └── golden/         # Golden output files
        ├── commands/   # Command-specific golden files
        └── workflows/  # Workflow golden files
```

## Running Tests

### Run All Tests
```bash
# Run all tests with proper isolation (must use -p=1 to prevent parallelization)
go test ./... -p=1

# Or use make command
make integration-test
```

### Run Specific Test Packages
```bash
# Golden tests only
go test ./golden

# Integration tests only
go test ./integration

# Compatibility tests only
go test ./compatibility
```

### Run with Debugging
```bash
# Enable debug output
TREB_TEST_DEBUG=1 go test ./golden -v

# Run specific test with verbose output
go test ./golden -v -run "TestShowCommandGolden/show_counter"
```

### Update Golden Files
```bash
# Update all golden files
UPDATE_GOLDEN=true go test ./golden

# Update specific golden test
UPDATE_GOLDEN=true go test ./golden -v -run "TestShowCommandGolden/show_counter"
```

## Test Infrastructure

### Global Test Setup

The test suite uses a global setup pattern where:
1. Binaries are built once at the start
2. Anvil nodes are started globally and reused across tests
3. Each test runs in isolation with snapshots

This is managed through `TestMain` functions in each package that initialize the global resources.

### Test Isolation

Each test runs in complete isolation using the `IsolatedTest` helper:
- Creates an Anvil snapshot before the test
- Cleans all test artifacts
- Reverts to the snapshot after the test
- Ensures no test state leaks between runs

Example:
```go
func TestDeployment(t *testing.T) {
    helpers.IsolatedTest(t, "deploy_counter", func(t *testing.T, ctx *helpers.TrebContext) {
        // Test implementation
        output, err := ctx.Treb("deploy", "Counter")
        require.NoError(t, err)
        // ... assertions
    })
}
```

### Binary Version Support

The test suite supports testing against both v1 and v2 binaries:

```go
// Test with default version (from TREB_TEST_BINARY env var)
ctx := helpers.NewTrebContext(t, helpers.GetBinaryVersionFromEnv())

// Test with specific version
ctxV1 := helpers.NewTrebContext(t, helpers.BinaryV1)
ctxV2 := helpers.NewTrebContext(t, helpers.BinaryV2)
```

Environment variable:
- `TREB_TEST_BINARY`: Set to "v1" or "v2" (default: "v1")

## Test Types

### Golden Tests

Golden tests capture CLI output and compare against expected "golden" files. They ensure backward compatibility and consistent output formatting.

Example:
```go
func TestShowCommandGolden(t *testing.T) {
    tests := []golden.GoldenTest{
        {
            Name: "show_counter",
            Setup: func(t *testing.T, ctx *helpers.TrebContext) {
                // Deploy Counter first
                _, err := ctx.Treb("gen", "deploy", "src/Counter.sol:Counter")
                require.NoError(t, err)
                _, err = ctx.Treb("run", "script/deploy/DeployCounter.s.sol")
                require.NoError(t, err)
            },
            TestCmds: [][]string{
                {"show", "Counter"},
            },
            GoldenFile: "commands/show/counter.golden",
        },
    }
    
    golden.RunGoldenTests(t, tests)
}
```

Golden files support:
- Multiple commands in sequence
- Setup functions or commands
- Error expectations
- Custom normalizers

### Integration Tests

Integration tests verify complete workflows and complex scenarios.

Example:
```go
func TestMultiChainDeployment(t *testing.T) {
    helpers.IsolatedTest(t, "multi_chain", func(t *testing.T, ctx *helpers.TrebContext) {
        // Deploy to first chain
        _, err := ctx.Treb("deploy", "Counter", "--network", "anvil-31337")
        require.NoError(t, err)
        
        // Deploy to second chain
        _, err = ctx.Treb("deploy", "Counter", "--network", "anvil-31338")
        require.NoError(t, err)
        
        // Verify deployments
        output, err := ctx.Treb("list")
        require.NoError(t, err)
        assert.Contains(t, output, "31337")
        assert.Contains(t, output, "31338")
    })
}
```

### Compatibility Tests

Compatibility tests ensure v1 and v2 binaries produce consistent outputs.

Example:
```go
func TestListCompatibility(t *testing.T) {
    test := compatibility.CompatibilityTest{
        Name: "list_with_deployments",
        Setup: func(t *testing.T, ctx *helpers.TrebContext) {
            // Deploy Counter (only with v1 context)
            _, err := ctx.Treb("gen", "deploy", "src/Counter.sol:Counter")
            require.NoError(t, err)
            _, err = ctx.Treb("run", "script/deploy/DeployCounter.s.sol")
            require.NoError(t, err)
        },
        TestCmds: [][]string{
            {"list"},
        },
    }
    
    compatibility.RunCompatibilityTest(t, test)
}
```

## Output Normalizers

Normalizers ensure consistent test output by replacing dynamic content with placeholders:

- **TimestampNormalizer**: Replaces timestamps with `<TIMESTAMP>`
- **VersionNormalizer**: Replaces version strings with `v<VERSION>`
- **TargetedGitCommitNormalizer**: Replaces git commits in specific contexts
- **TargetedHashNormalizer**: Replaces transaction/bytecode hashes in specific contexts
- **ColorNormalizer**: Removes ANSI color codes

**Note**: We intentionally don't normalize addresses as they should be deterministic in our deployments.

Default normalizers are automatically applied. Custom normalizers can be added:
```go
test := golden.GoldenTest{
    Name: "custom_test",
    Normalizers: []helpers.Normalizer{
        helpers.ColorNormalizer{},
        helpers.TimestampNormalizer{},
        // Add custom normalizer
        MyCustomNormalizer{},
    },
}
```

## Best Practices

1. **Test Isolation**: Always use `IsolatedTest` to ensure clean state
2. **Deterministic Tests**: Use consistent test data and avoid time-dependent logic
3. **Golden File Updates**: Review changes carefully before updating golden files
4. **Resource Cleanup**: Tests automatically clean up, but avoid creating files outside the fixture directory
5. **Sequential Execution**: Run tests with `-p=1` to prevent parallel execution conflicts
6. **Descriptive Names**: Use clear, descriptive test names that explain what's being tested
7. **Error Messages**: Include helpful context in test failures

## Environment Variables

- `TREB_TEST_BINARY`: Binary version to test ("v1" or "v2", default: "v1")
- `UPDATE_GOLDEN`: Update golden files instead of comparing (default: false)
- `TREB_TEST_DEBUG`: Enable debug logging (default: false)
- `TREB_TEST_V2_GOLDEN`: Use v2-specific golden files (default: false)

## Troubleshooting

### Tests Failing Due to Parallel Execution
Always run with `-p=1` to prevent parallel execution:
```bash
go test ./... -p=1
```

### Golden File Mismatches
1. Check if the change is intentional
2. Review the diff carefully
3. Update if correct: `UPDATE_GOLDEN=true go test ./golden -v -run "TestName"`

### Anvil Node Issues
The test suite manages Anvil nodes automatically. If you encounter issues:
1. Check if ports 8545/9545 are available
2. Ensure `anvil` is in your PATH
3. Check logs in `/tmp/treb-anvil-*.log`

### Binary Not Found
Ensure binaries are built before running tests:
```bash
make build
```

## Writing New Tests

### Adding a Golden Test
1. Create test function in appropriate `*_test.go` file
2. Define test cases with setup, commands, and golden file path
3. Run with `UPDATE_GOLDEN=true` to create initial golden file
4. Review and commit the golden file

### Adding an Integration Test
1. Create test function using `IsolatedTest`
2. Implement test logic with proper assertions
3. Ensure cleanup happens automatically

### Adding a Compatibility Test
1. Create test using `CompatibilityTest` struct
2. Define setup that works with v1 context
3. Test commands that should work with both versions
4. Framework automatically compares outputs

## CI Integration

The test suite runs in CI with:
1. Binary building
2. Sequential test execution
3. Golden file verification
4. Failure reporting with diffs

To update golden files in CI:
1. Run tests locally with `UPDATE_GOLDEN=true`
2. Review and commit updated golden files
3. Push changes with your PR
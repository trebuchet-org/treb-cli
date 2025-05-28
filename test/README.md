# Treb CLI Integration Tests

This directory contains integration tests for the Treb CLI using Go's standard testing framework with [testify](https://github.com/stretchr/testify) assertions.

## Running Tests

### Run all integration tests
```bash
make integration-test
```

### Run specific test
```bash
make integration-test-run TEST=TestDeploymentFlow
```

### Run with coverage
```bash
make integration-test-coverage
```

### Run tests directly with go test
```bash
cd test
go test -v -timeout=10m
```

## Test Structure

The tests use standard Go testing patterns:

- **TestMain**: Sets up the test environment (builds treb, starts anvil, builds contracts)
- **Table-driven tests**: For parameterized testing (see `TestBasicCommands`, `TestGenerateCommands`)
- **Subtests**: For organizing related tests (see `TestShowAndList`)
- **Test helpers**: `runTreb()` for executing CLI commands with timeout
- **Cleanup**: `cleanupGeneratedFiles()` ensures clean state between tests

## Test Coverage

The integration tests cover:

1. **Basic Commands** - Version, help, list, error handling
2. **Non-Interactive Mode** - Ensures all prompts can be bypassed
3. **Generate Commands** - Script generation for deployments and proxies
4. **Deployment Flow** - Full generate → deploy → show workflow
5. **Show and List** - Viewing deployments with filters
6. **Verify Command** - Contract verification behavior
7. **Command Structure** - Ensures all commands and subcommands exist

## Requirements

- Go 1.21+
- Foundry (forge, anvil)
- The test fixture project in `test/fixture/`

## Test Environment

The tests automatically:
1. Build the treb binary
2. Start an anvil node on port 8545
3. Build contracts in the fixture project
4. Run tests with 30-second timeout per command
5. Clean up generated files and stop anvil

## Writing New Tests

Follow Go testing best practices:

```go
func TestNewFeature(t *testing.T) {
    // Setup
    cleanupGeneratedFiles(t)
    
    // Execute
    output, err := runTreb(t, "new-command", "--flag")
    
    // Assert
    require.NoError(t, err)
    assert.Contains(t, output, "expected text")
}
```

For table-driven tests:

```go
func TestFeatureVariations(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantErr  bool
        contains string
    }{
        {
            name:     "valid case",
            args:     []string{"command", "--flag"},
            contains: "success",
        },
        {
            name:    "error case",
            args:    []string{"command", "--invalid"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            output, err := runTreb(t, tt.args...)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            if tt.contains != "" {
                assert.Contains(t, output, tt.contains)
            }
        })
    }
}
```

## CI Integration

The tests are run in CI via GitHub Actions. See `.github/workflows/integration-tests.yml` for the CI configuration.
# Phase 0: Integration Testing Infrastructure Implementation

## Overview

This phase establishes a comprehensive testing framework that ensures backwards compatibility throughout the migration. It must be completed before any architectural changes begin.

## Implementation Steps

### 1. Project Structure Setup

```bash
# Create testing directories
mkdir -p test/integration/{fixtures,helpers,suites}
mkdir -p testdata/fixtures/{commands,workflows,snapshots}

# Structure:
test/
├── integration/
│   ├── fixtures/          # Test data and setup
│   ├── helpers/           # Testing utilities
│   └── suites/           # Test suites by feature
└── testdata/
    └── fixtures/
        ├── commands/      # Golden files per command
        ├── workflows/     # Multi-command scenarios
        └── snapshots/     # Complex output snapshots
```

### 2. Core Testing Framework

Create `test/integration/helpers/cli_test.go`:

```go
package helpers

import (
    "bytes"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
    
    "github.com/google/go-cmp/cmp"
    "github.com/google/go-cmp/cmp/cmpopts"
)

// CLITest represents a CLI command test case
type CLITest struct {
    Name        string
    Description string
    Command     []string
    Args        []string
    Env         map[string]string
    WorkDir     string
    Stdin       string
    Setup       func(t *testing.T) error
    Teardown    func(t *testing.T) error
    GoldenFile  string
    Normalizers []OutputNormalizer
    
    // Expected behavior
    ExitCode    int
    StdoutEmpty bool
    StderrEmpty bool
}

// OutputNormalizer processes output before comparison
type OutputNormalizer interface {
    Normalize(output string) string
    Description() string
}

// TimestampNormalizer replaces timestamps with placeholders
type TimestampNormalizer struct{}

func (n TimestampNormalizer) Normalize(output string) string {
    // Match various timestamp formats
    patterns := []string{
        `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`,  // ISO
        `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,   // Standard
        `\d{1,2} \w+ ago`,                        // Relative
    }
    
    result := output
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        result = re.ReplaceAllString(result, "<TIMESTAMP>")
    }
    return result
}

func (n TimestampNormalizer) Description() string {
    return "normalize timestamps"
}

// AddressNormalizer replaces Ethereum addresses
type AddressNormalizer struct{}

func (n AddressNormalizer) Normalize(output string) string {
    re := regexp.MustCompile(`0x[a-fA-F0-9]{40}`)
    return re.ReplaceAllString(output, "0x<ADDRESS>")
}

func (n AddressNormalizer) Description() string {
    return "normalize addresses"
}

// HashNormalizer replaces transaction hashes
type HashNormalizer struct{}

func (n HashNormalizer) Normalize(output string) string {
    re := regexp.MustCompile(`0x[a-fA-F0-9]{64}`)
    return re.ReplaceAllString(output, "0x<HASH>")
}

func (n HashNormalizer) Description() string {
    return "normalize hashes"
}

// RunCLITest executes a CLI test case
func RunCLITest(t *testing.T, test CLITest) {
    t.Helper()
    
    // Setup
    if test.Setup != nil {
        if err := test.Setup(t); err != nil {
            t.Fatalf("Setup failed: %v", err)
        }
    }
    
    // Cleanup
    if test.Teardown != nil {
        defer func() {
            if err := test.Teardown(t); err != nil {
                t.Errorf("Teardown failed: %v", err)
            }
        }()
    }
    
    // Build command
    args := append([]string{"treb"}, test.Command...)
    args = append(args, test.Args...)
    
    // Execute
    result := executeCLI(t, args, test.Env, test.WorkDir, test.Stdin)
    
    // Check exit code
    if result.ExitCode != test.ExitCode {
        t.Errorf("Exit code mismatch: want %d, got %d", test.ExitCode, result.ExitCode)
    }
    
    // Check stdout/stderr expectations
    if test.StdoutEmpty && result.Stdout != "" {
        t.Errorf("Expected empty stdout, got: %s", result.Stdout)
    }
    if test.StderrEmpty && result.Stderr != "" {
        t.Errorf("Expected empty stderr, got: %s", result.Stderr)
    }
    
    // Normalize output
    output := result.Stdout
    for _, normalizer := range test.Normalizers {
        output = normalizer.Normalize(output)
    }
    
    // Compare with golden file
    if test.GoldenFile != "" {
        compareWithGolden(t, output, test.GoldenFile)
    }
}

// CLIResult contains the results of CLI execution
type CLIResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

func executeCLI(t *testing.T, args []string, env map[string]string, workDir, stdin string) CLIResult {
    cmd := exec.Command(args[0], args[1:]...)
    
    // Set working directory
    if workDir != "" {
        cmd.Dir = workDir
    }
    
    // Set environment
    cmd.Env = os.Environ()
    for k, v := range env {
        cmd.Env = append(cmd.Env, k+"="+v)
    }
    
    // Set stdin
    if stdin != "" {
        cmd.Stdin = strings.NewReader(stdin)
    }
    
    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // Run command
    err := cmd.Run()
    
    // Get exit code
    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else {
            t.Fatalf("Command failed to run: %v", err)
        }
    }
    
    return CLIResult{
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
        ExitCode: exitCode,
    }
}

func compareWithGolden(t *testing.T, actual, goldenFile string) {
    t.Helper()
    
    goldenPath := filepath.Join("testdata/fixtures", goldenFile)
    
    // Update golden files if requested
    if os.Getenv("UPDATE_GOLDEN") == "true" {
        if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
            t.Fatalf("Failed to update golden file: %v", err)
        }
        t.Logf("Updated golden file: %s", goldenPath)
        return
    }
    
    // Read expected output
    expected, err := os.ReadFile(goldenPath)
    if err != nil {
        t.Fatalf("Failed to read golden file: %v", err)
    }
    
    // Compare
    if diff := cmp.Diff(string(expected), actual); diff != "" {
        t.Errorf("Output mismatch (-want +got):\n%s", diff)
    }
}
```

### 3. Command Test Suites

Create `test/integration/suites/list_test.go`:

```go
package suites

import (
    "testing"
    "github.com/trebuchet-org/treb-cli/test/integration/helpers"
)

func TestListCommand(t *testing.T) {
    tests := []helpers.CLITest{
        {
            Name:        "list_default",
            Description: "List command with no arguments",
            Command:     []string{"list"},
            GoldenFile:  "commands/list/default.golden",
            Normalizers: []helpers.OutputNormalizer{
                helpers.TimestampNormalizer{},
                helpers.AddressNormalizer{},
            },
        },
        {
            Name:        "list_with_namespace",
            Description: "List command with namespace filter",
            Command:     []string{"list"},
            Args:        []string{"--namespace", "production"},
            GoldenFile:  "commands/list/with_namespace.golden",
            Normalizers: []helpers.OutputNormalizer{
                helpers.TimestampNormalizer{},
                helpers.AddressNormalizer{},
            },
        },
        {
            Name:        "list_empty",
            Description: "List command with no deployments",
            Command:     []string{"list"},
            Setup: func(t *testing.T) error {
                // Create empty registry
                return createEmptyRegistry(t)
            },
            GoldenFile: "commands/list/empty.golden",
        },
        {
            Name:        "list_json_future",
            Description: "List command with JSON output (future)",
            Command:     []string{"list"},
            Args:        []string{"--json"},
            Env:         map[string]string{"TREB_JSON_ENABLED": "true"},
            GoldenFile:  "commands/list/json.golden",
            Normalizers: []helpers.OutputNormalizer{
                helpers.TimestampNormalizer{},
                helpers.AddressNormalizer{},
                helpers.HashNormalizer{},
            },
        },
    }
    
    for _, test := range tests {
        t.Run(test.Name, func(t *testing.T) {
            helpers.RunCLITest(t, test)
        })
    }
}
```

### 4. Workflow Tests

Create `test/integration/suites/deployment_workflow_test.go`:

```go
package suites

import (
    "testing"
    "github.com/trebuchet-org/treb-cli/test/integration/helpers"
)

func TestDeploymentWorkflow(t *testing.T) {
    workflowTest := helpers.WorkflowTest{
        Name:        "full_deployment_flow",
        Description: "Complete deployment workflow from init to verify",
        Steps: []helpers.WorkflowStep{
            {
                Name:    "init_project",
                Command: []string{"init", "test-project"},
                Validate: func(t *testing.T, result helpers.CLIResult) {
                    // Check project structure created
                },
            },
            {
                Name:    "generate_script",
                Command: []string{"generate", "deploy", "Counter"},
                Validate: func(t *testing.T, result helpers.CLIResult) {
                    // Check script generated
                },
            },
            {
                Name:    "run_deployment",
                Command: []string{"run", "script/deploy/Counter.s.sol"},
                Args:    []string{"--network", "local"},
                Normalizers: []helpers.OutputNormalizer{
                    helpers.AddressNormalizer{},
                    helpers.HashNormalizer{},
                },
            },
            {
                Name:    "list_deployments",
                Command: []string{"list"},
                Validate: func(t *testing.T, result helpers.CLIResult) {
                    // Check deployment appears in list
                },
            },
            {
                Name:    "verify_deployment",
                Command: []string{"verify", "Counter"},
                Args:    []string{"--network", "local"},
            },
        },
        GoldenFile: "workflows/full_deployment.golden",
    }
    
    helpers.RunWorkflowTest(t, workflowTest)
}
```

### 5. Snapshot Testing for Complex Outputs

Create `test/integration/helpers/snapshot.go`:

```go
package helpers

import (
    "encoding/json"
    "testing"
)

// SnapshotTest handles complex structured output testing
type SnapshotTest struct {
    Name       string
    Producer   func() (interface{}, error)
    Snapshot   string
    Normalizer func(interface{}) interface{}
}

func RunSnapshotTest(t *testing.T, test SnapshotTest) {
    t.Helper()
    
    // Produce output
    output, err := test.Producer()
    if err != nil {
        t.Fatalf("Failed to produce output: %v", err)
    }
    
    // Normalize if needed
    if test.Normalizer != nil {
        output = test.Normalizer(output)
    }
    
    // Compare with snapshot
    compareSnapshot(t, output, test.Snapshot)
}

func compareSnapshot(t *testing.T, actual interface{}, snapshotFile string) {
    // Implementation for structured data comparison
}
```

### 6. CI Integration

Create `.github/workflows/compatibility-tests.yml`:

```yaml
name: Compatibility Tests

on:
  pull_request:
  push:
    branches: [main]

jobs:
  compatibility:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Foundry
        uses: foundry-rs/foundry-toolchain@v1
      
      - name: Build CLI
        run: make build
      
      - name: Run compatibility tests
        run: |
          go test ./test/integration/... -v
      
      - name: Check for output changes
        run: |
          git diff --exit-code testdata/fixtures/
      
      - name: Upload test artifacts
        if: failure()
        uses: actions/upload-artifact@v3
        with:
          name: test-outputs
          path: |
            test-output/
            testdata/fixtures/
```

### 7. Makefile Targets

Add to `Makefile`:

```makefile
# Testing targets
.PHONY: test-integration
test-integration:
	go test ./test/integration/... -v

.PHONY: test-compatibility
test-compatibility:
	go test ./test/integration/... -tags=compatibility -v

.PHONY: update-golden
update-golden:
	UPDATE_GOLDEN=true go test ./test/integration/... -v

.PHONY: test-coverage
test-coverage:
	go test ./test/integration/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Generate initial golden files
.PHONY: generate-golden
generate-golden:
	@echo "Generating golden files for current CLI output..."
	@./scripts/generate-golden-files.sh
```

### 8. Initial Golden File Generation Script

Create `scripts/generate-golden-files.sh`:

```bash
#!/bin/bash
set -e

echo "Generating golden files for treb CLI..."

# Create directories
mkdir -p testdata/fixtures/commands/{list,show,run,init,generate,verify}

# Helper function to run command and save output
save_golden() {
    local cmd="$1"
    local output_file="$2"
    echo "Generating: $output_file"
    $cmd > "$output_file" 2>&1 || true
}

# List command variations
save_golden "treb list" "testdata/fixtures/commands/list/default.golden"
save_golden "treb list --namespace production" "testdata/fixtures/commands/list/with_namespace.golden"
save_golden "treb list --chain 1" "testdata/fixtures/commands/list/with_chain.golden"

# Show command variations
save_golden "treb show Counter" "testdata/fixtures/commands/show/default.golden"
save_golden "treb show Counter --network sepolia" "testdata/fixtures/commands/show/with_network.golden"

# Add more commands...

echo "Golden files generated successfully!"
```

## Testing Strategy

### 1. Coverage Requirements

- **Command Coverage**: Every CLI command must have tests
- **Flag Coverage**: All flag combinations tested
- **Error Cases**: Common error scenarios captured
- **Workflows**: End-to-end scenarios tested

### 2. Normalizer Strategy

Dynamic content that needs normalization:
- Timestamps (creation dates, relative times)
- Ethereum addresses (0x...)
- Transaction hashes
- Block numbers
- Gas values
- File paths (make relative)
- ANSI color codes (strip or normalize)

### 3. Golden File Management

```bash
# Initial generation
make generate-golden

# Update after intentional changes
UPDATE_GOLDEN=true make test-integration

# Review changes
git diff testdata/fixtures/

# Commit if correct
git add testdata/fixtures/
git commit -m "Update golden files for new feature X"
```

### 4. Parallel Testing

The framework supports parallel test execution:

```go
func TestListCommandParallel(t *testing.T) {
    t.Parallel()
    
    // Each test gets isolated workspace
    workspace := createTestWorkspace(t)
    defer cleanupWorkspace(t, workspace)
    
    // Run tests in isolation
}
```

## Success Criteria

Phase 0 is complete when:

1. ✅ All existing CLI commands have integration tests
2. ✅ Golden files capture current output exactly
3. ✅ CI runs compatibility tests on every PR
4. ✅ Normalizers handle all dynamic content
5. ✅ Workflow tests cover common user journeys
6. ✅ Test framework is documented and easy to use
7. ✅ Performance baseline established

## Next Steps

After Phase 0 completion:

1. Run `make generate-golden` to capture current behavior
2. Review and commit golden files
3. Enable compatibility tests in CI
4. Begin Phase 1 with confidence that changes won't break compatibility
5. Use `UPDATE_GOLDEN=true` only for intentional output changes
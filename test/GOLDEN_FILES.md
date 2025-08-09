# Golden File Testing

This document explains the golden file testing infrastructure for ensuring backwards compatibility during the architecture migration.

## Overview

Golden file tests capture the exact CLI output and compare it against expected output files. This ensures that any changes to the code don't accidentally break the user-facing interface.

## Structure

Golden files are stored in `test/fixture/testdata/golden/` with the following structure:

```
testdata/golden/
├── commands/
│   ├── list/
│   │   ├── default.golden
│   │   ├── empty.golden
│   │   └── with_filters.golden
│   ├── show/
│   │   └── not_found.golden
│   └── ...
└── workflows/
    └── multi_namespace_flow.golden
```

## How It Works

1. **Test Execution**: Tests run treb commands and capture the output
2. **Normalization**: Dynamic content (timestamps, addresses, hashes) is replaced with placeholders
3. **Comparison**: Normalized output is compared with golden files
4. **Updates**: When output changes are intentional, golden files can be updated

## Normalizers

The following normalizers ensure consistent output across test runs:

- **TimestampNormalizer**: Replaces timestamps with `<TIMESTAMP>`
- **AddressNormalizer**: Replaces Ethereum addresses with `0x<ADDRESS>`
- **HashNormalizer**: Replaces transaction hashes with `0x<HASH>`
- **BlockNumberNormalizer**: Replaces block numbers with `<BLOCK>`
- **GasNormalizer**: Replaces gas values with `<GAS>`
- **ColorNormalizer**: Removes ANSI color codes
- **PathNormalizer**: Makes paths relative to project root

## Running Golden Tests

### Run all golden tests
```bash
make golden-test
```

### Update golden files when output changes
```bash
make update-golden
```

### Generate initial golden files
```bash
make generate-golden
```

### Run specific golden test
```bash
cd test && go test -v -run "TestSimpleCommandsGolden/help"
```

### Update specific golden file
```bash
cd test && UPDATE_GOLDEN=true go test -v -run "TestSimpleCommandsGolden/help"
```

## Writing Golden Tests

Example test:

```go
func TestMyCommandGolden(t *testing.T) {
    tests := []struct {
        name       string
        args       []string
        goldenFile string
        expectErr  bool
    }{
        {
            name:       "basic",
            args:       []string{"mycommand"},
            goldenFile: "commands/mycommand/basic.golden",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := NewTrebContext(t)
            if tt.expectErr {
                ctx.trebGoldenWithError(tt.goldenFile, tt.args...)
            } else {
                ctx.trebGolden(tt.goldenFile, tt.args...)
            }
        })
    }
}
```

## CI Integration

Golden file tests run automatically in CI:

1. Tests execute and compare output
2. If differences are detected, CI fails
3. Developers must intentionally update golden files
4. Updated golden files are committed with the change

## Important Notes

- **Anvil Node**: Tests automatically start/stop an anvil node via `TestMain`
- **Working Directory**: Tests run in `test/fixture/` directory
- **Sequential Execution**: Tests run sequentially to avoid blockchain state conflicts
- **Determinism**: Use consistent test data and clean state between tests

## Troubleshooting

### Golden file not found
```bash
UPDATE_GOLDEN=true go test -v -run "TestName"
```

### Output differs from golden file
1. Check if the change is intentional
2. Review the diff carefully
3. Update if correct: `make update-golden`

### Dynamic content not normalized
Add a new normalizer in `golden_test.go` or update existing ones

## Best Practices

1. **Clean State**: Always clean test artifacts before tests
2. **Minimal Tests**: Test one thing per golden file
3. **Descriptive Names**: Use clear golden file names
4. **Review Changes**: Always review golden file diffs before committing
5. **Avoid Timestamps**: Use normalizers for any dynamic content
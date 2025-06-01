package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test contract generation
func TestGenerateCommands(t *testing.T) {
	// Cleanup before each test
	t.Run("setup", func(t *testing.T) {
		cleanupGeneratedFiles(t)
	})

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		checkFile   string
		contains    string
		errContains string
	}{
		{
			name:      "generate Counter with CREATE3",
			args:      []string{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "CREATE3"},
			checkFile: "script/deploy/DeployCounter.s.sol",
			contains:  ".create3(",
		},
		{
			name:      "generate TestCounter with CREATE2",
			args:      []string{"gen", "deploy", "src/TestCounter.sol:TestCounter", "--strategy", "CREATE2"},
			checkFile: "script/deploy/DeployTestCounter.s.sol",
			contains:  ".create2(",
		},
		{
			name:        "ambiguous contract name",
			args:        []string{"gen", "deploy", "Counter", "--strategy", "CREATE3", "--non-interactive"},
			wantErr:     true,
			errContains: "multiple contracts found matching",
		},
		{
			name:      "missing strategy uses default CREATE3",
			args:      []string{"gen", "deploy", "src/Counter.sol:Counter"},
			checkFile: "script/deploy/DeployCounter.s.sol",
			contains:  ".create3(",
		},
		{
			name:        "invalid strategy",
			args:        []string{"gen", "deploy", "src/Counter.sol:Counter", "--strategy", "INVALID"},
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "non-existent contract",
			args:        []string{"gen", "deploy", "src/NonExistent.sol:NonExistent", "--strategy", "CREATE3"},
			wantErr:     true,
			errContains: "contract 'src/NonExistent.sol:NonExistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean before each test
			cleanupGeneratedFiles(t)

			output, err := runTreb(t, tt.args...)

			if tt.wantErr {
				require.Error(t, err, "Command should have failed")
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains, "Error output should contain expected text")
				}
			} else {
				require.NoError(t, err, "Command failed: %s", output)
			}

			if tt.checkFile != "" {
				filePath := filepath.Join(fixtureDir, tt.checkFile)
				content, err := os.ReadFile(filePath)
				require.NoError(t, err, "Failed to read generated file")
				assert.Contains(t, string(content), tt.contains, "Generated file should contain expected text")
			}
		})
	}
}

// TestGenerateProxy was removed because 'gen proxy' command no longer exists
// Proxy deployments should be done manually using existing proxy contracts

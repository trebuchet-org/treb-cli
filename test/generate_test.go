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
			contains:  "DeployStrategy.CREATE3",
		},
		{
			name:      "generate TestCounter with CREATE2",
			args:      []string{"gen", "deploy", "src/TestCounter.sol:TestCounter", "--strategy", "CREATE2"},
			checkFile: "script/deploy/DeployTestCounter.s.sol",
			contains:  "DeployStrategy.CREATE2",
		},
		{
			name:        "ambiguous contract name",
			args:        []string{"gen", "deploy", "Counter", "--strategy", "CREATE3"},
			wantErr:     true,
			errContains: "multiple contracts found",
		},
		{
			name:        "missing strategy",
			args:        []string{"gen", "deploy", "src/Counter.sol:Counter"},
			wantErr:     true,
			errContains: "strategy",
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
			errContains: "not found",
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

// Test proxy generation
func TestGenerateProxy(t *testing.T) {
	// Non-interactive proxy generation is now implemented!
	
	cleanupGeneratedFiles(t)
	
	// Generate proxy script
	output, err := runTreb(t, "gen", "proxy",
		"src/UpgradeableCounter.sol:UpgradeableCounter",
		"--strategy", "CREATE3",
		"--proxy-contract", "lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol:TransparentUpgradeableProxy")
	
	if err != nil {
		t.Logf("Proxy generation failed with output: %s", output)
	}
	require.NoError(t, err)
	assert.Contains(t, output, "Generated")
	
	// Check file was created
	scriptPath := filepath.Join(fixtureDir, "script/deploy/DeployUpgradeableCounterProxy.s.sol")
	assert.FileExists(t, scriptPath)
	
	// Test missing proxy contract
	output, err = runTreb(t, "gen", "proxy",
		"src/UpgradeableCounter.sol:UpgradeableCounter",
		"--strategy", "CREATE3")
	assert.Error(t, err)
	assert.Contains(t, output, "proxy-contract")
}
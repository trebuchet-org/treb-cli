package integration_test

import (
	"testing"
	"os"
	"path/filepath"
)

func TestRunCommandGolden(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T)
		args       []string
		goldenFile string
		expectErr  bool
	}{
		{
			name: "deploy_simple",
			setup: func(t *testing.T) {
				cleanupGeneratedFiles(t)
				// Generate deployment script
				output, err := runTreb(t, "gen", "deploy", "src/Counter.sol:Counter")
				if err != nil {
					t.Fatalf("Failed to generate script: %v\nOutput:\n%s", err, output)
				}
			},
			args:       []string{"run", "script/deploy/DeployCounter.s.sol"},
			goldenFile: "commands/run/deploy_simple.golden",
		},
		{
			name: "deploy_with_label",
			setup: func(t *testing.T) {
				// Ensure script exists
				if _, err := os.Stat(filepath.Join(fixtureDir, "script/deploy/DeployCounter.s.sol")); os.IsNotExist(err) {
					output, err := runTreb(t, "gen", "deploy", "src/Counter.sol:Counter")
					if err != nil {
						t.Fatalf("Failed to generate script: %v\nOutput:\n%s", err, output)
					}
				}
				// Deploy first without label
				ctx := NewTrebContext(t)
				output, err := ctx.treb("run", "script/deploy/DeployCounter.s.sol")
				if err != nil {
					t.Fatalf("Failed initial deployment: %v\nOutput:\n%s", err, output)
				}
			},
			args: []string{"run", "script/deploy/DeployCounter.s.sol", "--env", "label=v2"},
			goldenFile: "commands/run/deploy_with_label.golden",
		},
		{
			name: "deploy_dry_run",
			setup: func(t *testing.T) {
				// Ensure script exists
				if _, err := os.Stat(filepath.Join(fixtureDir, "script/deploy/DeployCounter.s.sol")); os.IsNotExist(err) {
					output, err := runTreb(t, "gen", "deploy", "src/Counter.sol:Counter")
					if err != nil {
						t.Fatalf("Failed to generate script: %v\nOutput:\n%s", err, output)
					}
				}
			},
			args: []string{"run", "script/deploy/DeployCounter.s.sol", "--dry-run"},
			goldenFile: "commands/run/deploy_dry_run.golden",
		},
		// Skip this test for now - requires proper sender configuration
		// {
		// 	name: "deploy_with_params",
		// 	setup: func(t *testing.T) {
		// 		// Create a custom script that uses parameters
		// 		createParameterizedScript(t)
		// 	},
		// 	args: []string{"run", "script/deploy/ParameterizedDeploy.s.sol", 
		// 		"--env", "initialValue=42",
		// 		"--env", "owner=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		// 		"--env", "SENDER=anvil",
		// 	},
		// 	goldenFile: "commands/run/deploy_with_params.golden",
		// },
		{
			name: "deploy_already_exists",
			setup: func(t *testing.T) {
				// Generate script first
				output, err := runTreb(t, "gen", "deploy", "src/Counter.sol:Counter")
				if err != nil {
					t.Fatalf("Failed to generate script: %v\nOutput:\n%s", err, output)
				}
				// Deploy once first
				ctx := NewTrebContext(t)
				output, err = ctx.treb("run", "script/deploy/DeployCounter.s.sol")
				if err != nil {
					t.Fatalf("Failed to deploy first time: %v\nOutput:\n%s", err, output)
				}
			},
			args:       []string{"run", "script/deploy/DeployCounter.s.sol"},
			goldenFile: "commands/run/deploy_already_exists.golden",
			expectErr:  false, // CreateX returns existing address, so no error
		},
		{
			name: "script_not_found",
			args:       []string{"run", "script/deploy/NonExistent.s.sol"},
			goldenFile: "commands/run/script_not_found.golden",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		IsolatedTest(t, tt.name, func(t *testing.T, ctx *TrebContext) {
			if tt.setup != nil {
				tt.setup(t)
			}

			if tt.expectErr {
				ctx.trebGoldenWithError(tt.goldenFile, tt.args...)
			} else {
				ctx.trebGolden(tt.goldenFile, tt.args...)
			}
		})
	}
}

// createParameterizedScript creates a test script that accepts parameters
func createParameterizedScript(t *testing.T) {
	t.Helper()
	
	scriptContent := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {ConfigurableTrebScript} from "treb-sol/src/ConfigurableTrebScript.sol";
import {Counter} from "../../src/Counter.sol";

contract ParameterizedDeploy is ConfigurableTrebScript {
    /**
     * @custom:env {uint256} initialValue Initial counter value
     * @custom:env {address} owner Contract owner address
     */
    function run() public override {
        uint256 initialValue = vm.envUint("initialValue");
        address owner = vm.envAddress("owner");
        
        startBroadcast();
        Counter counter = new Counter();
        counter.setNumber(initialValue);
        vm.stopBroadcast();
        
        // Log deployment info
        log.info("Counter deployed at", vm.toString(address(counter)));
        log.info("Initial value", vm.toString(initialValue));
        log.info("Owner", vm.toString(owner));
    }
}
`
	
	scriptPath := filepath.Join(fixtureDir, "script/deploy/ParameterizedDeploy.s.sol")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create parameterized script: %v", err)
	}
}
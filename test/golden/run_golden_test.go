package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCommandGolden(t *testing.T) {
	tests := []GoldenTest{
		{
			Name: "deploy_simple",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			GoldenFile: "commands/run/deploy_simple.golden",
		},
		{
			Name: "deploy_with_label",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--env", "label=v2"},
			},
			GoldenFile: "commands/run/deploy_with_label.golden",
		},
		{
			Name: "deploy_dry_run",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol", "--dry-run"},
			},
			GoldenFile: "commands/run/deploy_dry_run.golden",
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
			Name: "deploy_already_exists",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			GoldenFile: "commands/run/deploy_already_exists.golden",
			ExpectErr:  false, // CreateX returns existing address, so no error
		},
		{
			Name: "script_not_found",
			TestCmds: [][]string{
				{"run", "script/deploy/NonExistent.s.sol"},
			},
			GoldenFile: "commands/run/script_not_found.golden",
			ExpectErr:  true,
		},
	}

	RunGoldenTests(t, tests)
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

	scriptPath := filepath.Join(helpers.GetFixtureDir(), "script/deploy/ParameterizedDeploy.s.sol")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create parameterized script: %v", err)
	}
}

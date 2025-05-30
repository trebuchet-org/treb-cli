// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";

import {Counter} from "../src/Counter.sol";
import {SampleToken} from "../src/SampleToken.sol";
import {console} from "forge-std/console.sol";

contract DeployWithTrebScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        // Get the sender
        Senders.Sender storage anvil = sender("anvil");

        // Deploy a Counter using CREATE3 with deterministic address
        address counter = anvil.create3("src/Counter.sol:Counter").setLabel("V100").deploy();
        
        // Initialize the counter
        Counter(counter).setNumber(100);
        
        // Deploy a token with constructor args
        address token = anvil.create3("src/SampleToken.sol:SampleToken").setLabel("v100").deploy(
            abi.encode("Test Token", "TEST", 1000000 * 10**18)
        );
        
        // Log the deployments
        console.log("Counter deployed at:", counter);
        console.log("Token deployed at:", token);
        
        // Check the counter value
        uint256 value = Counter(counter).number();
        console.log("Counter value:", value);
        
        // Read from registry (if deployments exist)
        address existingCounter = tryGetDeployment("Counter");
        if (existingCounter != address(0)) {
            console.log("Found existing Counter at:", existingCounter);
        }
    }
    
    // Helper to safely get deployment
    function tryGetDeployment(string memory name) internal view returns (address) {
        try this.getDeployment(name) returns (address addr) {
            return addr;
        } catch {
            return address(0);
        }
    }
}
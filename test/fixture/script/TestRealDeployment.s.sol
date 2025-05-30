// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestRealDeploymentScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public {
        console.log("Starting real deployment test...");
        
        // Get the test sender
        Senders.Sender storage testSender = sender("test-sender");
        console.log("Got test sender");
        
        // Deploy a contract - this should emit DeployingContract and ContractDeployed events
        address deployed = testSender.create3("src/Counter.sol:Counter").deploy();
        console.log("Deployed Counter at:", deployed);
        
        console.log("Deployment test completed");
    }
}
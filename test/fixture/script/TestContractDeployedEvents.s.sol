// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestContractDeployedEventsScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public {
        console.log("=== Testing ContractDeployed Events ===");
        
        // Get the test sender
        Senders.Sender storage testSender = sender("test-sender");
        console.log("Using sender:", testSender.account);
        console.log("Sender name:", testSender.name);
        
        console.log("\n--- Deploying Counter ---");
        // Deploy Counter - should emit DeployingContract and ContractDeployed events
        address counterAddr = testSender.create3("src/Counter.sol:Counter").deploy();
        console.log("Counter deployed at:", counterAddr);
        
        console.log("\n--- Deploying TestCounter ---");
        // Deploy TestCounter - should emit more events
        address testCounterAddr = testSender.create3("src/TestCounter.sol:TestCounter").deploy();
        console.log("TestCounter deployed at:", testCounterAddr);
        
        console.log("\n--- Deploying StringUtils ---");
        // Deploy a different contract to see more events  
        address stringUtilsAddr = testSender.create3("src/StringUtils.sol:StringUtils").deploy();
        console.log("StringUtils deployed at:", stringUtilsAddr);
        
        console.log("\n=== All deployments completed ===");
        console.log("Summary:");
        console.log("- Counter:", counterAddr);
        console.log("- TestCounter:", testCounterAddr);
        console.log("- StringUtils:", stringUtilsAddr);
    }
}
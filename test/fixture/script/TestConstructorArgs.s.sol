// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestConstructorArgsScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public {
        console.log("=== Testing Constructor Args ===");
        
        // Get the test sender
        Senders.Sender storage testSender = sender("test-sender");
        console.log("Using sender:", testSender.account);
        
        console.log("\n--- Deploying SampleToken with constructor args ---");
        // Deploy SampleToken with constructor arguments: name, symbol, totalSupply
        bytes memory constructorArgs = abi.encode("Test Token", "TEST", 1000000 * 10**18);
        address tokenAddr = testSender.create3("src/SampleToken.sol:SampleToken").deploy(constructorArgs);
        console.log("SampleToken deployed at:", tokenAddr);
        
        console.log("\n--- Deploying SimpleTokenV07 with different args ---");
        // Deploy another token with different args
        bytes memory constructorArgs2 = abi.encode("Another Token", "ANOTHER", 5000000 * 10**18);
        address token2Addr = testSender.create3("src/SimpleTokenV07.sol:SimpleTokenV07").deploy(constructorArgs2);
        console.log("SimpleTokenV07 deployed at:", token2Addr);
        
        console.log("\n=== Constructor args test completed ===");
    }
}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";
import {Counter} from "../src/Counter.sol";

contract TestEventParsingScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    // Emit some test events directly
    event TestEvent(string message, uint256 value);
    
    function run() public {
        console.log("TestEventParsingScript starting...");
        
        // Get the local sender
        Senders.Sender storage localSender = sender("local");
        
        // Emit a test event
        emit TestEvent("Starting deployment process", 42);
        
        // Deploy will emit DeployingContract and ContractDeployed events
        address counterAddr = localSender.create3("src/Counter.sol:Counter").deploy();
        console.log("Counter deployed at:", counterAddr);
        
        // Set a value on the counter
        Counter counter = Counter(counterAddr);
        counter.setNumber(100);
        
        // Deploy another contract with label
        address counter2 = localSender.create3("src/Counter.sol:Counter").setLabel("counter-v2").deploy();
        console.log("Counter v2 deployed at:", counter2);
        
        emit TestEvent("Deployment completed", 2);
        
        console.log("TestEventParsingScript completed");
    }
}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";
import {Counter} from "../src/Counter.sol";

contract TestSenderDeployScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        console.log("TestSenderDeployScript starting...");
        
        // Use the local sender to deploy a Counter
        Senders.Sender storage localSender = sender("local");
        address counterAddr = localSender.create3("src/Counter.sol:Counter").deploy();
        console.log("Counter deployed at:", counterAddr);
        
        // Verify it was deployed
        Counter counter = Counter(counterAddr);
        counter.setNumber(42);
        console.log("Counter number set to:", counter.number());
        
        console.log("TestSenderDeployScript completed");
    }
}
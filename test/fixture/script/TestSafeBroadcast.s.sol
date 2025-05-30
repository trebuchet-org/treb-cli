// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestSafeBroadcastScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        console.log("TestSafeBroadcast - before getting sender");
        
        // Try to use the safe13 sender
        Senders.Sender storage safeSender = sender("safe13");
        console.log("Got safe sender");
        
        // Deploy something with the safe sender
        address deployed = safeSender.create3("src/Counter.sol:Counter").deploy();
        console.log("Deployed at:", deployed);
    }
}
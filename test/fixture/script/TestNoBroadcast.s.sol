// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestNoBroadcastScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public {
        console.log("TestNoBroadcast starting...");
        
        // Use local sender without broadcast modifier
        Senders.Sender storage localSender = sender("local");
        console.log("Got local sender");
        
        // Try to deploy
        address deployed = localSender.create3("src/Counter.sol:Counter").deploy();
        console.log("Deployed at:", deployed);
        
        console.log("Success!");
    }
}
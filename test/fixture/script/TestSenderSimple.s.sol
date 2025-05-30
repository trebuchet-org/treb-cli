// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestSenderSimpleScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public {
        console.log("TestSenderSimpleScript starting...");
        
        // List all configured senders
        console.log("Checking configured senders...");
        
        // Get the anvil sender
        Senders.Sender storage anvilSender = sender("anvil");
        console.log("Anvil sender account:", anvilSender.account);
        console.log("Anvil sender type:", uint256(uint64(anvilSender.senderType)));
        
        // Get the local sender  
        Senders.Sender storage localSender = sender("local");
        console.log("Local sender account:", localSender.account);
        
        // Get the safe sender
        Senders.Sender storage safeSender = sender("safe");
        console.log("Safe sender account:", safeSender.account);
        
        console.log("TestSenderSimpleScript completed");
    }
}
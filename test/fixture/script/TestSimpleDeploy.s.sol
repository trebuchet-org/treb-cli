// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestSimpleDeployScript is TrebScript {
    using Deployer for Senders.Sender;

    function run() public {
        console.log("TestSimpleDeployScript starting...");
        
        // Get the local sender using the dispatcher method
        Senders.Sender storage localSender = sender("local");
        console.log("Got local sender, account:", localSender.account);
        
        // Check if CreateX is available
        address createX = address(0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed);
        uint256 codeSize;
        assembly {
            codeSize := extcodesize(createX)
        }
        console.log("CreateX code size:", codeSize);
        
        // Try to predict an address first
        bytes32 salt = keccak256("test-simple");
        address predicted = localSender.predictCreate3(salt);
        console.log("Predicted address:", predicted);
        
        // Now try a simple deployment using raw bytecode
        bytes memory bytecode = hex"6080604052348015600e575f5ffd5b50603e80601b5f395ff3fe6080604052348015600e575f5ffd5b50005fea";
        address deployed = localSender.deployCreate3(salt, bytecode, "");
        console.log("Deployed address:", deployed);
        
        console.log("TestSimpleDeployScript completed");
    }
}
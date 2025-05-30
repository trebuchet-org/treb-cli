// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {console} from "forge-std/console.sol";

contract TestSimpleDeployScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

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
        
        bytes memory bytecode = hex"6080604052348015600e575f5ffd5b50603e80601b5f395ff3fe6080604052348015600e575f5ffd5b50005fea";
        // Try to predict an address first
        string memory entropy = "test-simple";
        address predicted = localSender.create3(entropy, bytecode).predict();
        console.log("Predicted address:", predicted);
        
        // Now try a simple deployment using raw bytecode
        address deployed = localSender.create3(entropy, bytecode).deploy();
        console.log("Deployed address:", deployed);
        
        console.log("TestSimpleDeployScript completed");
    }
}
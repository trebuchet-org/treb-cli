// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {SenderTypes} from "treb-sol/internal/types.sol";

contract TestSenderInitScript is Script {
    using Senders for Senders.Registry;
    using Senders for Senders.Sender;
    
    function run() public {
        console.log("Testing sender initialization...");
        
        // Decode configs
        bytes memory rawConfigs = vm.envBytes("SENDER_CONFIGS");
        Senders.SenderInitConfig[] memory configs = abi.decode(rawConfigs, (Senders.SenderInitConfig[]));
        
        console.log("Number of configs:", configs.length);
        
        // Log each config
        for (uint i = 0; i < configs.length; i++) {
            console.log("Config", i);
            console.log("  Name:", configs[i].name);
            console.log("  Account:", configs[i].account);
            console.logBytes8(configs[i].senderType);
            console.log("  Config length:", configs[i].config.length);
            
            // Check sender type
            if (configs[i].senderType == SenderTypes.InMemory) {
                console.log("  Type: InMemory");
                uint256 pk = abi.decode(configs[i].config, (uint256));
                console.log("  Private key configured");
            } else if (configs[i].senderType == SenderTypes.GnosisSafe) {
                console.log("  Type: GnosisSafe");
                string memory proposer = abi.decode(configs[i].config, (string));
                console.log("  Proposer:", proposer);
            }
        }
        
        // Initialize registry
        console.log("\nInitializing registry...");
        Senders.Registry storage registry = Senders.registry();
        registry.initialize(configs);
        
        // Test getting senders
        console.log("\nTesting sender retrieval...");
        
        try this.getSender("local") returns (address account) {
            console.log("Local sender account:", account);
        } catch Error(string memory reason) {
            console.log("Failed to get local sender:", reason);
        } catch {
            console.log("Failed to get local sender: unknown error");
        }
    }
    
    function getSender(string memory name) external returns (address) {
        Senders.Sender storage s = Senders.get(name);
        return s.account;
    }
}
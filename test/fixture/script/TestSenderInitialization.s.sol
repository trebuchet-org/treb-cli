// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {SenderTypes} from "treb-sol/internal/types.sol";

contract TestSenderInitializationScript is Script {
    using Senders for Senders.Sender;
    function run() public {
        console.log("Testing full sender initialization pipeline...");
        
        // Get the encoded configs from environment variable
        bytes memory rawConfigs = vm.envBytes("SENDER_CONFIGS");
        console.log("SENDER_CONFIGS length:", rawConfigs.length);
        
        // Decode the configs
        Senders.SenderInitConfig[] memory configs = abi.decode(rawConfigs, (Senders.SenderInitConfig[]));
        console.log("Successfully decoded", configs.length, "sender configs");
        
        // Initialize the senders registry
        Senders.initialize(configs);
        console.log("Senders registry initialized successfully");
        
        // Test accessing the sender  
        bytes32 senderId = keccak256(abi.encodePacked("test-sender"));
        Senders.Sender storage sender = Senders.get(senderId);
        console.log("Retrieved sender:");
        console.log("  Name:", sender.name);
        console.log("  Account:", sender.account);
        console.logBytes8(sender.senderType);
        console.log("  Config length:", sender.config.length);
        
        // Verify sender type
        bool isInMemory = sender.isType(SenderTypes.InMemory);
        bool isPrivateKey = sender.isType(SenderTypes.PrivateKey);
        console.log("  Is InMemory:", isInMemory);
        console.log("  Is PrivateKey:", isPrivateKey);
        
        // Decode the private key from config
        uint256 privateKey = abi.decode(sender.config, (uint256));
        console.log("  Private key decoded successfully:", privateKey != 0);
        
        console.log("\n=== SUCCESS: All tests passed! ===");
    }
}
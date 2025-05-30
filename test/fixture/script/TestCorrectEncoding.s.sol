// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {SenderTypes} from "treb-sol/internal/types.sol";

contract TestCorrectEncodingScript is Script {
    function run() public {
        console.log("Creating correct SENDER_CONFIGS with proper address...");
        
        // Create config with the actual address derived from the private key
        // Private key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
        // Derived address: 0xf39Fd6e51aad88F6F4ce6aB8827279cFFb92266
        
        Senders.SenderInitConfig[] memory configs = new Senders.SenderInitConfig[](1);
        configs[0] = Senders.SenderInitConfig({
            name: "test-sender",
            account: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266,  // Correct address for the private key
            senderType: SenderTypes.InMemory,
            config: abi.encode(uint256(0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80))
        });
        
        // Encode the array
        bytes memory encoded = abi.encode(configs);
        
        console.log("=== CORRECT ENCODING ===");
        console.log("Encoded length:", encoded.length);
        
        // Print as hex string for environment variable
        string memory hexString = string(abi.encodePacked("0x", _bytesToHex(encoded)));
        console.log("SENDER_CONFIGS value:");
        console.log(hexString);
        
        // Test decoding
        Senders.SenderInitConfig[] memory decoded = abi.decode(encoded, (Senders.SenderInitConfig[]));
        console.log("\n=== VERIFICATION ===");
        console.log("Name:", decoded[0].name);
        console.log("Account:", decoded[0].account);
        console.logBytes8(decoded[0].senderType);
        console.log("Config length:", decoded[0].config.length);
        
        uint256 privateKey = abi.decode(decoded[0].config, (uint256));
        console.log("Private key correct:", privateKey == 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80);
        
        // Print values for comparison
        console.log("\n=== VALUES FOR GO COMPARISON ===");
        console.log("Expected name: test-sender");
        console.logAddress(decoded[0].account);
        console.log("SenderType (InMemory):");
        console.logBytes8(SenderTypes.InMemory);
    }
    
    function _bytesToHex(bytes memory data) internal pure returns (bytes memory) {
        bytes memory alphabet = "0123456789abcdef";
        bytes memory str = new bytes(data.length * 2);
        for (uint i = 0; i < data.length; i++) {
            str[i*2] = alphabet[uint(uint8(data[i] >> 4))];
            str[1+i*2] = alphabet[uint(uint8(data[i] & 0x0f))];
        }
        return str;
    }
}
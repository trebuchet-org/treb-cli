// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {SenderTypes} from "treb-sol/internal/types.sol";

contract TestSenderEncodingScript is Script {
    function run() public {
        console.log("Testing manual SenderInitConfig encoding...");
        
        // Create a simple private key sender config manually
        Senders.SenderInitConfig[] memory configs = new Senders.SenderInitConfig[](1);
        configs[0] = Senders.SenderInitConfig({
            name: "test-sender",
            account: address(0x1234567890AbcdEF1234567890aBcdef12345678),
            senderType: SenderTypes.InMemory,
            config: abi.encode(uint256(0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80)) // sample private key
        });
        
        // Encode the array
        bytes memory encoded = abi.encode(configs);
        
        console.log("=== MANUAL ENCODING ===");
        console.log("Encoded length:", encoded.length);
        console.logBytes(encoded);
        
        // Print as hex string for Go code
        console.log("Hex string for SENDER_CONFIGS:");
        string memory hexString = string(abi.encodePacked("0x", _bytesToHex(encoded)));
        console.log(hexString);
        
        // Test decoding
        console.log("\n=== TESTING DECODING ===");
        Senders.SenderInitConfig[] memory decoded = abi.decode(encoded, (Senders.SenderInitConfig[]));
        console.log("Decoded successfully!");
        console.log("Name:", decoded[0].name);
        console.log("Account:", decoded[0].account);
        console.logBytes8(decoded[0].senderType);
        console.log("Config length:", decoded[0].config.length);
        
        // Also test decoding the raw private key
        uint256 privateKey = abi.decode(decoded[0].config, (uint256));
        console.log("Decoded private key matches:", privateKey == 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80);
        
        // Test with environment variable if provided
        try vm.envBytes("SENDER_CONFIGS") returns (bytes memory envConfigs) {
            console.log("\n=== TESTING ENV CONFIGS ===");
            console.log("SENDER_CONFIGS length:", envConfigs.length);
            
            // Try direct decoding
            try this.externalDecode(envConfigs) returns (Senders.SenderInitConfig[] memory envDecoded) {
                console.log("ENV: Decoded successfully!");
                console.log("ENV: Name:", envDecoded[0].name);
                console.log("ENV: Account:", envDecoded[0].account);
                console.logBytes8(envDecoded[0].senderType);
            } catch Error(string memory reason) {
                console.log("ENV: Failed to decode:", reason);
            } catch {
                console.log("ENV: Failed to decode: unknown error");
            }
        } catch {
            console.log("\n=== NO ENV CONFIGS PROVIDED ===");
        }
    }
    
    function externalDecode(bytes memory data) external pure returns (Senders.SenderInitConfig[] memory) {
        return abi.decode(data, (Senders.SenderInitConfig[]));
    }
    
    function _bytesToHex(bytes memory data) internal pure returns (bytes memory) {
        bytes memory alphabet = "0123456789abcdef";
        bytes memory str = new bytes(2 + data.length * 2);
        for (uint i = 0; i < data.length; i++) {
            str[i*2] = alphabet[uint(uint8(data[i] >> 4))];
            str[1+i*2] = alphabet[uint(uint8(data[i] & 0x0f))];
        }
        return str;
    }
}
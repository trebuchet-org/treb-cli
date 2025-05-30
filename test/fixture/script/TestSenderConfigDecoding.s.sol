// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";

contract TestSenderConfigDecodingScript is Script {
    function run() public {
        console.log("Testing SENDER_CONFIGS decoding...");
        
        // Get the raw configs
        bytes memory rawConfigs = vm.envBytes("SENDER_CONFIGS");
        console.log("SENDER_CONFIGS length:", rawConfigs.length);
        
        // Log first 64 bytes to understand structure
        if (rawConfigs.length >= 64) {
            bytes32 first32 = bytes32(uint256(bytes32(rawConfigs)) >> (256 - 32 * 8));
            bytes32 second32;
            assembly {
                second32 := mload(add(rawConfigs, 64))
            }
            console.logBytes32(first32);
            console.logBytes32(second32);
        }
        
        // Try to decode as array
        try this.decodeAsArray(rawConfigs) {
            console.log("Successfully decoded as array");
        } catch Error(string memory reason) {
            console.log("Failed to decode as array:", reason);
        } catch {
            console.log("Failed to decode as array: unknown error");
        }
        
        // Try to decode with offset
        try this.decodeWithOffset(rawConfigs) {
            console.log("Successfully decoded with offset");
        } catch Error(string memory reason) {
            console.log("Failed to decode with offset:", reason);
        } catch {
            console.log("Failed to decode with offset: unknown error");
        }
    }
    
    function decodeAsArray(bytes memory data) external pure returns (Senders.SenderInitConfig[] memory) {
        return abi.decode(data, (Senders.SenderInitConfig[]));
    }
    
    function decodeWithOffset(bytes memory data) external pure returns (Senders.SenderInitConfig[] memory) {
        // Skip first 32 bytes (offset)
        bytes memory dataWithoutOffset = new bytes(data.length - 32);
        for (uint i = 32; i < data.length; i++) {
            dataWithoutOffset[i - 32] = data[i];
        }
        return abi.decode(dataWithoutOffset, (Senders.SenderInitConfig[]));
    }
}
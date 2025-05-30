// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";

contract TestDispatcherInit2Script is Script {
    error InvalidSenderConfigs();
    
    function run() public {
        console.log("Testing raw abi.decode of SENDER_CONFIGS...");
        
        // Get the raw configs
        bytes memory rawConfigs;
        try vm.envBytes("SENDER_CONFIGS") returns (bytes memory configs) {
            rawConfigs = configs;
            console.log("SENDER_CONFIGS found, length:", rawConfigs.length);
        } catch {
            console.log("SENDER_CONFIGS not found!");
            return;
        }
        
        // Try to decode it exactly as Dispatcher does
        console.log("Attempting abi.decode...");
        try this.tryDecode(rawConfigs) returns (uint256 numConfigs) {
            console.log("Successfully decoded", numConfigs, "configs");
        } catch Error(string memory reason) {
            console.log("abi.decode failed with reason:", reason);
        } catch (bytes memory data) {
            console.log("abi.decode failed with data:");
            console.logBytes(data);
            
            // Try to decode the error selector
            if (data.length >= 4) {
                bytes4 selector;
                assembly {
                    selector := mload(add(data, 0x20))
                }
                console.log("Error selector:");
                console.logBytes4(selector);
            }
        }
    }
    
    function tryDecode(bytes memory rawConfigs) external pure returns (uint256) {
        Senders.SenderInitConfig[] memory configs = abi.decode(rawConfigs, (Senders.SenderInitConfig[]));
        return configs.length;
    }
}
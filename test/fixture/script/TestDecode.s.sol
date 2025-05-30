// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {console} from "forge-std/console.sol";
import {Vm} from "forge-std/Vm.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";

contract TestDecodeScript {
    Vm constant vm = Vm(address(bytes20(uint160(uint256(keccak256("hevm cheat code"))))));

    function run() public {
        console.log("Testing SENDER_CONFIGS decode");
        
        try vm.envBytes("SENDER_CONFIGS") returns (bytes memory rawConfigs) {
            console.log("Got raw configs, length:", rawConfigs.length);
            
            // Try to decode
            try this.tryDecode(rawConfigs) returns (Senders.SenderInitConfig[] memory configs) {
                console.log("Decode successful! Config count:", configs.length);
                for (uint256 i = 0; i < configs.length; i++) {
                    console.log("Config", i, ":");
                    console.log("  Name:", configs[i].name);
                    console.log("  Account:", configs[i].account);
                    console.logBytes8(configs[i].senderType);
                }
            } catch (bytes memory reason) {
                console.log("Decode failed with reason:");
                console.logBytes(reason);
            }
        } catch {
            console.log("Failed to get SENDER_CONFIGS env var");
        }
    }
    
    function tryDecode(bytes memory data) external pure returns (Senders.SenderInitConfig[] memory) {
        return abi.decode(data, (Senders.SenderInitConfig[]));
    }
}
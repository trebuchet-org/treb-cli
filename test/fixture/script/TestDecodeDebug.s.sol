// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {console} from "forge-std/console.sol";
import {Vm} from "forge-std/Vm.sol";

contract TestDecodeDebugScript {
    Vm constant vm = Vm(address(bytes20(uint160(uint256(keccak256("hevm cheat code"))))));

    struct SimpleStruct {
        string name;
        address account;
    }

    function run() public {
        console.log("Testing simple array decode");
        
        // First test: decode a simple uint256[]
        bytes memory simpleArray = hex"0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002";
        uint256[] memory nums = abi.decode(simpleArray, (uint256[]));
        console.log("Simple array decoded successfully, length:", nums.length);
        
        // Manually encode one struct
        bytes memory oneStruct = abi.encode(SimpleStruct("test", address(0x123)));
        console.log("One struct encoded, length:", oneStruct.length);
        console.logBytes(oneStruct);
        
        // Try to decode it back
        SimpleStruct memory decoded = abi.decode(oneStruct, (SimpleStruct));
        console.log("Decoded name:", decoded.name);
        console.log("Decoded account:", decoded.account);
        
        // Now try array of structs
        SimpleStruct[] memory structs = new SimpleStruct[](1);
        structs[0] = SimpleStruct("test", address(0x123));
        bytes memory structArray = abi.encode(structs);
        console.log("Struct array encoded, length:", structArray.length);
        
        // The SENDER_CONFIGS should be an array of structs with:
        // string name, address account, bytes8 senderType, bytes config
        // Let's check if the hex looks right
        try vm.envBytes("SENDER_CONFIGS") returns (bytes memory rawConfigs) {
            console.log("Raw configs first 32 bytes:");
            bytes32 first32;
            assembly {
                first32 := mload(add(rawConfigs, 0x20))
            }
            console.logBytes32(first32);
            
            // Should be 0x20 (32) for the offset to the array
            uint256 offset = abi.decode(rawConfigs, (uint256));
            console.log("First uint256 (offset):", offset);
        } catch {
            console.log("No SENDER_CONFIGS");
        }
    }
}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

contract TestArtifactPathScript is Script {
    function run() public {
        console.log("Testing artifact paths...");
        
        // Try different formats for PrivateKeySender
        bytes memory args = abi.encode(address(0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266));
        
        string[7] memory formats = [
            "PrivateKeySender",
            "PrivateKeySender.sol",
            "PrivateKeySender.sol:PrivateKeySender",
            "senders/PrivateKeySender.sol:PrivateKeySender",
            "internal/senders/PrivateKeySender.sol:PrivateKeySender",
            "src/internal/senders/PrivateKeySender.sol:PrivateKeySender",
            "lib/treb-sol/src/internal/senders/PrivateKeySender.sol:PrivateKeySender"
        ];
        
        for (uint i = 0; i < formats.length; i++) {
            console.log("Trying format:", formats[i]);
            try vm.deployCode(formats[i], args) returns (address sender) {
                console.log("SUCCESS! Deployed at:", sender);
                return;
            } catch Error(string memory reason) {
                console.log("Failed with:", reason);
            } catch {
                console.log("Failed with low-level error");
            }
        }
        
        console.log("All formats failed!");
    }
}
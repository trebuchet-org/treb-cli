// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

contract TestDispatcherScript is Script {
    function run() public {
        console.log("TestDispatcherScript starting...");
        
        // Check environment variables
        console.log("Checking NETWORK env var...");
        try vm.envString("NETWORK") returns (string memory network) {
            console.log("NETWORK:", network);
        } catch {
            console.log("NETWORK env var not found");
        }
        
        // Check SENDER_CONFIGS
        console.log("Checking SENDER_CONFIGS env var...");
        try vm.envBytes("SENDER_CONFIGS") returns (bytes memory configs) {
            console.log("SENDER_CONFIGS found, length:", configs.length);
            // Try to decode first few bytes
            if (configs.length > 32) {
                console.logBytes32(bytes32(configs));
            }
        } catch {
            console.log("SENDER_CONFIGS env var not found");
        }
        
        // Try to deploy a simple artifact
        console.log("Testing vm.deployCode...");
        try vm.deployCode("Counter.sol:Counter") returns (address counter) {
            console.log("Counter deployed at:", counter);
        } catch {
            console.log("Failed to deploy Counter");
        }
        
        // Try with sender artifact
        console.log("Testing PrivateKeySender deployment...");
        bytes memory args = abi.encode(address(0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266));
        try vm.deployCode("PrivateKeySender", args) returns (address sender) {
            console.log("PrivateKeySender deployed at:", sender);
        } catch {
            console.log("Failed to deploy PrivateKeySender");
        }
        
        console.log("TestDispatcherScript completed");
    }
}
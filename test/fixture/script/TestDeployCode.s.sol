// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

contract TestDeployCodeScript is Script {
    function run() public {
        console.log("TestDeployCodeScript starting...");
        
        // Try with exact artifact path matching output structure
        console.log("Testing deployment with exact output path...");
        bytes memory args = abi.encode(address(0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266));
        try vm.deployCode("PrivateKeySender.sol/PrivateKeySender.json", args) returns (address sender) {
            console.log("SUCCESS: PrivateKeySender deployed at:", sender);
        } catch Error(string memory reason) {
            console.log("FAILED with reason:", reason);
        } catch {
            console.log("FAILED with low-level error");
        }
        
        // Try with artifact name only
        console.log("Testing deployment with artifact name only...");
        try vm.deployCode("PrivateKeySender", args) returns (address sender) {
            console.log("SUCCESS: PrivateKeySender deployed at:", sender);
        } catch Error(string memory reason) {
            console.log("FAILED with reason:", reason);
        } catch {
            console.log("FAILED with low-level error");
        }
        
        // Try with relative path
        console.log("Testing deployment with relative path...");
        try vm.deployCode("internal/senders/PrivateKeySender.sol:PrivateKeySender", args) returns (address sender) {
            console.log("SUCCESS: PrivateKeySender deployed at:", sender);
        } catch Error(string memory reason) {
            console.log("FAILED with reason:", reason);
        } catch {
            console.log("FAILED with low-level error");
        }
        
        console.log("TestDeployCodeScript completed");
    }
}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {TrebScript} from "treb-sol/TrebScript.sol";

contract TestDispatcherInitScript is Script {
    function run() public {
        console.log("Testing Dispatcher initialization...");
        
        // First, let's verify the env var is set
        try vm.envBytes("SENDER_CONFIGS") returns (bytes memory rawConfigs) {
            console.log("SENDER_CONFIGS found, length:", rawConfigs.length);
        } catch {
            console.log("SENDER_CONFIGS not found!");
            return;
        }
        
        // Now let's create a TrebScript instance and see what happens
        console.log("Creating TrebScript instance...");
        TestTrebScript test = new TestTrebScript();
        
        console.log("Calling test.testInit()...");
        try test.testInit() {
            console.log("Success!");
        } catch Error(string memory reason) {
            console.log("Failed with reason:", reason);
        } catch (bytes memory data) {
            console.log("Failed with data:");
            console.logBytes(data);
        }
    }
}

contract TestTrebScript is TrebScript {
    function testInit() external {
        console.log("Inside testInit, calling sender()...");
        
        // This should trigger the initialization
        try this.getSenderExternal("local") returns (address account) {
            console.log("Got sender account:", account);
        } catch Error(string memory reason) {
            console.log("Failed to get sender:", reason);
            revert(reason);
        } catch {
            console.log("Failed to get sender: unknown error");
            revert("Unknown error");
        }
    }
    
    function getSenderExternal(string memory name) external returns (address) {
        return sender(name).account;
    }
}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {console} from "forge-std/console.sol";

contract TestDeploySpecificScript is TrebScript {
    function run() public {
        console.log("TestDeploySpecificScript starting...");
        
        // Test if senders are loaded properly
        console.log("Testing sender access...");
        
        // Try accessing different senders
        try this.testSender("anvil") {
            console.log("Anvil sender access successful");
        } catch {
            console.log("Anvil sender access failed");
        }
        
        try this.testSender("local") {
            console.log("Local sender access successful"); 
        } catch {
            console.log("Local sender access failed");
        }
        
        try this.testSender("safe") {
            console.log("Safe sender access successful");
        } catch {
            console.log("Safe sender access failed");
        }
        
        console.log("TestDeploySpecificScript completed");
    }
    
    function testSender(string memory senderId) external {
        // This will trigger sender loading if not already loaded
        address senderAddr = sender(senderId).account;
        require(senderAddr != address(0), "Sender not found");
    }
}
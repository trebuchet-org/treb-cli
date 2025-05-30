// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {console} from "forge-std/console.sol";

contract TestTrebScriptMinimalScript is TrebScript {
    function run() public {
        console.log("TestTrebScriptMinimal starting...");
        
        // Just try to get a sender without using broadcast modifier
        console.log("Attempting to get 'local' sender...");
        address account = sender("local").account;
        console.log("Got local sender account:", account);
        
        console.log("Success!");
    }
}
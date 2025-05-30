// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {console} from "forge-std/console.sol";

contract MinimalTrebScript is TrebScript {
    function run() public {
        console.log("MinimalTrebScript starting...");
        
        // Just log something simple
        console.log("Dispatcher initialized");
        console.log("Registry initialized");
        
        console.log("MinimalTrebScript completed");
    }
}
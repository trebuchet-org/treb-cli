// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {console} from "forge-std/console.sol";

contract TestBroadcastIssueScript is TrebScript {
    function run() public broadcast {
        console.log("TestBroadcastIssue starting...");
        console.log("This should trigger the broadcast modifier");
        console.log("Completed");
    }
}
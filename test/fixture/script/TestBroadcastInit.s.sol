// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {console} from "forge-std/console.sol";

contract TestBroadcastInitScript is TrebScript {
    function run() public broadcast {
        console.log("TestBroadcastInit - this should trigger initialization");
        console.log("If we see this, initialization succeeded");
    }
}
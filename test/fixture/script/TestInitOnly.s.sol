// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {console} from "forge-std/console.sol";

contract TestInitOnlyScript is TrebScript {
    function run() public {
        console.log("TestInitOnly - testing initialization without broadcast");
        console.log("Script completed successfully");
    }
}
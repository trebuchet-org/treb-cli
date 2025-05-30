// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {console} from "forge-std/console.sol";

contract TestEventsSimpleScript {
    event TestEvent(string message, uint256 value);
    event DeploymentEvent(address deployed, string name);

    function run() public {
        console.log("Testing event emission");
        
        emit TestEvent("Hello events", 42);
        emit DeploymentEvent(address(0x123), "TestContract");
        
        console.log("Events emitted");
    }
}
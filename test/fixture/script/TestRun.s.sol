// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";
import {Counter} from "../src/Counter.sol";

contract TestRunScript is Script {
    event ContractDeployed(
        bytes32 indexed operationId,
        address indexed sender,
        address deployedAddress,
        bytes32 salt,
        bytes32 initCodeHash,
        string createStrategy
    );

    function run() public {
        // Start broadcasting
        vm.startBroadcast();

        // Deploy a simple counter contract
        Counter counter = new Counter();
        counter.setNumber(42);

        // Emit deployment event for treb to parse
        bytes32 operationId = keccak256("test-operation");
        bytes32 salt = keccak256("test-salt");
        bytes32 initCodeHash = keccak256(type(Counter).creationCode);
        
        emit ContractDeployed(
            operationId,
            msg.sender,
            address(counter),
            salt,
            initCodeHash,
            "create"
        );

        console.log("Deployed Counter at:", address(counter));
        console.log("Initial number:", counter.number());

        vm.stopBroadcast();
    }
}
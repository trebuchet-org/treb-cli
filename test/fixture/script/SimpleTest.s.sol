// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";
import {Counter} from "../src/Counter.sol";

contract SimpleTestScript is Script {
    // Event to match what our parser expects
    event ContractDeployed(
        bytes32 indexed operationId,
        address indexed sender,
        address deployedAddress,
        bytes32 salt,
        bytes32 initCodeHash,
        string createStrategy
    );

    function run() public {
        vm.startBroadcast();
        
        // Deploy counter
        Counter counter = new Counter();
        counter.setNumber(42);
        
        // Emit deployment event
        emit ContractDeployed(
            keccak256("simple-test"),
            msg.sender,
            address(counter),
            bytes32(0),
            keccak256(type(Counter).creationCode),
            "create"
        );
        
        console.log("Counter deployed at:", address(counter));
        console.log("Number set to:", counter.number());
        
        // Deploy another counter
        Counter counter2 = new Counter{salt: bytes32(uint256(1))}();
        counter2.setNumber(100);
        
        emit ContractDeployed(
            keccak256("simple-test-2"),
            msg.sender,
            address(counter2),
            bytes32(uint256(1)),
            keccak256(type(Counter).creationCode),
            "create2"
        );
        
        console.log("Counter2 deployed at:", address(counter2));
        console.log("Number set to:", counter2.number());
        
        vm.stopBroadcast();
    }
}
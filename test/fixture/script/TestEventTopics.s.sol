// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script, console} from "forge-std/Script.sol";

contract TestEventTopicsScript is Script {
    function run() public {
        console.log("Calculating event topics...");
        
        // Calculate event topics to verify they match Go code
        bytes32 deployingContractTopic = keccak256("DeployingContract(string,string,bytes32)");
        bytes32 contractDeployedTopic = keccak256("ContractDeployed(address,address,bytes32,bytes32,bytes32,bytes,string)");
        bytes32 safeTransactionQueuedTopic = keccak256("SafeTransactionQueued(address,address,bytes32,bytes32,uint256)");
        bytes32 bundleSentTopic = keccak256("BundleSent(address,bytes32,uint8,((string,address,bytes,uint256),bytes,bytes)[])");
        
        console.log("DeployingContract topic:");
        console.logBytes32(deployingContractTopic);
        
        console.log("ContractDeployed topic:");
        console.logBytes32(contractDeployedTopic);
        
        console.log("SafeTransactionQueued topic:");
        console.logBytes32(safeTransactionQueuedTopic);
        
        console.log("BundleSent topic:");
        console.logBytes32(bundleSentTopic);
    }
}
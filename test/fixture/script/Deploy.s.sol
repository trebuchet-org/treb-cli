// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "treb-sol/src/Script.sol";

contract Deploy is Script {
    function run() public {
        // Get the default sender (configured via treb profile)
        Sender deployer = sender("default");
        
        // Example: Deploy a simple contract
        address myContract = deployer.deployCreate3("MyContract.sol:MyContract");
        
        // Example: Deploy with constructor arguments
        bytes memory args = abi.encode(1000, "Hello");
        address tokenContract = deployer.deployCreate3("Token.sol:Token", args);
        
        // Example: Deploy with specific salt for deterministic address
        bytes32 salt = keccak256("my-unique-salt");
        address deterministicContract = deployer.deployCreate3(
            salt,
            getCode("DeterministicContract.sol:DeterministicContract"),
            ""
        );
        
        // Example: Look up existing deployment from registry
        address existingContract = getDeployment("ExistingContract");
        
        // Example: Deploy proxy pattern
        address implementation = deployer.deployCreate3("MyImplementation.sol:MyImplementation");
        bytes memory proxyArgs = abi.encode(implementation, "");
        address proxy = deployer.deployCreate3("ERC1967Proxy.sol:ERC1967Proxy", proxyArgs);
        
        // All deployments are automatically tracked via events
        // The treb CLI will parse these events and update the registry
    }
    
    // Helper function to get bytecode
    function getCode(string memory what) internal returns (bytes memory) {
        return vm.getCode(what);
    }
}

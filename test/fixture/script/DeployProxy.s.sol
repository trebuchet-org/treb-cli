// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {Counter} from "../src/Counter.sol";
import {UpgradeableCounter} from "../src/UpgradeableCounter.sol";
import {console} from "forge-std/console.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract DeployWithanvilScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        // This script uses the "anvil" sender which will queue transactions
        // instead of executing them directly
        
        // Get the sender
        Senders.Sender storage anvil = sender("anvil");
        
        // Deploy implementation with concatenated name
        address implementation = getDeployment("src/UpgradeableCounter.sol:UpgradeableCounter:UpgradeableCounter");
        console.log("Implementation to be deployed at:", implementation);
        
        // Deploy proxy with initialization
        // Note: For now we'll skip proxy deployment as it requires special handling
        address proxy = anvil.create3("ERC1967Proxy").setLabel(vm.envString("LABEL")).deploy(abi.encode(implementation, ""));
        console.log("Proxy to be deployed at:", proxy);
        
        // Deploy another counter for testing  
        console.log("All transactions queued for anvil execution");
        console.log("anvil address:", anvil.account);
    }
}
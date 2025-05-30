// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {Counter} from "../src/Counter.sol";
import {UpgradeableCounter} from "../src/UpgradeableCounter.sol";
import {console} from "forge-std/console.sol";

contract DeployWithSafeScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        // This script uses the "safe" sender which will queue transactions
        // instead of executing them directly
        
        // Get the sender
        Senders.Sender storage safe = sender("safe");
        
        // Deploy implementation with concatenated name
        address implementation = safe.create3("src/UpgradeableCounter.sol:UpgradeableCounter:impl-v1").deploy();
        console.log("Implementation to be deployed at:", implementation);
        
        // Deploy proxy with initialization
        // Note: For now we'll skip proxy deployment as it requires special handling
        // address proxy = safe.deployCreate3("UpgradeableCounter:proxy");
        // console.log("Proxy to be deployed at:", proxy);
        
        // Deploy another counter for testing  
        address counter2 = safe.create3("src/Counter.sol:Counter:v2").deploy();
        Counter(counter2).setNumber(42);
        
        console.log("All transactions queued for Safe execution");
        console.log("Safe address:", safe.account);
    }
}
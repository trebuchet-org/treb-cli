// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {Counter} from "../src/Counter.sol";
import {UpgradeableCounter} from "../src/UpgradeableCounter.sol";
import {console} from "forge-std/console.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

/// @title DeployerUCProxy
/// @notice This script deploys a proxy for an upgradeable counter contract
/// @dev This script uses the deployer sender to deploy the proxy and implementation
/// @dev The label is used to identify the deployment
/// @dev The deployer is the sender that will deploy the proxy and implementation
/// @dev The implementation is the upgradeable counter contract
/// @dev The proxy is the ERC1967 proxy contract
/// @custom:env-arg string label
/// @custom:env-arg string deployer
contract DeployerUCProxy is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        string memory label = vm.envString("label");
        string memory _deployer = vm.envString("deployer");

        Senders.Sender storage deployer = sender(_deployer);

        address implementation = deployer
            .create3("src/UpgradeableCounter.sol:UpgradeableCounter")
            .setLabel(label)
            .deploy();

        address proxy = deployer.create3("ERC1967Proxy").setLabel(label).deploy(
            abi.encode(implementation, "")
        );

        UpgradeableCounter(deployer.harness(proxy)).increment();
    }
}


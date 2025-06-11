// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
import {Counter} from "../src/Counter.sol";
import {UpgradeableCounter} from "../src/UpgradeableCounter.sol";
import {console} from "forge-std/console.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import {TransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import {UUPSUpgradeable} from "@openzeppelin/contracts/proxy/utils/UUPSUpgradeable.sol";

/// @title DeployerUCProxy
/// @notice This script deploys a proxy for an upgradeable counter contract
/// @dev This script uses the deployer sender to deploy the proxy and implementation
/// @dev The proxy is the ERC1967 proxy contract
contract DeployerUCProxy is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender} deployer The sender which will deploy the contract
     * @custom:env {string:optional} label Label for the proxy and implementation
     * @custom:env {deployment:optional} implementation Implementation to use for the proxy
     */
    function run() public broadcast {
        string memory label = vm.envOr("label", string(""));
        address implementation = vm.envOr("implementation", address(0));
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("deployer"))
        );

        if (implementation == address(0)) {
            implementation = deployer
                .create3("src/UpgradeableCounter.sol:UpgradeableCounter")
                .setLabel(label)
                .deploy();
        }

        address proxy = deployer.create3("ERC1967Proxy").setLabel(label).deploy(
            abi.encode(implementation, abi.encodeWithSelector(UpgradeableCounter.initialize.selector, deployer.account))
        );

        UpgradeableCounter(deployer.harness(proxy)).increment();
    }
}

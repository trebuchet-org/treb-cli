// SPDX-License-Identifier: MIT
pragma solidity ^0.8;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

import {UpgradeableCounterV2} from "../src/UpgradeableCounterV2.sol";

interface IProxy {
    function _setImplementation(address implementation) external;
}

/// @title UpgradeCustomProxy
/// @notice Upgrades the custom proxy to use UpgradeableCounterV2 as the new implementation
contract UpgradeCustomProxy is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender("anvil");

        // Deploy new implementation
        address newImplementation = deployer.create3("UpgradeableCounterV2").deploy();

        // Look up existing proxy from registry
        address proxy = lookup("Proxy");

        // Upgrade the proxy to the new implementation
        IProxy(deployer.harness(proxy))._setImplementation(newImplementation);
    }
}

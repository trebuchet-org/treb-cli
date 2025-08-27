// SPDX-License-Identifier: MIT
pragma solidity =0.8.30;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

import {UpgradeableCounter} from "../src/UpgradeableCounter.sol";

interface IProxy {
    function _setAndInitializeImplementation(
        address implementation,
        bytes calldata callbackData
    ) external;
}

/// @title DeployerCustomProxy
/// @notice This script deploys a custom proxy for an upgradeable counter contract
contract DeployerCustomProxy is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {string:optional} label Label for the proxy and implementation
     * @custom:env {deployment:optional} implementation Implementation to use for the proxy
     * @custom:senders anvil
     */
    function run() public broadcast {
        string memory label = vm.envOr("label", string(""));
        address implementation = vm.envOr("implementation", address(0));
        Senders.Sender storage deployer = sender("anvil");

        if (implementation == address(0)) {
            implementation = deployer.create3("UpgradeableCounter").deploy();
        }

        address proxy = deployer
            .create3("src/Proxy.sol:Proxy")
            .setLabel(label)
            .deploy(abi.encode(deployer.account));
        bytes memory init = abi.encodeWithSelector(
            UpgradeableCounter.initialize.selector,
            deployer.account
        );

        IProxy(deployer.harness(proxy))._setAndInitializeImplementation(
            implementation,
            init
        );

        UpgradeableCounter uc = UpgradeableCounter(deployer.harness(proxy));
        uc.increment();
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";
import {Transaction} from "treb-sol/src/internal/types.sol";

import {Counter} from "../src/Counter.sol";
import {UpgradeableCounter} from "../src/UpgradeableCounter.sol";
import {console} from "forge-std/console.sol";
import {ERC1967Proxy} from "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import {TransparentUpgradeableProxy} from "lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import {UUPSUpgradeable} from "lib/openzeppelin-contracts/contracts/proxy/utils/UUPSUpgradeable.sol";

/// @title DeployerUCProxy
/// @notice This script deploys a proxy for an upgradeable counter contract
/// @dev This script uses the deployer sender to deploy the proxy and implementation
/// @dev The proxy is the ERC1967 proxy contract
contract DeployerUCProxy is TrebScript {
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
            implementation = deployer
                .create3("src/UpgradeableCounter.sol:UpgradeableCounter")
                .deploy();
        }

        address proxy = deployer.create3("ERC1967Proxy").setLabel(label).deploy(
            abi.encode(
                implementation,
                abi.encodeWithSelector(
                    UpgradeableCounter.initialize.selector,
                    deployer.account
                )
            )
        );

        UpgradeableCounter uc = UpgradeableCounter(deployer.harness(proxy));
        uc.increment();
    }
}

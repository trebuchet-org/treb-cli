// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";

/**
 * @title DeployUpgradeableCounterProxy
 * @notice Deployment script for UpgradeableCounter with TransparentUpgradeable Proxy
 * @dev Generated automatically by treb
 */

import { TransparentUpgradeableProxy } from "lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

contract DeployUpgradeableCounterProxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol:TransparentUpgradeableProxy",
        "src/UpgradeableCounter.sol:UpgradeableCounter",
        DeployStrategy.CREATE3
    ) {}


    /// @notice Get constructor arguments - override to include admin parameter
    function _getConstructorArgs() internal view override returns (bytes memory) {
        address admin = executor; // Use executor as the ProxyAdmin owner
        bytes memory initData = _getProxyInitializer();
        return abi.encode(implementationAddress, admin, initData);
    }

    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal view override returns (bytes memory) {
        // Initialize method arguments detected from ABI
        address initialOwner = address(0);
        
        bytes4 selector = bytes4(keccak256("initialize(address)"));
        return abi.encodeWithSelector(selector, initialOwner);
    }

}
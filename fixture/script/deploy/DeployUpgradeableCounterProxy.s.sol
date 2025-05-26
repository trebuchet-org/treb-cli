// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";

/**
 * @title DeployUpgradeableCounterProxy
 * @notice Deployment script for UpgradeableCounter with Transparent Upgradeable Proxy
 * @dev Generated automatically by treb
 */
contract DeployUpgradeableCounterProxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "TransparentUpgradeableProxy",
        "UpgradeableCounter",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get constructor arguments - override to include admin parameter
    function _getConstructorArgs() internal view override returns (bytes memory) {
        address implementation = implementationAddress;
        address admin = executor; // Use executor as the ProxyAdmin owner
        bytes memory initData = _getProxyInitializer();
        return abi.encode(implementation, admin, initData);
    }

    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal view override returns (bytes memory) {
        // Initialize method arguments detected from ABI
        address initialOwner = executor; // Use the deployer as initial owner
        
        bytes4 selector = bytes4(keccak256("initialize(address)"));
        return abi.encodeWithSelector(selector, initialOwner);
    }
}
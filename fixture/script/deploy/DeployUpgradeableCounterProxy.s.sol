// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";
import {TransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import { UpgradeableCounter } from "../../src/UpgradeableCounter.sol";

/**
 * @title DeployUpgradeableCounterProxy
 * @notice Deployment script for UpgradeableCounter with Transparent Upgradeable Proxy
 * @dev Generated automatically by treb
 */
contract DeployUpgradeableCounterProxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "UpgradeableCounterProxy",
        "lib/openzeppelin-contracts/contracts/proxy/transparent/TransparentUpgradeableProxy.sol:TransparentUpgradeableProxy",
        DeployStrategy.CREATE3,
        "UpgradeableCounter"
    ) {}

    /// @notice Get contract bytecode for the proxy
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(TransparentUpgradeableProxy).creationCode;
    }

    /// @notice Get constructor arguments - override to include admin parameter
    function _getConstructorArgs() internal view override returns (bytes memory) {
        address implementation = getDeployment(implementationIdentifier);
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
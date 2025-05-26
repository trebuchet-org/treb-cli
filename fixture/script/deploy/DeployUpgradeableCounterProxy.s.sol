// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";

/**
 * @title DeployUpgradeableCounterProxy
 * @notice Deployment script for UpgradeableCounter with UUPSUpgradeable Proxy
 * @dev Generated automatically by treb
 */

import { ERC1967Proxy } from "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract DeployUpgradeableCounterProxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy",
        "src/UpgradeableCounter.sol:UpgradeableCounter",
        DeployStrategy.CREATE3
    ) {}


    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal view override returns (bytes memory) {
        // Initialize method arguments detected from ABI
        address initialOwner = address(0);
        
        bytes4 selector = bytes4(keccak256("initialize(address)"));
        return abi.encodeWithSelector(selector, initialOwner);
    }

}
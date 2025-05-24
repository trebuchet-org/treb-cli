// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import { Counter } from "../../src/Counter.sol";

/**
 * @title DeployCounterProxy
 * @notice Deployment script for Counter with UUPS Upgradeable Proxy
 * @dev Generated automatically by treb
 */
contract DeployCounterProxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "Counter",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode for the proxy
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(ERC1967Proxy).creationCode;
    }

    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal pure override returns (bytes memory) {
        // No initialize method detected - proxy will be deployed without initialization
        return "";
    }

}
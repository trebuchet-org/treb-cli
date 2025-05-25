// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
import { UpgradeableCounter } from "../../src/UpgradeableCounter.sol";

/**
 * @title DeployUpgradeableCounter
 * @notice Deployment script for UpgradeableCounter contract
 * @dev Generated automatically by treb
 */
contract DeployUpgradeableCounter is Deployment {
    constructor() Deployment(
        "UpgradeableCounter",
        "src/UpgradeableCounter.sol:UpgradeableCounter",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(UpgradeableCounter).creationCode;
    }


}
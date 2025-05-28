// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployUpgradeableCounter
 * @notice Deployment script for UpgradeableCounter contract
 * @dev Generated automatically by treb
 */
contract DeployUpgradeableCounter is Deployment {
    constructor() Deployment(
        "UpgradeableCounter",
        DeployStrategy.CREATE2
    ) {}


}
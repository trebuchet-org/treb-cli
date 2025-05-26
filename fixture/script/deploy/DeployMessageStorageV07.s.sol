// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployMessageStorageV07
 * @notice Deployment script for MessageStorageV07 contract
 * @dev Generated automatically by treb
 * @dev Target contract version: 0.7.6 (cross-version deployment)
 */
contract DeployMessageStorageV07 is Deployment {
    constructor() Deployment(
        "MessageStorageV07",
        DeployStrategy.CREATE3
    ) {}


}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
// Target contract uses Solidity 0.7.6, which is incompatible with this deployment script (0.8)
// Import commented out to avoid version conflicts. Using artifact-based deployment instead.
// import "../../src/MessageStorageV07.sol";

/**
 * @title DeployMessageStorageV07
 * @notice Deployment script for MessageStorageV07 contract
 * @dev Generated automatically by treb
 * @dev Target contract version: 0.7.6 (cross-version deployment)
 */
contract DeployMessageStorageV07 is Deployment {
    constructor() Deployment(
        "src/MessageStorageV07.sol:MessageStorageV07",
        DeployStrategy.CREATE3
    ) {}
}
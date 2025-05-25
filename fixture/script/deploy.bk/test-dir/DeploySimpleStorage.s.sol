// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
// Target contract uses Solidity unknown, which is incompatible with this deployment script (0.8)
// Import commented out to avoid version conflicts. Using artifact-based deployment instead.
// import "../../../src/test-dir/SimpleStorage.sol";

/**
 * @title DeploySimpleStorage
 * @notice Deployment script for SimpleStorage contract
 * @dev Generated automatically by treb
 * @dev Target contract version: unknown (cross-version deployment)
 */
contract DeploySimpleStorage is Deployment {
    constructor() Deployment(
        "src/test-dir/SimpleStorage.sol:SimpleStorage",
        DeployStrategy.CREATE3
    ) {}
}
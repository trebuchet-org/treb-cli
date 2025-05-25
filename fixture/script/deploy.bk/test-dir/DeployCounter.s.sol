// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
// Target contract uses Solidity unknown, which is incompatible with this deployment script (0.8)
// Import commented out to avoid version conflicts. Using artifact-based deployment instead.
// import "../../../src/test-dir/Counter.sol";

/**
 * @title DeployCounter
 * @notice Deployment script for Counter contract
 * @dev Generated automatically by treb
 * @dev Target contract version: unknown (cross-version deployment)
 */
contract DeployCounter is Deployment {
    constructor() Deployment(
        "src/test-dir/Counter.sol:Counter",
        DeployStrategy.CREATE3
    ) {}
}
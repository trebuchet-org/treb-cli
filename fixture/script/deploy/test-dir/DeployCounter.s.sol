// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
/**
 * @title DeployCounter
 * @notice Deployment script for Counter contract
 * @dev Generated automatically by treb
 */
contract DeployCounter is Deployment {
    constructor() Deployment(
        "src/test-dir/Counter.sol:Counter",
        DeployStrategy.CREATE3
    ) {}
}
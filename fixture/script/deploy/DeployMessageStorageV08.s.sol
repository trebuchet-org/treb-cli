// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployMessageStorageV08
 * @notice Deployment script for MessageStorageV08 contract
 * @dev Generated automatically by treb
 */
contract DeployMessageStorageV08 is Deployment {
    constructor() Deployment(
        "MessageStorageV08",
        DeployStrategy.CREATE3
    ) {}


}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployTestCounter
 * @notice Deployment script for TestCounter contract
 * @dev Generated automatically by treb
 */
contract DeployTestCounter is Deployment {
    constructor() Deployment(
        "TestCounter",
        DeployStrategy.CREATE3
    ) {}


}
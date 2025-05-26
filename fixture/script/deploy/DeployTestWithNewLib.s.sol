// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployTestWithNewLib
 * @notice Deployment script for TestWithNewLib contract
 * @dev Generated automatically by treb
 */
contract DeployTestWithNewLib is Deployment {
    constructor() Deployment(
        "TestWithNewLib",
        DeployStrategy.CREATE3
    ) {}


}
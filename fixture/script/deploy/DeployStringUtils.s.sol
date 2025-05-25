// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {LibraryDeployment} from "treb-sol/LibraryDeployment.sol";
import { StringUtils } from "../../src/StringUtils.sol";

/**
 * @title DeployStringUtils
 * @notice Deployment script for StringUtils library
 * @dev Generated automatically by treb
 * @dev Libraries are deployed globally (no environment) for cross-chain consistency
 */
contract DeployStringUtils is LibraryDeployment {
    constructor() LibraryDeployment("StringUtils") {}
}

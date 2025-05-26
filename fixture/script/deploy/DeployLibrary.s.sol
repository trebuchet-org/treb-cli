// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {LibraryDeployment} from "treb-sol/LibraryDeployment.sol";

/// @title DeployLibrary
/// @notice Concrete implementation of LibraryDeployment for deploying libraries
contract DeployLibrary is LibraryDeployment {
    constructor() LibraryDeployment() {}
}
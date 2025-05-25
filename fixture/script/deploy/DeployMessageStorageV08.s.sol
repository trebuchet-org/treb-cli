// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
import { MessageStorageV08 } from "../../src/MessageStorageV08.sol";

/**
 * @title DeployMessageStorageV08
 * @notice Deployment script for MessageStorageV08 contract
 * @dev Generated automatically by treb
 */
contract DeployMessageStorageV08 is Deployment {
    constructor() Deployment(
        "MessageStorageV08",
        "src/MessageStorageV08.sol:MessageStorageV08",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(MessageStorageV08).creationCode;
    }


}
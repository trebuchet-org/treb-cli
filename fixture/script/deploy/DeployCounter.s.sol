// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
import {Counter} from "../../src/Counter.sol";

/**
 * @title DeployCounter
 * @notice Deployment script for Counter contract
 * @dev Generated automatically by treb
 */
contract DeployCounter is Deployment {
    constructor() Deployment(
        "Counter",
        "src/Counter.sol:Counter",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(Counter).creationCode;
    }
}
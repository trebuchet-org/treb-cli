// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
import { Counter } from "../../src/Counter.sol";

/**
 * @title DeployCounter
 * @notice Deployment script for Counter contract
 * @dev Generated automatically by treb
 */
contract DeployCounter is ContractDeployment {
    constructor() ContractDeployment(
        "Counter",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(Counter).creationCode;
    }


}
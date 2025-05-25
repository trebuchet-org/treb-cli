// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
import {CalculatorV08} from "../../src/CalculatorV08.sol";

/**
 * @title DeployCalculatorV08
 * @notice Deployment script for CalculatorV08 contract
 * @dev Generated automatically by treb
 */
contract DeployCalculatorV08 is ContractDeployment {
    constructor() ContractDeployment(
        "CalculatorV08",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal view override returns (bytes memory) {
        return type(CalculatorV08).creationCode;
    }
}
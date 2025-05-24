// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
import { UpgradeableCounter } from "../../src/UpgradeableCounter.sol";

/**
 * @title DeployUpgradeableCounter
 * @notice Deployment script for UpgradeableCounter contract
 * @dev Generated automatically by treb
 */
contract DeployUpgradeableCounter is ContractDeployment {
    constructor() ContractDeployment(
        "UpgradeableCounter",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(UpgradeableCounter).creationCode;
    }


}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "forge-deploy/CreateXDeployment.sol";
import "../../src/Counter.sol";

/**
 * @title DeployCounter
 * @notice Deployment script for Counter contract
 * @dev Generated automatically by fdeploy
 */
contract DeployCounter is CreateXDeployment {
    constructor() CreateXDeployment(
        "Counter",
        DeploymentType.IMPLEMENTATION,
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function getContractBytecode() internal pure override returns (bytes memory) {
        return type(Counter).creationCode;
    }


}
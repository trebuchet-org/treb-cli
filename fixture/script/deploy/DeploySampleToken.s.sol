// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
import {SampleToken} from "../../src/SampleToken.sol";

/**
 * @title DeploySampleToken
 * @notice Deployment script for SampleToken contract
 * @dev Generated automatically by treb
 */
contract DeploySampleToken is ContractDeployment {
    constructor() ContractDeployment(
        "SampleToken",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(SampleToken).creationCode;
    }

    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure override returns (bytes memory) {
        // Constructor arguments detected from ABI
        string memory _name = "";
        string memory _symbol = "";
        uint256 _totalSupply = 0;
        return abi.encode(_name, _symbol, _totalSupply);
    }
}
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
import {MyToken} from "../../src/tokens/MyToken.sol";

/**
 * @title DeployMyToken
 * @notice Deployment script for MyToken contract
 * @dev Generated automatically by treb
 */
contract DeployMyToken is ContractDeployment {
    constructor() ContractDeployment(
        "MyToken",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(MyToken).creationCode;
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
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

/**
 * @title DeployMyToken
 * @notice Deployment script for MyToken contract
 * @dev Generated automatically by treb
 */
contract DeployMyToken is Deployment {
    constructor() Deployment(
        "src/tokens/MyToken.sol:MyToken",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure override returns (bytes memory) {
        // Constructor arguments detected from ABI
        string memory _name = "";
        string memory _symbol = "";
        uint256 _totalSupply = 0;
        return abi.encode(_name, _symbol, _totalSupply);
    }
}
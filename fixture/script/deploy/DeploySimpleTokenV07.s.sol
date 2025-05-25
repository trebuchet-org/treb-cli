// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
// Target contract uses Solidity 0.7.0, which is incompatible with this deployment script (0.8)
// Import commented out to avoid version conflicts. Using artifact-based deployment instead.
// import "../../src/SimpleTokenV07.sol";

/**
 * @title DeploySimpleTokenV07
 * @notice Deployment script for SimpleTokenV07 contract
 * @dev Generated automatically by treb
 * @dev Target contract version: 0.7.0 (cross-version deployment)
 */
contract DeploySimpleTokenV07 is Deployment {
    constructor() Deployment(
        "SimpleTokenV07",
        "src/SimpleTokenV07.sol:SimpleTokenV07",
        DeployStrategy.CREATE3
    ) {}

    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure override returns (bytes memory) {
        // Constructor arguments detected from ABI
        string memory _name = "Testy";
        string memory _symbol = "TT";
        uint256 _totalSupply = 1000;
        return abi.encode(_name, _symbol, _totalSupply);
    }
}
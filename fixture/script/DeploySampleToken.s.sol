// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "forge-deploy/base/CreateXDeployment.sol";
import "../src/SampleToken.sol";

/**
 * @title DeploySampleToken
 * @notice Deployment script for SampleToken using forge-deploy library
 */
contract DeploySampleToken is CreateXDeployment {
    // Token configuration
    string private constant TOKEN_NAME = "Sample Token";
    string private constant TOKEN_SYMBOL = "SAMPLE";
    uint256 private constant TOTAL_SUPPLY = 1_000_000e18;
    
    constructor() CreateXDeployment(
        "SampleToken",
        vm.envOr("CONTRACT_VERSION", string("v1.0.0")),
        _buildSaltComponents()
    ) {}
    
    /// @notice Build salt components for deterministic deployment
    function _buildSaltComponents() private view returns (string[] memory) {
        string[] memory components = new string[](3);
        components[0] = "SampleToken";
        components[1] = vm.envOr("CONTRACT_VERSION", string("v1.0.0"));
        components[2] = vm.envOr("DEPLOYMENT_ENV", string("staging"));
        return components;
    }
    
    /// @notice Get the init code for the contract deployment
    function getInitCode() internal pure override returns (bytes memory) {
        return abi.encodePacked(
            type(SampleToken).creationCode,
            abi.encode(TOKEN_NAME, TOKEN_SYMBOL, TOTAL_SUPPLY)
        );
    }
}
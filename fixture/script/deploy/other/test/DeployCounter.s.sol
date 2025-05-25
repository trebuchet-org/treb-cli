// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment} from "treb-sol/Deployment.sol";
import {DeployStrategy, DeploymentResult} from "treb-sol/internal/type.sol";
import {Counter} from "../../../../src/other/test/Counter.sol";

contract DeployCounter is Deployment {
    constructor() Deployment(
        "src/other/test/Counter.sol:Counter",
        DeployStrategy.CREATE3
    ) {}

    function _getInitCode() internal pure override returns (bytes memory) {
        return type(Counter).creationCode;
    }

    function _preDeploy() internal override {}

    function _postDeploy(DeploymentResult memory) internal override {}
}
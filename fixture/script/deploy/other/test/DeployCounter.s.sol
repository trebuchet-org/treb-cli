// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";

contract DeployCounter is Deployment {
    constructor() Deployment(
        "src/other/test/Counter.sol:Counter",
        DeployStrategy.CREATE3
    ) {}
}
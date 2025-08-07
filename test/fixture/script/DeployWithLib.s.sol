// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";

contract DeployWithLib is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );
        deployer.create2("TestWithNewLib.sol:TestWithNewLib").deploy();
    }
}

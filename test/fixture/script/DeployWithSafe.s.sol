// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";

import {Counter} from "../src/Counter.sol";
import {SampleToken} from "../src/SampleToken.sol";
import {console} from "forge-std/console.sol";

contract DeployWithTrebScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;
    using Senders for Senders.Sender;

    function run() public broadcast {
        // Get the sender
        Senders.Sender storage deployer = sender("safe");

        address _counter = deployer
            .create3("src/Counter.sol:Counter")
            .setLabel("test")
            .deploy();
        Counter counter = Counter(deployer.harness(_counter));
        counter.increment();
    }
}

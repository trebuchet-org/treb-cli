// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";
import {OZGovernor} from "treb-sol/src/internal/sender/OZGovernorSender.sol";

import {Counter} from "../src/Counter.sol";

contract DeployWithGovernorScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;
    using Senders for Senders.Sender;
    using OZGovernor for OZGovernor.Sender;

    /**
     * @custom:senders governor
     */
    function run() public broadcast {
        Senders.Sender storage gov = sender("governor");

        // Cast to OZGovernor sender and set proposal description
        OZGovernor.Sender storage ozGov = OZGovernor.cast(gov);
        ozGov.setProposalDescription("Deploy Counter via governance");

        // Deploy counter through governance
        address counter = gov.create3("src/Counter.sol:Counter").setLabel("gov").deploy();

        // Interact through governance
        Counter(gov.harness(counter)).increment();
    }
}

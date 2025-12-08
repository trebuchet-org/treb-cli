// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

import {GovernanceToken} from "../src/governance/GovernanceToken.sol";
import {TrebTimelock} from "../src/governance/TrebTimelock.sol";
import {TrebGovernor} from "../src/governance/TrebGovernor.sol";

contract DeployGovernanceScript is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;
    using Senders for Senders.Sender;

    /**
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender("anvil");

        // Deploy governance token
        address token = deployer.create3("src/governance/GovernanceToken.sol:GovernanceToken").deploy();

        // Setup timelock with minimal delay for testing
        address[] memory proposers = new address[](0);
        address[] memory executors = new address[](1);
        executors[0] = address(0); // Anyone can execute

        address timelock = deployer.create3("src/governance/TrebTimelock.sol:TrebTimelock").deploy(
            abi.encode(
                uint256(1),  // 1 second min delay for testing
                proposers,
                executors,
                deployer.account
            )
        );

        // Deploy governor with token and timelock
        address governor = deployer.create3("src/governance/TrebGovernor.sol:TrebGovernor").deploy(
            abi.encode(token, timelock)
        );

        // Grant proposer role to governor on timelock
        TrebTimelock(payable(deployer.harness(timelock))).grantRole(
            TrebTimelock(payable(timelock)).PROPOSER_ROLE(),
            governor
        );

        // Self-delegate to deployer for voting power
        GovernanceToken(deployer.harness(token)).delegate(deployer.account);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.23;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Counter} from "src/Counter.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

/**
 * @notice Deployment script that can be controlled to fail via environment variable
 * @dev Uses FLAKY_SHOULD_FAIL env var to determine if deployment should revert
 * @custom:senders anvil
 */
contract DeployFlaky is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /// @custom:senders anvil
    function run() public broadcast {
        // Read the environment variable to determine if we should fail
        Senders.Sender storage deployer = sender("anvil");
        bool shouldFail = vm.envOr("FLAKY_SHOULD_FAIL", false);

        if (shouldFail) {
            revert("Deployment intentionally failed for testing");
        }

        deployer.create3("UpgradeableCounter").deploy();
    }
}


// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";

/// @notice Fork setup script that always reverts (for testing failure handling)
contract SetupForkFailing is Script {
    function run() public pure {
        revert("SetupFork intentional failure");
    }
}

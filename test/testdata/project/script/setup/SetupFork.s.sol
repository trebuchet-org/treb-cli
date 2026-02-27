// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";

/// @notice Fork setup script that gives a known address some ETH for testing.
/// Uses an actual ETH transfer so the state persists on the fork anvil.
contract SetupFork is Script {
    // Known test address to receive ETH
    address constant TEST_ADDR = 0x1234567890123456789012345678901234567890;

    // Anvil's default account #0 private key (has 10000 ETH)
    uint256 constant ANVIL_PK = 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80;

    function run() public {
        vm.startBroadcast(ANVIL_PK);
        payable(TEST_ADDR).transfer(100 ether);
        vm.stopBroadcast();
    }
}

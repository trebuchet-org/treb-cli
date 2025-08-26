// SPDX-License-Identifier: MIT
pragma solidity =0.8.30;

contract TestCounter {
    uint256 public count;

    function increment() public {
        count++;
    }

    function getCount() public view returns (uint256) {
        return count;
    }
}


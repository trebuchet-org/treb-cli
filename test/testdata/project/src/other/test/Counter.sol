// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.30;

contract Counter {
    uint256 public number;
    uint256 public constant MAX_NUMBER = 100;

    function setNumber(uint256 newNumber) public {
        number = newNumber;
    }

    function increment() public {
        number++;
    }
}

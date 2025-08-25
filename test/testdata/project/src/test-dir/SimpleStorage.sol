// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

contract SimpleStorage {
    uint256 private storedData;
    address public owner;

    event DataStored(uint256 data, address indexed setter);

    constructor(uint256 _initialValue) {
        storedData = _initialValue;
        owner = msg.sender;
    }

    function set(uint256 _value) public {
        storedData = _value;
        emit DataStored(_value, msg.sender);
    }

    function get() public view returns (uint256) {
        return storedData;
    }

    function getOwner() public view returns (address) {
        return owner;
    }
}


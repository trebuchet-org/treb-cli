// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract UpgradeableCounter {
    uint256 private _count;
    address private _owner;
    bool private _initialized;

    event CountIncremented(uint256 newCount);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    modifier onlyOwner() {
        require(msg.sender == _owner, "Not the owner");
        _;
    }

    function initialize(address initialOwner) external {
        require(!_initialized, "Already initialized");
        _initialized = true;
        _owner = initialOwner;
        _count = 0;
    }

    function increment() external {
        _count += 1;
        emit CountIncremented(_count);
    }

    function getCount() external view returns (uint256) {
        return _count;
    }

    function owner() external view returns (address) {
        return _owner;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "New owner is zero address");
        address oldOwner = _owner;
        _owner = newOwner;
        emit OwnershipTransferred(oldOwner, newOwner);
    }
}
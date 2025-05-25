// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title SampleToken
 * @notice A simple ERC20-like token for testing deployments
 */
contract MyToken {
    string public name;
    string public symbol;
    uint8 public decimals;
    uint256 public totalSupply;
    
    mapping(address => uint256) public balanceOf;
    
    constructor(string memory _name, string memory _symbol, uint256 _totalSupply) {
        name = _name;
        symbol = _symbol;
        decimals = 18;
        totalSupply = _totalSupply;
        balanceOf[msg.sender] = _totalSupply;
    }

    function mint(address to, uint256 amount) public {
        balanceOf[to] += amount;
        totalSupply += amount;
    }

    function burn(address from, uint256 amount) public {
        balanceOf[from] -= amount;
        totalSupply -= amount;
    }
}
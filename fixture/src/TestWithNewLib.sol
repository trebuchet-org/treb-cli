// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

library MathUtils {
    function add(uint256 a, uint256 b) public pure returns (uint256) {
        return a + b;
    }
    
    function multiply(uint256 a, uint256 b) public pure returns (uint256) {
        return a * b;
    }
}

contract TestWithNewLib {
    using MathUtils for uint256;
    
    uint256 public result;
    
    function calculate(uint256 a, uint256 b) public {
        result = a.add(b).multiply(2);
    }
}
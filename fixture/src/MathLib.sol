// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

library MathLib {
    function add(uint256 a, uint256 b) public pure returns (uint256) {
        return a + b;
    }
    
    function multiply(uint256 a, uint256 b) public pure returns (uint256) {
        return a * b;
    }
    
    function power(uint256 base, uint256 exp) public pure returns (uint256) {
        if (exp == 0) return 1;
        uint256 result = base;
        for (uint256 i = 1; i < exp; i++) {
            result = multiply(result, base);
        }
        return result;
    }
    
    function factorial(uint256 n) public pure returns (uint256) {
        if (n <= 1) return 1;
        uint256 result = 1;
        for (uint256 i = 2; i <= n; i++) {
            result = multiply(result, i);
        }
        return result;
    }
}
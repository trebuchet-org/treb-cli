// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

import "./MathLibV07.sol";

contract CalculatorV07 {
    using MathLibV07 for uint256;
    
    uint256 public lastResult;
    address public owner;
    
    event Calculation(string operation, uint256 a, uint256 b, uint256 result);
    
    constructor() {
        owner = msg.sender;
        lastResult = 0;
    }
    
    function addNumbers(uint256 a, uint256 b) public returns (uint256) {
        uint256 result = MathLibV07.add(a, b);
        lastResult = result;
        emit Calculation("add", a, b, result);
        return result;
    }
    
    function multiplyNumbers(uint256 a, uint256 b) public returns (uint256) {
        uint256 result = MathLibV07.multiply(a, b);
        lastResult = result;
        emit Calculation("multiply", a, b, result);
        return result;
    }
    
    function powerOf(uint256 base, uint256 exp) public returns (uint256) {
        uint256 result = MathLibV07.power(base, exp);
        lastResult = result;
        emit Calculation("power", base, exp, result);
        return result;
    }
    
    function factorialOf(uint256 n) public returns (uint256) {
        require(n <= 20, "Factorial too large");
        uint256 result = MathLibV07.factorial(n);
        lastResult = result;
        emit Calculation("factorial", n, 0, result);
        return result;
    }
    
    function complexCalculation(uint256 a, uint256 b, uint256 c) public returns (uint256) {
        // (a + b) * c^2
        uint256 sum = MathLibV07.add(a, b);
        uint256 cSquared = MathLibV07.power(c, 2);
        uint256 result = MathLibV07.multiply(sum, cSquared);
        lastResult = result;
        emit Calculation("complex", a, b, result);
        return result;
    }
    
    function getLastResult() public view returns (uint256) {
        return lastResult;
    }
}
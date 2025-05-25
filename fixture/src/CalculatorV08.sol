// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./MathLib.sol";

contract CalculatorV08 {
    using MathLib for uint256;
    
    uint256 public lastResult;
    address public owner;
    
    event Calculation(string operation, uint256 a, uint256 b, uint256 result);
    
    constructor() {
        owner = msg.sender;
        lastResult = 0;
    }
    
    function addNumbers(uint256 a, uint256 b) public returns (uint256) {
        uint256 result = MathLib.add(a, b);
        lastResult = result;
        emit Calculation("add", a, b, result);
        return result;
    }
    
    function multiplyNumbers(uint256 a, uint256 b) public returns (uint256) {
        uint256 result = MathLib.multiply(a, b);
        lastResult = result;
        emit Calculation("multiply", a, b, result);
        return result;
    }
    
    function powerOf(uint256 base, uint256 exp) public returns (uint256) {
        uint256 result = MathLib.power(base, exp);
        lastResult = result;
        emit Calculation("power", base, exp, result);
        return result;
    }
    
    function factorialOf(uint256 n) public returns (uint256) {
        require(n <= 20, "Factorial too large");
        uint256 result = MathLib.factorial(n);
        lastResult = result;
        emit Calculation("factorial", n, 0, result);
        return result;
    }
    
    function complexCalculation(uint256 a, uint256 b, uint256 c) public returns (uint256) {
        // (a + b) * c^2
        uint256 sum = MathLib.add(a, b);
        uint256 cSquared = MathLib.power(c, 2);
        uint256 result = MathLib.multiply(sum, cSquared);
        lastResult = result;
        emit Calculation("complex", a, b, result);
        return result;
    }
    
    function getLastResult() public view returns (uint256) {
        return lastResult;
    }
}
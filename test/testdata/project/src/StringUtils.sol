// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0 || ^0.8.0;

library StringUtils {
    // External function to concatenate two strings
    function concat(string memory a, string memory b) external pure returns (string memory) {
        return string(abi.encodePacked(a, b));
    }
    
    // External function to get string length
    function length(string memory str) external pure returns (uint256) {
        return bytes(str).length;
    }
    
    // External function to convert string to uppercase (simplified for ASCII)
    function toUpperCase(string memory str) external pure returns (string memory) {
        bytes memory strBytes = bytes(str);
        bytes memory result = new bytes(strBytes.length);
        
        for (uint256 i = 0; i < strBytes.length; i++) {
            uint8 char = uint8(strBytes[i]);
            // Check if character is lowercase letter (a-z)
            if (char >= 97 && char <= 122) {
                // Convert to uppercase by subtracting 32
                result[i] = bytes1(char - 32);
            } else {
                result[i] = strBytes[i];
            }
        }
        
        return string(result);
    }
    
    // External function to check if strings are equal
    function equal(string memory a, string memory b) external pure returns (bool) {
        return keccak256(abi.encodePacked(a)) == keccak256(abi.encodePacked(b));
    }
}
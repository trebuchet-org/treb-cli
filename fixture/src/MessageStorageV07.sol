// SPDX-License-Identifier: MIT
pragma solidity ^0.7.6;

import "./StringUtils.sol";

contract MessageStorageV07 {
    using StringUtils for string;
    
    mapping(address => string) private messages;
    mapping(address => uint256) private messageLengths;
    
    event MessageStored(address indexed user, string message, uint256 length);
    event MessageUpdated(address indexed user, string oldMessage, string newMessage);
    
    // Store a message for the sender
    function storeMessage(string memory message) public {
        messages[msg.sender] = message;
        messageLengths[msg.sender] = StringUtils.length(message);
        emit MessageStored(msg.sender, message, messageLengths[msg.sender]);
    }
    
    // Get the stored message for an address
    function getMessage(address user) public view returns (string memory) {
        return messages[user];
    }
    
    // Get the length of stored message
    function getMessageLength(address user) public view returns (uint256) {
        return messageLengths[user];
    }
    
    // Append to existing message
    function appendMessage(string memory additionalText) public {
        string memory currentMessage = messages[msg.sender];
        string memory newMessage = StringUtils.concat(currentMessage, additionalText);
        
        emit MessageUpdated(msg.sender, currentMessage, newMessage);
        
        messages[msg.sender] = newMessage;
        messageLengths[msg.sender] = StringUtils.length(newMessage);
    }
    
    // Store message in uppercase
    function storeUppercaseMessage(string memory message) public {
        string memory upperMessage = StringUtils.toUpperCase(message);
        messages[msg.sender] = upperMessage;
        messageLengths[msg.sender] = StringUtils.length(upperMessage);
        emit MessageStored(msg.sender, upperMessage, messageLengths[msg.sender]);
    }
    
    // Check if user has a specific message
    function hasMessage(address user, string memory expectedMessage) public view returns (bool) {
        return StringUtils.equal(messages[user], expectedMessage);
    }
}
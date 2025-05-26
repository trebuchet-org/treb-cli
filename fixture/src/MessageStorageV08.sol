// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./StringUtilsV2.sol";

contract MessageStorageV08 {
    using StringUtilsV2 for string;
    
    struct MessageInfo {
        string content;
        uint256 length;
        uint256 timestamp;
    }
    
    mapping(address => MessageInfo) private userMessages;
    mapping(address => string[]) private messageHistory;
    
    event MessageStored(address indexed user, string message, uint256 length, uint256 timestamp);
    event MessagesCompared(address indexed user1, address indexed user2, bool areEqual);
    
    // Store a message with timestamp
    function storeMessage(string calldata message) external {
        uint256 msgLength = StringUtilsV2.length(message);
        
        // Store in history before updating
        if (bytes(userMessages[msg.sender].content).length > 0) {
            messageHistory[msg.sender].push(userMessages[msg.sender].content);
        }
        
        userMessages[msg.sender] = MessageInfo({
            content: message,
            length: msgLength,
            timestamp: block.timestamp
        });
        
        emit MessageStored(msg.sender, message, msgLength, block.timestamp);
    }
    
    // Get message info for a user
    function getMessageInfo(address user) external view returns (MessageInfo memory) {
        return userMessages[user];
    }
    
    // Get message history for a user
    function getMessageHistory(address user) external view returns (string[] memory) {
        return messageHistory[user];
    }
    
    // Combine messages from two users
    function combineMessages(address user1, address user2) external view returns (string memory) {
        string memory message1 = userMessages[user1].content;
        string memory message2 = userMessages[user2].content;
        return StringUtilsV2.concat(message1, message2);
    }
    
    // Store a formatted message
    function storeFormattedMessage(string calldata prefix, string calldata message, string calldata suffix) external {
        string memory formatted = StringUtilsV2.concat(prefix, message);
        formatted = StringUtilsV2.concat(formatted, suffix);
        
        uint256 msgLength = StringUtilsV2.length(formatted);
        
        userMessages[msg.sender] = MessageInfo({
            content: formatted,
            length: msgLength,
            timestamp: block.timestamp
        });
        
        emit MessageStored(msg.sender, formatted, msgLength, block.timestamp);
    }
    
    // Compare messages between two users
    function compareMessages(address user1, address user2) external returns (bool) {
        bool areEqual = StringUtilsV2.equal(
            userMessages[user1].content,
            userMessages[user2].content
        );
        
        emit MessagesCompared(user1, user2, areEqual);
        return areEqual;
    }
    
    // Store uppercase message with validation
    function storeValidatedUppercaseMessage(string calldata message) external {
        require(bytes(message).length > 0, "Message cannot be empty");
        require(bytes(message).length <= 256, "Message too long");
        
        string memory upperMessage = StringUtilsV2.toUpperCase(message);
        uint256 msgLength = StringUtilsV2.length(upperMessage);
        
        userMessages[msg.sender] = MessageInfo({
            content: upperMessage,
            length: msgLength,
            timestamp: block.timestamp
        });
        
        emit MessageStored(msg.sender, upperMessage, msgLength, block.timestamp);
    }
}
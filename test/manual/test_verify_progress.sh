#!/bin/bash
# Test script to demonstrate verify command progress indicators

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
TREB_CLI_DIR="$SCRIPT_DIR/../.."
TREB_BIN="$TREB_CLI_DIR/bin/treb"

# Create a test project
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"

echo "Creating test project in $TEST_DIR"

# Initialize project
forge init test-project --no-git
cd test-project
"$TREB_BIN" init test-project

# Create contracts
cat > src/Counter.sol << 'EOF'
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Counter {
    uint256 public count;
    
    function increment() external {
        count++;
    }
}
EOF

cat > src/Token.sol << 'EOF'
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Token {
    string public name = "Test Token";
    string public symbol = "TST";
    uint8 public decimals = 18;
    uint256 public totalSupply = 1000000 * 10**18;
    
    mapping(address => uint256) public balanceOf;
    
    constructor() {
        balanceOf[msg.sender] = totalSupply;
    }
}
EOF

# Start anvil on custom port
echo "Starting local test chain..."
anvil --port 8547 > /dev/null 2>&1 &
ANVIL_PID=$!
sleep 2

# Configure network
"$TREB_BIN" config set network test-network --rpc-url http://localhost:8547 --chain-id 31337

# Deploy contracts
echo ""
echo "Deploying contracts..."
"$TREB_BIN" gen deploy Counter
"$TREB_BIN" run script/deploy/DeployCounter.s.sol --network test-network

"$TREB_BIN" gen deploy Token
"$TREB_BIN" run script/deploy/DeployToken.s.sol --network test-network

echo ""
echo "==========================================="
echo "Testing verify command with progress indicators"
echo "==========================================="
echo ""

# Test single contract verification
echo "1. Single contract verification:"
echo "Command: $TREB_BIN verify Counter --network test-network"
echo ""
"$TREB_BIN" verify Counter --network test-network || true

echo ""
echo "2. Verify all contracts:"
echo "Command: $TREB_BIN verify --all"
echo ""
"$TREB_BIN" verify --all || true

echo ""
echo "3. Force re-verification:"
echo "Command: $TREB_BIN verify --all --force"
echo ""
"$TREB_BIN" verify --all --force || true

# Cleanup
kill $ANVIL_PID 2>/dev/null || true
echo ""
echo "Test directory preserved at: $TEST_DIR"
echo "To clean up: rm -rf $TEST_DIR"
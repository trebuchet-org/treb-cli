#!/bin/bash
# Manual test script to verify interactive selection in verify command

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

# Create a simple counter contract
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

# Start anvil on custom port
echo "Starting local test chain..."
anvil --port 8546 > /dev/null 2>&1 &
ANVIL_PID=$!
sleep 2

# Configure network
"$TREB_BIN" config set network test-network --rpc-url http://localhost:8546 --chain-id 31337

# Deploy Counter multiple times with different labels
echo "Deploying Counter with different labels..."
"$TREB_BIN" gen deploy Counter
"$TREB_BIN" run script/deploy/DeployCounter.s.sol --network test-network --env LABEL=v1
"$TREB_BIN" run script/deploy/DeployCounter.s.sol --network test-network --env LABEL=v2
"$TREB_BIN" run script/deploy/DeployCounter.s.sol --network test-network --namespace staging --env LABEL=v1

echo ""
echo "Deployments created:"
"$TREB_BIN" list

echo ""
echo "Now testing verify command with ambiguous reference 'Counter'..."
echo "This should trigger interactive selection since there are multiple Counter deployments"
echo ""
echo "Run: $TREB_BIN verify Counter --network test-network"
echo ""
echo "Expected: Interactive prompt to select between:"
echo "  - default/31337/Counter:v1"
echo "  - default/31337/Counter:v2" 
echo "  - staging/31337/Counter:v1"

# Cleanup
kill $ANVIL_PID 2>/dev/null || true
# Keep test directory for manual inspection
echo ""
echo "Test directory preserved at: $TEST_DIR"
echo "To clean up: rm -rf $TEST_DIR"
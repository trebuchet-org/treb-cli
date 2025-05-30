#!/bin/bash

# Test script to demonstrate ContractDeployed events capture
# This shows the exact events you should see in the JSON output

echo "🧪 Testing ContractDeployed Events Capture"
echo "=========================================="

# Set up working SENDER_CONFIGS for a single private key sender
export SENDER_CONFIGS="0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000080000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb922669bff76f73df2dfba00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000b746573742d73656e6465720000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

echo "📋 Running deployment script with event capture..."
echo "   This will deploy multiple contracts and capture all events"
echo ""

# Run the test script with JSON output to capture events
OUTPUT_FILE="deployment-events-test.json"
forge script script/TestContractDeployedEvents.s.sol \
    --fork-url http://localhost:8545 \
    --json -vvvv > "$OUTPUT_FILE" 2>&1

echo "✅ Script execution completed"
echo "📊 Raw output saved to: $OUTPUT_FILE"
echo ""

# Check if jq is available
if command -v jq >/dev/null 2>&1; then
    echo "🔍 Analyzing events with jq..."
    echo ""
    
    # Show all events excluding console.log
    echo "📋 All events (excluding console.log):"
    jq -r '.raw_logs[] | select(.address != "0x000000000000000000636f6e736f6c652e6c6f67") | 
        "Event: " + .topics[0] + " from " + .address' "$OUTPUT_FILE" 2>/dev/null || echo "No events found or JSON parsing failed"
    echo ""
    
    # Look specifically for ContractDeployed events (from our earlier test)
    echo "🎯 ContractDeployed events (topic: 0x8f4dc992827388d6f9546363611ad0a09a82022d386aa9110c021a68bbc2a3e9):"
    DEPLOYED_COUNT=$(jq '[.raw_logs[] | select(.topics[0] == "0x8f4dc992827388d6f9546363611ad0a09a82022d386aa9110c021a68bbc2a3e9")] | length' "$OUTPUT_FILE" 2>/dev/null || echo "0")
    echo "Found $DEPLOYED_COUNT ContractDeployed events:"
    jq -r '.raw_logs[] | select(.topics[0] == "0x8f4dc992827388d6f9546363611ad0a09a82022d386aa9110c021a68bbc2a3e9") | 
        "✓ ContractDeployed - Deployer: " + .topics[1] + " → Contract: " + .topics[2]' "$OUTPUT_FILE" 2>/dev/null || echo "No ContractDeployed events found"
    echo ""
    
    # Look for DeployingContract events 
    echo "🚀 DeployingContract events (if any):"
    # Note: We'd need to calculate the keccak256 hash for DeployingContract(string,string,bytes32)
    # For now, just show any events from the script contract address
    SCRIPT_ADDRESS=$(jq -r '.address // empty' "$OUTPUT_FILE" 2>/dev/null)
    if [ ! -z "$SCRIPT_ADDRESS" ]; then
        echo "Script contract address: $SCRIPT_ADDRESS"
        jq -r ".raw_logs[]? | select(.address == \"$SCRIPT_ADDRESS\") | 
            \"Event from script: \" + .topics[0]" "$OUTPUT_FILE" 2>/dev/null || echo "No events from script address"
    fi
    echo ""
    
    # Show success status
    echo "📈 Script execution status:"
    jq -r 'if .success then "✅ SUCCESS" else "❌ FAILED" end' "$OUTPUT_FILE" 2>/dev/null || echo "Status unknown"
    
else
    echo "⚠️  jq not found - install jq to analyze events automatically"
    echo "   You can manually inspect the JSON in: $OUTPUT_FILE"
    echo ""
    echo "💡 Look for these key things in the JSON:"
    echo "   - .raw_logs[] array contains all events"
    echo "   - ContractDeployed topic: 0x8f4dc992827388d6f9546363611ad0a09a82022d386aa9110c021a68bbc2a3e9"
    echo "   - Events from CreateX factory: 0xba5ed099633d3b313e4d5f7bdc1305d3c28ba5ed"
    echo "   - Script contract events (your deployed contracts)"
fi

echo ""
echo "🎯 What to look for:"
echo "   1. ContractDeployed events with topic 0x8f4dc992..."
echo "   2. Events from CreateX factory (0xba5ed099...)"
echo "   3. Multiple deployment events for Counter, TestCounter, and Counter v2"
echo "   4. Deployer address in topics[1], deployed contract address in topics[2]"
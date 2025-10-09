#!/bin/bash
# Simple test to check verify command progress

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
TREB_BIN="$SCRIPT_DIR/../../bin/treb"

# Go to test data directory
cd "$SCRIPT_DIR/../testdata/project"

echo "Testing verify command progress indicators..."
echo ""

# Try verify on non-existent deployment to see error message
echo "1. Testing verify with no deployments:"
"$TREB_BIN" verify --all 2>&1 || true

echo ""
echo "2. Testing verify with specific contract that doesn't exist:"
"$TREB_BIN" verify Counter 2>&1 || true

echo ""
echo "Done!"
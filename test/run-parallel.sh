#!/bin/bash

# Script to run compatibility tests in parallel mode

set -e

echo "🚀 Running compatibility tests in parallel mode..."
echo ""

# Set environment variables for parallel execution
export TREB_TEST_PARALLEL=true
export TREB_TEST_POOL_SIZE=${TREB_TEST_POOL_SIZE:-10}

# Optional: Set test pattern from first argument
TEST_PATTERN=${1:-""}

echo "Configuration:"
echo "  Pool size: $TREB_TEST_POOL_SIZE"
echo "  Test pattern: ${TEST_PATTERN:-all}"
echo ""

# Change to test directory
cd "$(dirname "$0")"

# Run tests
if [ -z "$TEST_PATTERN" ]; then
    echo "Running all compatibility tests..."
    go test ./compatibility -v -parallel=$TREB_TEST_POOL_SIZE
else
    echo "Running tests matching: $TEST_PATTERN"
    go test ./compatibility -v -parallel=$TREB_TEST_POOL_SIZE -run "$TEST_PATTERN"
fi
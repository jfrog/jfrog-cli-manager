#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[HEADER]${NC} $1"
}

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

print_header "Starting jfcm E2E Test Suite"
print_status "Project root: $PROJECT_ROOT"
print_status "Current directory: $(pwd)"
print_status "Platform: $(uname -s)"
print_status "Architecture: $(uname -m)"

# Change to project root
cd "$PROJECT_ROOT"

# Verify jfcm_PATH is set
if [ -z "$jfcm_PATH" ]; then
    print_error "jfcm_PATH environment variable is not set"
    exit 1
fi

print_status "jfcm_PATH: $jfcm_PATH"

# Build jfcm if it doesn't exist
if [ ! -f "$jfcm_PATH" ]; then
    print_status "Building jfcm..."
    if ! go build -o "$jfcm_PATH" .; then
        print_error "Failed to build jfcm"
        exit 1
    fi
fi

# Make sure the binary is executable
chmod +x "$jfcm_PATH"

# Verify the binary works
print_status "Testing jfcm binary..."
if ! "$jfcm_PATH" --help > /dev/null 2>&1; then
    print_error "jfcm binary is not working correctly"
    exit 1
fi

print_status "jfcm binary is ready for testing"

# Set test timeout
TEST_TIMEOUT="15m"
if [ -n "$TEST_TIMEOUT_OVERRIDE" ]; then
    TEST_TIMEOUT="$TEST_TIMEOUT_OVERRIDE"
fi

print_status "Test timeout: $TEST_TIMEOUT"

# Run specific test if TEST_FILTER is set
if [ -n "$TEST_FILTER" ]; then
    print_status "Running filtered tests: $TEST_FILTER"
    if ! go test -v -timeout "$TEST_TIMEOUT" ./tests/e2e/... -run "$TEST_FILTER"; then
        print_error "Filtered tests failed"
        exit 1
    fi
else
    # Run all tests
    print_status "Running all E2E tests..."
    if ! go test -v -timeout "$TEST_TIMEOUT" ./tests/e2e/...; then
        print_error "E2E tests failed"
        exit 1
    fi
fi

print_header "All E2E tests completed successfully! 🎉" 
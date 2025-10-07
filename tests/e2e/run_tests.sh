#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

print_status "Running jfcm E2E tests"
print_status "Project root: $PROJECT_ROOT"

# Change to project root
cd "$PROJECT_ROOT"

# Build jfcm
print_status "Building jfcm..."
if ! go build -o jfcm .; then
    print_error "Failed to build jfcm"
    exit 1
fi

# Set the jfcm_PATH environment variable
export jfcm_PATH="$PROJECT_ROOT/jfcm"

print_status "jfcm_PATH set to: $jfcm_PATH"

# Verify the binary exists and is executable
if [ ! -x "$jfcm_PATH" ]; then
    print_error "jfcm binary not found or not executable at $jfcm_PATH"
    exit 1
fi

print_status "jfcm binary is ready for testing"

# Run the tests
print_status "Running E2E tests..."

# Run tests with verbose output and coverage
if go test -v -timeout 10m ./tests/e2e/...; then
    print_status "All E2E tests passed! 🎉"
    exit 0
else
    print_error "Some E2E tests failed! ❌"
    exit 1
fi 
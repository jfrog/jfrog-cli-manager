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

print_status "Running jfvm E2E tests"
print_status "Project root: $PROJECT_ROOT"

# Change to project root
cd "$PROJECT_ROOT"

# Build jfvm
print_status "Building jfvm..."
if ! go build -o jfvm .; then
    print_error "Failed to build jfvm"
    exit 1
fi

# Set the JFVM_PATH environment variable
export JFVM_PATH="$PROJECT_ROOT/jfvm"

print_status "JFVM_PATH set to: $JFVM_PATH"

# Verify the binary exists and is executable
if [ ! -x "$JFVM_PATH" ]; then
    print_error "jfvm binary not found or not executable at $JFVM_PATH"
    exit 1
fi

print_status "jfvm binary is ready for testing"

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
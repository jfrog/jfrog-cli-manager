#!/bin/bash
set -eu

# Main build script for JFVM
# Usage: ./build.sh [executable_name] [version]

if [ $# -eq 0 ]; then
    exe_name="jfvm"
else
    exe_name="$1"
fi

# Get version information
if [ $# -ge 2 ]; then
    version="$2"
else
    # Try to get version from git tag
    version=$(git describe --tags --exact-match HEAD 2>/dev/null || echo "dev-$(date +%Y%m%d%H%M%S)")
fi

# Get build information
build_date=$(date -u '+%Y-%m-%d_%H:%M:%S')
git_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
ldflags="-w -extldflags \"-static\""
ldflags="$ldflags -X main.Version=$version"
ldflags="$ldflags -X main.BuildDate=$build_date"
ldflags="$ldflags -X main.GitCommit=$git_commit"

echo "Building $exe_name..."
echo "  Version: $version"
echo "  Build Date: $build_date"
echo "  Git Commit: $git_commit"
echo "  GOOS: ${GOOS:-$(go env GOOS)}"
echo "  GOARCH: ${GOARCH:-$(go env GOARCH)}"

# Ensure CGO is disabled for static compilation
export CGO_ENABLED=0

# Build the binary
go build -o "$exe_name" -ldflags "$ldflags" main.go

echo "The $exe_name executable was successfully created."

# Display binary information
if command -v file >/dev/null 2>&1; then
    echo "Binary info: $(file "$exe_name")"
fi

if command -v ls >/dev/null 2>&1; then
    echo "Binary size: $(ls -lh "$exe_name" | awk '{print $5}')"
fi



#!/bin/bash
# build-binary.sh - Build binary for specific platform/architecture
set -euo pipefail

VERSION="$1"
GOOS="$2"
GOARCH="$3"

BUILD_DATE="$(date -u '+%Y-%m-%d_%H:%M:%S')"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Validate variables are not empty
if [[ -z "$VERSION" || -z "$BUILD_DATE" ]]; then
  echo "❌ Error: Required build variables are empty"
  echo "VERSION='$VERSION'"
  echo "BUILD_DATE='$BUILD_DATE'"
  echo "GIT_COMMIT='$GIT_COMMIT'"
  exit 1
fi

OUT="dist/$GOOS-$GOARCH"
BIN_NAME="jfvm"
EXT=""
if [ "$GOOS" = "windows" ]; then 
  EXT=".exe"
fi

echo "🔧 Building for $GOOS/$GOARCH"
echo "  Version: $VERSION"
echo "  Build Date: $BUILD_DATE"
echo "  Git Commit: $GIT_COMMIT"
echo "  Output: $OUT/${BIN_NAME}${EXT}"

mkdir -p "$OUT"

# Build with comprehensive ldflags and error checking
GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
  go build \
  -trimpath \
  -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildDate=${BUILD_DATE}' -X 'main.GitCommit=${GIT_COMMIT}'" \
  -o "$OUT/${BIN_NAME}${EXT}" \
  . || {
    echo "❌ Build failed for $GOOS/$GOARCH"
    echo "Build environment:"
    go env
    exit 1
  }

# Verify the binary was created and is executable
if [[ ! -f "$OUT/${BIN_NAME}${EXT}" ]]; then
  echo "❌ Error: Binary was not created at $OUT/${BIN_NAME}${EXT}"
  ls -la "$OUT/" || echo "Output directory does not exist"
  exit 1
fi

# Get binary info
# Get binary size in a portable way
case "$(uname)" in
  Darwin)
    BINARY_SIZE=$(stat -f%z "$OUT/${BIN_NAME}${EXT}" 2>/dev/null || echo "unknown")
    ;;
  Linux)
    BINARY_SIZE=$(stat -c%s "$OUT/${BIN_NAME}${EXT}" 2>/dev/null || echo "unknown")
    ;;
  *)
    echo "⚠️  Warning: Unsupported platform for stat. Binary size may be unknown."
    BINARY_SIZE="unknown"
    ;;
esac
echo "✅ Binary created successfully"
echo "  Size: $BINARY_SIZE bytes"
echo "  Path: $OUT/${BIN_NAME}${EXT}"

# Test the binary can execute (version check) - only on Linux for compatibility
if [[ "$GOOS" == "linux" ]]; then
  echo "🧪 Testing binary execution..."
  "$OUT/${BIN_NAME}${EXT}" --version || {
    echo "⚠️  Warning: Binary version check failed, but continuing..."
  }
fi

# Output for next steps
echo "binary_path=$OUT/${BIN_NAME}${EXT}" >> "$GITHUB_OUTPUT"
echo "binary_size=$BINARY_SIZE" >> "$GITHUB_OUTPUT"

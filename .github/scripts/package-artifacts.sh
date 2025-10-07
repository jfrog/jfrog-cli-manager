#!/bin/bash
# package-artifacts.sh - Package binaries and generate checksums
set -euo pipefail

VERSION="$1"
GOOS="$2"
GOARCH="$3"
BINARY_PATH="$4"

OUT="dist/$GOOS-$GOARCH"
BIN_NAME="jfcm"
EXT=""
if [ "$GOOS" = "windows" ]; then 
  EXT=".exe"
fi

BASENAME="jfcm-${VERSION}-$GOOS-$GOARCH"

echo "📦 Packaging for $GOOS/$GOARCH"
echo "  Basename: $BASENAME"
echo "  Source: $BINARY_PATH"

# Verify source binary exists before packaging
if [[ ! -f "$BINARY_PATH" ]]; then
  echo "❌ Error: Source binary not found at $BINARY_PATH"
  ls -la "$OUT/" || echo "Output directory does not exist"
  exit 1
fi

# Create packages and checksums
if [ "$GOOS" = "windows" ]; then
  # Windows: ZIP format
  echo "  Creating ZIP archive..."
  zip -j "${BASENAME}.zip" "$BINARY_PATH" || {
    echo "❌ Error: Failed to create ZIP archive"
    exit 1
  }
  
  # Verify ZIP was created
  if [[ ! -f "${BASENAME}.zip" ]]; then
    echo "❌ Error: ZIP archive was not created"
    exit 1
  fi
  
  echo "  Generating checksums..."
  sha256sum "${BASENAME}.zip" | awk '{print $1"  "$2}' > "${BASENAME}.zip.sha256"
  
  # Standalone binary for direct download
  echo "  Creating standalone binary..."
  cp "$BINARY_PATH" "${BASENAME}${EXT}"
  sha256sum "${BASENAME}${EXT}" | awk '{print $1"  "$2}' > "${BASENAME}${EXT}.sha256"
  
  echo "✅ Windows packaging complete"
  
else
  # Unix: tar.gz format
  echo "  Creating tar.gz archive..."
  tar -C "$OUT" -czf "${BASENAME}.tar.gz" "${BIN_NAME}${EXT}" || {
    echo "❌ Error: Failed to create tar.gz archive"
    echo "Contents of $OUT:"
    ls -la "$OUT/"
    exit 1
  }
  
  # Verify tar.gz was created
  if [[ ! -f "${BASENAME}.tar.gz" ]]; then
    echo "❌ Error: tar.gz archive was not created"
    exit 1
  fi
  
  echo "  Generating checksums..."
  sha256sum "${BASENAME}.tar.gz" | awk '{print $1"  "$2}' > "${BASENAME}.tar.gz.sha256"
  
    # Standalone binary for direct download
    echo "  Creating standalone binary..."
    cp "$BINARY_PATH" "${BASENAME}"
    sha256sum "${BASENAME}" | awk '{print $1"  "$2}' > "${BASENAME}.sha256"
  
  echo "✅ Unix packaging complete"
fi

# Verify checksums
echo "🔒 Verifying checksums..."
if [ "$GOOS" = "windows" ]; then
  for file in "${BASENAME}.zip" "${BASENAME}${EXT}"; do
    if [[ -f "$file" && -f "$file.sha256" ]]; then
      echo "  Checking $file..."
      sha256sum -c "$file.sha256" || {
        echo "❌ Error: Checksum verification failed for $file"
        exit 1
      }
    else
      echo "❌ Error: Missing file or checksum: $file"
      exit 1
    fi
  done
else
  for file in "${BASENAME}.tar.gz" "${BASENAME}"; do
    if [[ -f "$file" && -f "$file.sha256" ]]; then
      echo "  Checking $file..."
      sha256sum -c "$file.sha256" || {
        echo "❌ Error: Checksum verification failed for $file"
        exit 1
      }
    else
      echo "❌ Error: Missing file or checksum: $file"
      exit 1
    fi
  done
fi

echo "✅ All checksums verified successfully"
echo "📋 Generated files:"
ls -la "${BASENAME}"* || echo "No files generated with basename ${BASENAME}"

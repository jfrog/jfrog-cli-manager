#!/bin/bash
# add-summary.sh - Add content to GitHub Actions Summary
set -euo pipefail

SUMMARY_TYPE="$1"
shift # Remove first argument, rest are data

case "$SUMMARY_TYPE" in
  "build")
    GOOS="$1"
    GOARCH="$2"
    BINARY_SIZE="$3"
    VERSION="$4"
    BUILD_DATE="$5"
    GIT_COMMIT="$6"
    
    {
      echo "## 🔧 Build Results - $GOOS/$GOARCH"
      echo ""
      echo "| Property | Value |"
      echo "|----------|-------|"
      echo "| **Platform** | $GOOS/$GOARCH |"
      echo "| **Binary Size** | $(numfmt --to=iec "$BINARY_SIZE" 2>/dev/null || echo "$BINARY_SIZE bytes") |"
      echo "| **Version** | $VERSION |"
      echo "| **Build Date** | $BUILD_DATE |"
      echo "| **Git Commit** | $GIT_COMMIT |"
      echo "| **Status** | ✅ Success |"
      echo ""
    } >> "$GITHUB_STEP_SUMMARY"
    ;;
    
  "release")
    VERSION="$1"
    TOTAL_FILES="$2"
    RELEASE_URL="$3"
    
    {
      echo "## 🎉 Release Published Successfully!"
      echo ""
      echo "| Property | Value |"
      echo "|----------|-------|"
      echo "| **Version** | [\`$VERSION\`]($RELEASE_URL) |"
      echo "| **Total Files** | $TOTAL_FILES |"
      echo "| **GitHub Release** | [$VERSION]($RELEASE_URL) |"
      echo ""
      echo "### 🔗 Quick Install Commands"
      echo ""
      echo '```bash'
      echo "# Homebrew (recommended)"
      echo "brew install jfrog/tap/jfvm"
      echo ""
      echo "# Direct download (macOS ARM64)"
      echo "curl -L $RELEASE_URL/jfvm-$VERSION-darwin-arm64-raw -o jfvm && chmod +x jfvm"
      echo '```'
      echo ""
    } >> "$GITHUB_STEP_SUMMARY"
    ;;
    
  "homebrew")
    VERSION="$1"
    FORMULA_SIZE="$2"
    
    {
      echo "## 🍺 Homebrew Formula Generated"
      echo ""
      echo "| Property | Value |"
      echo "|----------|-------|"
      echo "| **Formula Name** | \`jfvm.rb\` |"
      echo "| **Version** | \`$VERSION\` |"
      echo "| **Platforms** | macOS (Intel, ARM), Linux (Intel, ARM) |"
      echo "| **File Size** | $(numfmt --to=iec "$FORMULA_SIZE" 2>/dev/null || echo "$FORMULA_SIZE bytes") |"
      echo "| **Status** | ✅ Generated & Validated |"
      echo ""
    } >> "$GITHUB_STEP_SUMMARY"
    ;;
    
  *)
    echo "❌ Error: Unknown summary type: $SUMMARY_TYPE"
    echo "Supported types: build, release, homebrew"
    exit 1
    ;;
esac

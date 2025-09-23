#!/bin/bash
# validate-tag.sh - Validate and determine release tag
set -euo pipefail

# Determine the tag
if [[ "$GITHUB_EVENT_NAME" == "workflow_dispatch" ]]; then
  TAG="$INPUT_TAG"
  echo "Using manually provided tag: $TAG"
else
  TAG="${GITHUB_REF#refs/tags/}"
  echo "Using pushed tag: $TAG"
fi

# Validate tag format (semantic versioning with v prefix)
SEMVER_REGEX="^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+(\.[0-9A-Za-z]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$"
if [[ ! "$TAG" =~ $SEMVER_REGEX ]]; then
  echo "❌ Error: Invalid tag format '$TAG'. Expected format: v1.2.3 or v1.2.3-beta.1"
  echo "Examples of valid tags:"
  echo "  - v1.0.0"
  echo "  - v2.1.3"
  echo "  - v1.0.0-beta.1"
  echo "  - v2.0.0-rc.1"
  exit 1
fi

echo "✅ Tag format is valid: $TAG"
echo "tag=$TAG" >> "$GITHUB_OUTPUT"

# Add to GitHub Actions Summary
{
  echo "## 🏷️ Release Validation"
  echo ""
  echo "| Field | Value |"
  echo "|-------|-------|"
  echo "| **Tag** | \`$TAG\` |"
  echo "| **Version** | \`${TAG#v}\` |"
  echo "| **Trigger** | $GITHUB_EVENT_NAME |"
  echo "| **Status** | ✅ Valid |"
} >> "$GITHUB_STEP_SUMMARY"

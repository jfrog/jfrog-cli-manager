#!/bin/bash
# generate-homebrew-formula.sh - Generate Homebrew formula
set -euo pipefail

TAG="$1"
REPO="$2"
DARWIN_AMD="$3"
DARWIN_ARM="$4"
LINUX_AMD="$5"
LINUX_ARM="$6"

VERSION_NO_V="${TAG#v}"

echo "🍺 Generating Homebrew formula for jfcm $TAG"
echo "  Repository: $REPO"
echo "  Version: $VERSION_NO_V"

# Validate all required variables
for var_name in "TAG" "VERSION_NO_V" "REPO" "DARWIN_AMD" "DARWIN_ARM" "LINUX_AMD" "LINUX_ARM"; do
  var_value="${!var_name}"
  if [[ -z "$var_value" ]]; then
    echo "❌ Error: Required variable $var_name is empty"
    exit 1
  fi
done

# Validate checksums are valid SHA256 hashes
for sha_var in "$DARWIN_AMD" "$DARWIN_ARM" "$LINUX_AMD" "$LINUX_ARM"; do
  if [[ ! "$sha_var" =~ ^[a-f0-9]{64}$ ]]; then
    echo "❌ Error: Invalid SHA256 format: $sha_var"
    exit 1
  fi
done

# Generate the Homebrew formula
cat > jfcm.rb << EOF
class jfcm < Formula
  desc "Manage multiple versions of JFrog CLI"
  homepage "https://github.com/${REPO}"
  version "${VERSION_NO_V}"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/${REPO}/releases/download/${TAG}/jfcm-${TAG}-darwin-arm64.tar.gz"
      sha256 "${DARWIN_ARM}"
    end
    on_intel do
      url "https://github.com/${REPO}/releases/download/${TAG}/jfcm-${TAG}-darwin-amd64.tar.gz"
      sha256 "${DARWIN_AMD}"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/${REPO}/releases/download/${TAG}/jfcm-${TAG}-linux-arm64.tar.gz"
      sha256 "${LINUX_ARM}"
    end
    on_intel do
      url "https://github.com/${REPO}/releases/download/${TAG}/jfcm-${TAG}-linux-amd64.tar.gz"
      sha256 "${LINUX_AMD}"
    end
  end

  def install
    bin.install "jfcm"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/jfcm --version")
    assert_match "Manage multiple versions of JFrog CLI", shell_output("#{bin}/jfcm --help")
  end
end
EOF

echo "✅ Homebrew formula generated successfully"

# Validate the generated formula
echo "🔍 Validating generated formula..."

# Check file was created and has content
if [[ ! -f "jfcm.rb" ]]; then
  echo "❌ Error: Formula file was not created"
  exit 1
fi

# Check file size (should be reasonable)
FORMULA_SIZE=$(stat -c%s "jfcm.rb" 2>/dev/null || stat -f%z "jfcm.rb" 2>/dev/null)
if [[ $FORMULA_SIZE -lt 100 ]]; then
  echo "❌ Error: Formula file seems too small ($FORMULA_SIZE bytes)"
  cat jfcm.rb
  exit 1
fi

# Verify all checksums are present in the formula
for sha in "$DARWIN_AMD" "$DARWIN_ARM" "$LINUX_AMD" "$LINUX_ARM"; do
  if ! grep -q "$sha" jfcm.rb; then
    echo "❌ Error: Checksum $sha not found in formula"
    exit 1
  fi
done

echo "✅ Formula validation passed"
echo "formula_size=$FORMULA_SIZE" >> "$GITHUB_OUTPUT"

#!/bin/bash
set -e

# macOS binary signing and notarization script
# This script signs and notarizes macOS binaries using Apple's tools

BINARY_PATH=""
OUTPUT_PATH=""
BUNDLE_ID="com.jfrog.jfvm"
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--input)
            BINARY_PATH="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_PATH="$2"
            shift 2
            ;;
        -b|--bundle-id)
            BUNDLE_ID="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 -i <input_binary> -o <output_binary> [-b <bundle_id>] [-v|--verbose]"
            echo ""
            echo "Sign and notarize macOS binaries"
            echo ""
            echo "Options:"
            echo "  -i, --input      Input binary file"
            echo "  -o, --output     Output signed binary file"
            echo "  -b, --bundle-id  Bundle identifier (default: com.jfrog.jfvm)"
            echo "  -v, --verbose    Enable verbose output"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  APPLE_TEAM_ID              Apple Developer Team ID"
            echo "  APPLE_ACCOUNT_ID            Apple ID for notarization"
            echo "  APPLE_APP_SPECIFIC_PASSWORD App-specific password for notarization"
            echo "  SIGNING_IDENTITY            Code signing identity (optional)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Validate required arguments
if [ -z "$BINARY_PATH" ] || [ -z "$OUTPUT_PATH" ]; then
    echo "Error: Both input and output paths must be specified"
    echo "Usage: $0 -i <input_binary> -o <output_binary>"
    exit 1
fi

# Check if input file exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Input binary '$BINARY_PATH' does not exist"
    exit 1
fi

# Verbose output function
log() {
    if [ "$VERBOSE" = true ]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
    fi
}

log "Starting macOS binary signing and notarization..."
log "Input binary: $BINARY_PATH"
log "Output binary: $OUTPUT_PATH"
log "Bundle ID: $BUNDLE_ID"

# Check for required environment variables
if [ -z "$APPLE_TEAM_ID" ]; then
    echo "Warning: APPLE_TEAM_ID environment variable not set"
    echo "Copying unsigned binary to output (signing disabled for this build)"
    cp "$BINARY_PATH" "$OUTPUT_PATH"
    log "Binary copied without signing"
    exit 0
fi

log "Apple Team ID: $APPLE_TEAM_ID"

# Determine signing identity
if [ -z "$SIGNING_IDENTITY" ]; then
    SIGNING_IDENTITY="Developer ID Application: JFrog Ltd ($APPLE_TEAM_ID)"
fi

log "Signing identity: $SIGNING_IDENTITY"

# Create output directory if it doesn't exist
OUTPUT_DIR=$(dirname "$OUTPUT_PATH")
mkdir -p "$OUTPUT_DIR"

# Copy binary to output location first
cp "$BINARY_PATH" "$OUTPUT_PATH"

# Step 1: Code signing
log "Step 1: Code signing the binary..."

codesign_cmd="codesign"
codesign_cmd="$codesign_cmd --sign '$SIGNING_IDENTITY'"
codesign_cmd="$codesign_cmd --timestamp"
codesign_cmd="$codesign_cmd --deep"
codesign_cmd="$codesign_cmd --options runtime"
codesign_cmd="$codesign_cmd --force"
codesign_cmd="$codesign_cmd --identifier '$BUNDLE_ID'"

if [ "$VERBOSE" = true ]; then
    codesign_cmd="$codesign_cmd --verbose"
fi

codesign_cmd="$codesign_cmd '$OUTPUT_PATH'"

log "Executing: $codesign_cmd"

if eval "$codesign_cmd"; then
    log "Code signing completed successfully"
else
    echo "Error: Code signing failed"
    exit 1
fi

# Step 2: Verify code signature
log "Step 2: Verifying code signature..."

if codesign --verify --deep --strict --verbose=2 "$OUTPUT_PATH"; then
    log "Code signature verification passed"
else
    echo "Error: Code signature verification failed"
    exit 1
fi

# Step 3: Check for notarization requirements
if [ -z "$APPLE_ACCOUNT_ID" ] || [ -z "$APPLE_APP_SPECIFIC_PASSWORD" ]; then
    echo "Warning: Notarization credentials not provided"
    echo "Skipping notarization step"
    log "Binary signed but not notarized"
    exit 0
fi

log "Apple Account ID: $APPLE_ACCOUNT_ID"

# Step 4: Create zip file for notarization
log "Step 4: Creating zip file for notarization..."

temp_dir=$(mktemp -d)
temp_zip="$temp_dir/jfvm-notarization.zip"

# Create zip file
(cd "$(dirname "$OUTPUT_PATH")" && zip -r "$temp_zip" "$(basename "$OUTPUT_PATH")")

log "Created zip file: $temp_zip"

# Step 5: Submit for notarization
log "Step 5: Submitting for notarization..."

notarize_cmd="xcrun notarytool submit"
notarize_cmd="$notarize_cmd '$temp_zip'"
notarize_cmd="$notarize_cmd --apple-id '$APPLE_ACCOUNT_ID'"
notarize_cmd="$notarize_cmd --team-id '$APPLE_TEAM_ID'"
notarize_cmd="$notarize_cmd --password '$APPLE_APP_SPECIFIC_PASSWORD'"
notarize_cmd="$notarize_cmd --wait"

if [ "$VERBOSE" = true ]; then
    notarize_cmd="$notarize_cmd --verbose"
fi

log "Executing notarization submission..."

if eval "$notarize_cmd"; then
    log "Notarization completed successfully"
else
    echo "Error: Notarization failed"
    rm -rf "$temp_dir"
    exit 1
fi

# Step 6: Staple the notarization
log "Step 6: Stapling notarization..."

if xcrun stapler staple "$OUTPUT_PATH"; then
    log "Notarization stapling completed successfully"
else
    echo "Warning: Notarization stapling failed (binary is still notarized)"
fi

# Cleanup
rm -rf "$temp_dir"

# Final verification
log "Final verification..."

if spctl --assess --type execute --verbose "$OUTPUT_PATH"; then
    log "Final verification passed"
else
    echo "Warning: Final verification failed"
fi

# Display file information
if command -v ls >/dev/null 2>&1; then
    log "Signed and notarized binary size: $(ls -lh "$OUTPUT_PATH" | awk '{print $5}')"
fi

echo "Successfully signed and notarized: $BINARY_PATH -> $OUTPUT_PATH"
log "macOS binary signing and notarization completed"



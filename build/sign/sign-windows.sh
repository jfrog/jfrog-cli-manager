#!/bin/bash
set -e

# Windows binary signing script
# This script signs Windows executables using osslsigncode

INPUT_FILE=""
OUTPUT_FILE=""
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -in|--input)
            INPUT_FILE="$2"
            shift 2
            ;;
        -out|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 -in <input_file> -out <output_file> [-v|--verbose]"
            echo ""
            echo "Sign Windows executables using osslsigncode"
            echo ""
            echo "Options:"
            echo "  -in, --input     Input executable file"
            echo "  -out, --output   Output signed executable file"
            echo "  -v, --verbose    Enable verbose output"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  CERT_FILE        Path to the certificate file (.p12 or .pfx)"
            echo "  CERT_PASSWORD    Password for the certificate"
            echo "  TIMESTAMP_URL    Timestamp server URL (optional)"
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
if [ -z "$INPUT_FILE" ] || [ -z "$OUTPUT_FILE" ]; then
    echo "Error: Both input and output files must be specified"
    echo "Usage: $0 -in <input_file> -out <output_file>"
    exit 1
fi

# Check if input file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: Input file '$INPUT_FILE' does not exist"
    exit 1
fi

# Verbose output function
log() {
    if [ "$VERBOSE" = true ]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
    fi
}

log "Starting Windows binary signing process..."
log "Input file: $INPUT_FILE"
log "Output file: $OUTPUT_FILE"

# Check for certificate file
if [ -z "$CERT_FILE" ]; then
    echo "Warning: CERT_FILE environment variable not set"
    echo "Copying unsigned binary to output (signing disabled for this build)"
    cp "$INPUT_FILE" "$OUTPUT_FILE"
    log "Binary copied without signing"
    exit 0
fi

if [ ! -f "$CERT_FILE" ]; then
    echo "Error: Certificate file '$CERT_FILE' does not exist"
    exit 1
fi

log "Certificate file: $CERT_FILE"

# Default timestamp URL if not provided
if [ -z "$TIMESTAMP_URL" ]; then
    TIMESTAMP_URL="http://timestamp.digicert.com"
fi

log "Timestamp URL: $TIMESTAMP_URL"

# Prepare signing command
SIGN_CMD="osslsigncode sign"
SIGN_CMD="$SIGN_CMD -pkcs12 '$CERT_FILE'"

if [ -n "$CERT_PASSWORD" ]; then
    SIGN_CMD="$SIGN_CMD -pass '$CERT_PASSWORD'"
    log "Using certificate password"
else
    log "No certificate password provided"
fi

SIGN_CMD="$SIGN_CMD -t '$TIMESTAMP_URL'"
SIGN_CMD="$SIGN_CMD -in '$INPUT_FILE'"
SIGN_CMD="$SIGN_CMD -out '$OUTPUT_FILE'"

# Add description and URL for better identification
SIGN_CMD="$SIGN_CMD -n 'JFVM - JFrog CLI Version Manager'"
SIGN_CMD="$SIGN_CMD -i 'https://github.com/jfrog/jfrog-cli-vm'"

log "Executing signing command..."

# Execute the signing command
if eval "$SIGN_CMD"; then
    log "Signing completed successfully"
    
    # Verify the signature
    if command -v osslsigncode >/dev/null 2>&1; then
        log "Verifying signature..."
        if osslsigncode verify -in "$OUTPUT_FILE"; then
            log "Signature verification passed"
        else
            echo "Warning: Signature verification failed"
        fi
    fi
    
    # Display file information
    if command -v ls >/dev/null 2>&1; then
        log "Signed binary size: $(ls -lh "$OUTPUT_FILE" | awk '{print $5}')"
    fi
    
    echo "Successfully signed: $INPUT_FILE -> $OUTPUT_FILE"
else
    echo "Error: Signing failed"
    exit 1
fi

log "Windows binary signing process completed"



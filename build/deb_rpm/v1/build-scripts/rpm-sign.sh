#!/bin/bash
set -e

# RPM package signing script
# This script signs RPM packages using GPG

RPM_FILE=""
GPG_KEY_FILE=""
GPG_PASSPHRASE=""
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--file)
            RPM_FILE="$2"
            shift 2
            ;;
        -k|--key)
            GPG_KEY_FILE="$2"
            shift 2
            ;;
        -p|--passphrase)
            GPG_PASSPHRASE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 -f <rpm_file> [-k <gpg_key_file>] [-p <passphrase>] [-v|--verbose]"
            echo ""
            echo "Sign RPM packages using GPG"
            echo ""
            echo "Options:"
            echo "  -f, --file       RPM file to sign"
            echo "  -k, --key        GPG key file to import"
            echo "  -p, --passphrase GPG key passphrase"
            echo "  -v, --verbose    Enable verbose output"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  RPM_GPG_KEY_FILE      Path to GPG key file"
            echo "  RPM_SIGN_PASSPHRASE   GPG key passphrase"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Use environment variables if not provided as arguments
if [ -z "$GPG_KEY_FILE" ] && [ -n "$RPM_GPG_KEY_FILE" ]; then
    GPG_KEY_FILE="$RPM_GPG_KEY_FILE"
fi

if [ -z "$GPG_PASSPHRASE" ] && [ -n "$RPM_SIGN_PASSPHRASE" ]; then
    GPG_PASSPHRASE="$RPM_SIGN_PASSPHRASE"
fi

# Validate required arguments
if [ -z "$RPM_FILE" ]; then
    echo "Error: RPM file must be specified"
    echo "Usage: $0 -f <rpm_file>"
    exit 1
fi

# Check if RPM file exists
if [ ! -f "$RPM_FILE" ]; then
    echo "Error: RPM file '$RPM_FILE' does not exist"
    exit 1
fi

# Verbose output function
log() {
    if [ "$VERBOSE" = true ]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
    fi
}

log "Starting RPM package signing..."
log "RPM file: $RPM_FILE"

# Check for GPG key file
if [ -z "$GPG_KEY_FILE" ]; then
    echo "Warning: No GPG key file provided"
    echo "Skipping RPM signing (package will remain unsigned)"
    log "RPM signing skipped"
    exit 0
fi

if [ ! -f "$GPG_KEY_FILE" ]; then
    echo "Error: GPG key file '$GPG_KEY_FILE' does not exist"
    exit 1
fi

log "GPG key file: $GPG_KEY_FILE"

# Check if rpm and gpg commands are available
if ! command -v rpm >/dev/null 2>&1; then
    echo "Error: rpm command not found"
    exit 1
fi

if ! command -v gpg >/dev/null 2>&1; then
    echo "Error: gpg command not found"
    exit 1
fi

# Step 1: Import GPG key
log "Step 1: Importing GPG key..."

gpg_import_cmd="gpg --batch --import"

if [ "$VERBOSE" = true ]; then
    gpg_import_cmd="$gpg_import_cmd --verbose"
else
    gpg_import_cmd="$gpg_import_cmd --quiet"
fi

gpg_import_cmd="$gpg_import_cmd '$GPG_KEY_FILE'"

if eval "$gpg_import_cmd"; then
    log "GPG key imported successfully"
else
    echo "Error: Failed to import GPG key"
    exit 1
fi

# Step 2: Get key ID
log "Step 2: Getting GPG key ID..."

KEY_ID=$(gpg --list-secret-keys --with-colons | grep '^sec:' | head -1 | cut -d: -f5)

if [ -z "$KEY_ID" ]; then
    echo "Error: Could not determine GPG key ID"
    exit 1
fi

log "GPG key ID: $KEY_ID"

# Step 3: Configure RPM macros for signing
log "Step 3: Configuring RPM signing macros..."

RPM_MACROS_FILE="$HOME/.rpmmacros"

# Backup existing macros file if it exists
if [ -f "$RPM_MACROS_FILE" ]; then
    cp "$RPM_MACROS_FILE" "$RPM_MACROS_FILE.backup.$(date +%s)"
    log "Backed up existing .rpmmacros file"
fi

# Create or update .rpmmacros file
cat > "$RPM_MACROS_FILE" << EOF
%_gpg_name $KEY_ID
%_signature gpg
%_gpg_path $HOME/.gnupg
%_gpgbin /usr/bin/gpg
%__gpg_sign_cmd %{__gpg} \\
    gpg --force-v3-sigs --batch --verbose --no-armor \\
    --passphrase-fd 3 --no-secmem-warning \\
    -u "%{_gpg_name}" -sbo %{__signature_filename} \\
    --digest-algo sha256 %{__plaintext_filename}
EOF

log "RPM signing macros configured"

# Step 4: Sign the RPM
log "Step 4: Signing RPM package..."

if [ -n "$GPG_PASSPHRASE" ]; then
    # Sign with passphrase
    echo "$GPG_PASSPHRASE" | rpm --addsign "$RPM_FILE"
else
    # Sign without passphrase (assumes key has no passphrase)
    rpm --addsign "$RPM_FILE"
fi

if [ $? -eq 0 ]; then
    log "RPM signing completed successfully"
else
    echo "Error: RPM signing failed"
    exit 1
fi

# Step 5: Verify the signature
log "Step 5: Verifying RPM signature..."

if rpm --checksig "$RPM_FILE"; then
    log "RPM signature verification passed"
else
    echo "Warning: RPM signature verification failed"
fi

# Display package information
if command -v rpm >/dev/null 2>&1; then
    log "Signed RPM info:"
    rpm -qip "$RPM_FILE" | head -10
fi

echo "Successfully signed RPM: $RPM_FILE"
log "RPM package signing completed"

# Cleanup: restore original .rpmmacros if it existed
if [ -f "$RPM_MACROS_FILE.backup."* ]; then
    LATEST_BACKUP=$(ls -t "$RPM_MACROS_FILE.backup."* | head -1)
    mv "$LATEST_BACKUP" "$RPM_MACROS_FILE"
    log "Restored original .rpmmacros file"
fi



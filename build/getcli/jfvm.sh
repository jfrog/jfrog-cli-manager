#!/bin/bash

# JFVM Download Script
# This script downloads and installs JFVM on Unix-like systems
# Similar to the JFrog CLI getcli script pattern

set -e

# Configuration
JFVM_VERSION="${JFVM_VERSION:-latest}"
INSTALL_DIR="${JFVM_INSTALL_DIR:-/usr/local/bin}"
JFVM_HOME="${JFVM_HOME:-$HOME/.jfvm}"
FORCE_INSTALL="${JFVM_FORCE:-false}"
SILENT="${JFVM_SILENT:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Disable colors if not in terminal
if [ ! -t 1 ]; then
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

print_banner() {
    if [ "$SILENT" != "true" ]; then
        echo -e "${BLUE}"
        echo "╔═══════════════════════════════════════════════════════════════════╗"
        echo "║                     JFVM Download Script                         ║"
        echo "║                                                                   ║"
        echo "║  This script downloads and installs JFVM (JFrog CLI Version      ║"
        echo "║  Manager) on your system.                                        ║"
        echo "╚═══════════════════════════════════════════════════════════════════╝"
        echo -e "${NC}\n"
    fi
}

log() {
    if [ "$SILENT" != "true" ]; then
        echo -e "$1"
    fi
}

error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

warning() {
    if [ "$SILENT" != "true" ]; then
        echo -e "${YELLOW}Warning: $1${NC}" >&2
    fi
}

success() {
    if [ "$SILENT" != "true" ]; then
        echo -e "${GREEN}$1${NC}"
    fi
}

info() {
    if [ "$SILENT" != "true" ]; then
        echo -e "${BLUE}$1${NC}"
    fi
}

detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$os" in
        linux*)
            case "$arch" in
                x86_64) echo "linux-amd64" ;;
                aarch64|arm64) echo "linux-arm64" ;;
                armv7l) echo "linux-arm" ;;
                i*86) echo "linux-386" ;;
                s390x) echo "linux-s390x" ;;
                ppc64le) echo "linux-ppc64le" ;;
                ppc64) echo "linux-ppc64" ;;
                *) echo "unsupported" ;;
            esac
            ;;
        darwin*)
            case "$arch" in
                x86_64) echo "mac-amd64" ;;
                arm64) echo "mac-arm64" ;;
                *) echo "unsupported" ;;
            esac
            ;;
        *)
            echo "unsupported"
            ;;
    esac
}

check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
        missing_deps+=("curl or wget")
    fi
    
    if ! command -v tar >/dev/null 2>&1; then
        missing_deps+=("tar")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        error "Missing required dependencies: ${missing_deps[*]}"
    fi
}

get_latest_version() {
    local api_url="https://api.github.com/repos/jfrog/jfrog-cli-vm/releases/latest"
    local version
    
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -s "$api_url" | grep '"tag_name"' | head -1 | cut -d '"' -f 4 2>/dev/null)
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$api_url" | grep '"tag_name"' | head -1 | cut -d '"' -f 4 2>/dev/null)
    fi
    
    if [ -z "$version" ] || [ "$version" = "null" ]; then
        error "Failed to fetch latest version from GitHub API"
    fi
    
    echo "$version"
}

download_file() {
    local url="$1"
    local dest="$2"
    
    if command -v curl >/dev/null 2>&1; then
        curl -L -f -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$dest" "$url"
    else
        error "Neither curl nor wget is available"
    fi
}

check_existing_installation() {
    if command -v jfvm >/dev/null 2>&1 && [ "$FORCE_INSTALL" != "true" ]; then
        local existing_version=$(jfvm --version 2>/dev/null | head -1 || echo "unknown")
        warning "JFVM is already installed: $existing_version"
        
        if [ "$SILENT" != "true" ]; then
            echo ""
            echo "To force reinstallation, set JFVM_FORCE=true or use --force"
            echo "To update to the latest version, use: jfvm use latest"
            echo ""
            exit 0
        fi
    fi
}

install_jfvm() {
    local platform=$(detect_platform)
    if [ "$platform" = "unsupported" ]; then
        error "Unsupported platform: $(uname -s)-$(uname -m)"
    fi
    
    local version="$JFVM_VERSION"
    if [ "$version" = "latest" ]; then
        info "Fetching latest JFVM version..."
        version=$(get_latest_version)
        success "Latest version: $version"
    fi
    
    # Remove 'v' prefix if present
    version=${version#v}
    
    local download_url="https://releases.jfrog.io/artifactory/jfvm/v1/v${version}/jfvm-${platform}/jfvm"
    local temp_file="/tmp/jfvm-${version}-$$"
    
    info "Downloading JFVM ${version} for ${platform}..."
    
    if ! download_file "$download_url" "$temp_file"; then
        error "Failed to download JFVM from $download_url"
    fi
    
    success "JFVM downloaded successfully"
    
    # Verify download
    if [ ! -f "$temp_file" ] || [ ! -s "$temp_file" ]; then
        error "Downloaded file is empty or missing"
    fi
    
    # Make executable
    chmod +x "$temp_file"
    
    # Test the binary
    if ! "$temp_file" --version >/dev/null 2>&1; then
        warning "Downloaded binary failed basic test (may still work on target system)"
    fi
    
    # Install to target directory
    install_binary "$temp_file" "$version"
    
    # Cleanup
    rm -f "$temp_file"
}

install_binary() {
    local temp_binary="$1"
    local version="$2"
    local target_binary="$INSTALL_DIR/jfvm"
    
    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            # Try with sudo if we don't have permissions
            if ! sudo mkdir -p "$INSTALL_DIR" 2>/dev/null; then
                # Fall back to user directory
                INSTALL_DIR="$HOME/.local/bin"
                target_binary="$INSTALL_DIR/jfvm"
                mkdir -p "$INSTALL_DIR"
                warning "Installing to user directory: $INSTALL_DIR"
            fi
        fi
    fi
    
    # Install the binary
    if [ -w "$INSTALL_DIR" ]; then
        cp "$temp_binary" "$target_binary"
    else
        if ! sudo cp "$temp_binary" "$target_binary" 2>/dev/null; then
            # Fall back to user directory
            INSTALL_DIR="$HOME/.local/bin"
            target_binary="$INSTALL_DIR/jfvm"
            mkdir -p "$INSTALL_DIR"
            cp "$temp_binary" "$target_binary"
            warning "Installed to user directory: $INSTALL_DIR"
        fi
    fi
    
    success "JFVM installed to $target_binary"
    
    # Add to PATH if needed
    setup_path
    
    # Initialize JFVM directories
    mkdir -p "$JFVM_HOME"/{versions,shim}
    
    # Display installation info
    show_installation_info "$version"
}

setup_path() {
    # Check if INSTALL_DIR is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warning "$INSTALL_DIR is not in your PATH"
        
        # Add to shell configuration files
        local shell_configs=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile")
        local path_line="export PATH=\"$INSTALL_DIR:\$PATH\""
        
        for config_file in "${shell_configs[@]}"; do
            if [ -f "$config_file" ]; then
                if ! grep -q "$INSTALL_DIR" "$config_file" 2>/dev/null; then
                    echo "" >> "$config_file"
                    echo "# Added by JFVM installer" >> "$config_file"
                    echo "$path_line" >> "$config_file"
                    success "Added $INSTALL_DIR to PATH in $config_file"
                fi
            fi
        done
        
        # Update current session
        export PATH="$INSTALL_DIR:$PATH"
    fi
}

show_installation_info() {
    local version="$1"
    
    if [ "$SILENT" != "true" ]; then
        echo ""
        echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║                    Installation Complete!                        ║${NC}"
        echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
        echo ""
        success "JFVM $version installed successfully!"
        echo ""
        
        # Verify installation
        if command -v jfvm >/dev/null 2>&1; then
            local installed_version=$(jfvm --version 2>/dev/null | head -1 || echo "unknown")
            info "Installed version: $installed_version"
        else
            warning "jfvm command not found in current session"
            info "You may need to restart your terminal or run: source ~/.bashrc"
        fi
        
        echo ""
        info "Quick start:"
        echo "  jfvm install latest    # Install latest JFrog CLI"
        echo "  jfvm use latest        # Switch to latest version"
        echo "  jfvm list              # List installed versions"
        echo "  jfvm --help            # Show all commands"
        echo ""
        
        info "💡 Optional: Install JFrog CLI for full platform integration:"
        echo "  curl -fL https://install-cli.jfrog.io | sh"
        echo ""
        
        info "📖 Documentation: https://github.com/jfrog/jfrog-cli-vm/blob/main/README.md"
        echo ""
    fi
}

show_help() {
    cat << EOF
JFVM Download Script

Downloads and installs JFVM (JFrog CLI Version Manager) on Unix-like systems.

Usage:
  curl -fL https://get.jfrog.io/jfvm | sh
  wget -qO- https://get.jfrog.io/jfvm | sh

Environment Variables:
  JFVM_VERSION        Version to install (default: latest)
  JFVM_INSTALL_DIR    Installation directory (default: /usr/local/bin)
  JFVM_HOME           JFVM home directory (default: ~/.jfvm)
  JFVM_FORCE          Force installation over existing (default: false)
  JFVM_SILENT         Silent installation (default: false)

Examples:
  # Install specific version
  JFVM_VERSION=v1.0.0 curl -fL https://get.jfrog.io/jfvm | sh
  
  # Install to custom directory
  JFVM_INSTALL_DIR=/opt/jfvm curl -fL https://get.jfrog.io/jfvm | sh
  
  # Silent installation
  JFVM_SILENT=true curl -fL https://get.jfrog.io/jfvm | sh
  
  # Force reinstallation
  JFVM_FORCE=true curl -fL https://get.jfrog.io/jfvm | sh

For more information, visit:
https://github.com/jfrog/jfrog-cli-vm
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            JFVM_VERSION="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --force)
            FORCE_INSTALL="true"
            shift
            ;;
        --silent)
            SILENT="true"
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Main execution
main() {
    print_banner
    
    check_dependencies
    
    check_existing_installation
    
    install_jfvm
}

# Run main function if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi



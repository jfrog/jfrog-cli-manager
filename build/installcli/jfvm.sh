#!/bin/bash

# JFVM Installation Script with JFrog CLI Integration Option
# This script installs JFVM and optionally JFrog CLI as requested in the requirements

set -e

# Configuration
JFVM_VERSION="${JFVM_VERSION:-latest}"
JFROG_CLI_VERSION="${JFROG_CLI_VERSION:-latest}"
INSTALL_DIR="${JFVM_INSTALL_DIR:-/usr/local/bin}"
JFVM_HOME="${JFVM_HOME:-$HOME/.jfvm}"
INSTALL_JFROG_CLI="${JFVM_INSTALL_JFROG_CLI:-}"
SILENT_INSTALL="${JFVM_SILENT_INSTALL:-false}"

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
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════════════╗"
    echo "║                     JFrog CLI Installation Wizard                ║"
    echo "║                                                                   ║"
    echo "║  This installer will set up JFVM (JFrog CLI Version Manager)     ║"
    echo "║  You can optionally install JFrog CLI alongside JFVM.            ║"
    echo "║                                                                   ║"
    echo "║  JFVM helps you manage multiple versions of JFrog CLI with ease. ║"
    echo "╚═══════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}\n"
}

log() {
    echo -e "$1"
}

error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

warning() {
    echo -e "${YELLOW}Warning: $1${NC}"
}

success() {
    echo -e "${GREEN}$1${NC}"
}

info() {
    echo -e "${BLUE}$1${NC}"
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
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        error "Missing required dependencies: ${missing_deps[*]}"
    fi
}

get_latest_version() {
    local repo="$1"
    local api_url="https://api.github.com/repos/jfrog/${repo}/releases/latest"
    local version
    
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -s "$api_url" | grep '"tag_name"' | head -1 | cut -d '"' -f 4 2>/dev/null)
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$api_url" | grep '"tag_name"' | head -1 | cut -d '"' -f 4 2>/dev/null)
    fi
    
    if [ -z "$version" ] || [ "$version" = "null" ]; then
        error "Failed to fetch latest version from GitHub API for $repo"
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

prompt_jfrog_cli_installation() {
    # Check environment variable first
    if [ "$INSTALL_JFROG_CLI" = "true" ]; then
        return 0
    elif [ "$INSTALL_JFROG_CLI" = "false" ]; then
        return 1
    fi
    
    # Skip prompt if silent install
    if [ "$SILENT_INSTALL" = "true" ]; then
        return 1  # Default to NO when silent
    fi
    
    echo ""
    echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║                    Optional Component: JFrog CLI                 ║${NC}"
    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "JFrog CLI provides comprehensive artifact management capabilities."
    echo "Installing it alongside JFVM enables full JFrog platform integration."
    echo ""
    echo "Features with JFrog CLI:"
    echo "  • Upload and download artifacts"
    echo "  • Build integration and metadata"
    echo "  • Security scanning (Xray)"
    echo "  • Repository management"
    echo "  • CI/CD pipeline integration"
    echo ""
    echo -e "📖 Learn more: ${BLUE}https://docs.jfrog-applications.jfrog.io/jfrog-applications/jfrog-cli${NC}"
    echo ""
    
    # Default is NO (unchecked by default as per requirements)
    printf "Install JFrog CLI alongside JFVM? [y/N]: "
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            INSTALL_JFROG_CLI="true"
            success "JFrog CLI will be installed alongside JFVM"
            return 0
            ;;
        *)
            info "Proceeding with JFVM installation only"
            echo ""
            info "💡 You can install JFrog CLI later using:"
            echo "  • curl -fL https://install-cli.jfrog.io | sh"
            echo "  • jfvm install latest  # (installs JFrog CLI via JFVM)"
            return 1
            ;;
    esac
    echo ""
}

install_jfvm() {
    local platform=$(detect_platform)
    if [ "$platform" = "unsupported" ]; then
        error "Unsupported platform: $(uname -s)-$(uname -m)"
    fi
    
    local version="$JFVM_VERSION"
    if [ "$version" = "latest" ]; then
        info "Fetching latest JFVM version..."
        version=$(get_latest_version "jfrog-cli-vm")
        success "Latest JFVM version: $version"
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
    
    # Make executable
    chmod +x "$temp_file"
    
    # Install to target directory
    install_binary "$temp_file" "jfvm" "$INSTALL_DIR"
    
    # Initialize JFVM directories
    mkdir -p "$JFVM_HOME"/{versions,shim}
    
    # Set up shell integration
    setup_jfvm_shell_integration
    
    # Cleanup
    rm -f "$temp_file"
    
    success "JFVM installation completed"
}

install_jfrog_cli() {
    if [ "$INSTALL_JFROG_CLI" != "true" ]; then
        return 0
    fi
    
    local platform=$(detect_platform)
    local version="$JFROG_CLI_VERSION"
    
    if [ "$version" = "latest" ]; then
        info "Fetching latest JFrog CLI version..."
        version=$(get_latest_version "jfrog-cli")
        success "Latest JFrog CLI version: $version"
    fi
    
    # Remove 'v' prefix if present
    version=${version#v}
    
    # Map platform for JFrog CLI naming convention
    local cli_platform
    case "$platform" in
        "linux-amd64") cli_platform="linux-amd64" ;;
        "linux-arm64") cli_platform="linux-arm64" ;;
        "linux-386") cli_platform="linux-386" ;;
        "mac-amd64") cli_platform="mac-386" ;;
        "mac-arm64") cli_platform="mac-arm64" ;;
        *) cli_platform="$platform" ;;
    esac
    
    local download_url="https://releases.jfrog.io/artifactory/jfrog-cli/v2-jf/${version}/jfrog-cli-${cli_platform}/jf"
    local temp_file="/tmp/jf-${version}-$$"
    
    info "Downloading JFrog CLI ${version} for ${cli_platform}..."
    
    if ! download_file "$download_url" "$temp_file"; then
        error "Failed to download JFrog CLI from $download_url"
    fi
    
    success "JFrog CLI downloaded successfully"
    
    # Make executable
    chmod +x "$temp_file"
    
    # Install to target directory
    install_binary "$temp_file" "jf" "$INSTALL_DIR"
    
    # Cleanup
    rm -f "$temp_file"
    
    success "JFrog CLI installation completed"
}

install_binary() {
    local temp_binary="$1"
    local binary_name="$2"
    local install_dir="$3"
    local target_binary="$install_dir/$binary_name"
    
    # Create install directory if it doesn't exist
    if [ ! -d "$install_dir" ]; then
        if ! mkdir -p "$install_dir" 2>/dev/null; then
            # Try with sudo if we don't have permissions
            if ! sudo mkdir -p "$install_dir" 2>/dev/null; then
                # Fall back to user directory
                install_dir="$HOME/.local/bin"
                target_binary="$install_dir/$binary_name"
                mkdir -p "$install_dir"
                warning "Installing $binary_name to user directory: $install_dir"
            fi
        fi
    fi
    
    # Install the binary
    if [ -w "$install_dir" ]; then
        cp "$temp_binary" "$target_binary"
    else
        if ! sudo cp "$temp_binary" "$target_binary" 2>/dev/null; then
            # Fall back to user directory
            install_dir="$HOME/.local/bin"
            target_binary="$install_dir/$binary_name"
            mkdir -p "$install_dir"
            cp "$temp_binary" "$target_binary"
            warning "Installed $binary_name to user directory: $install_dir"
        fi
    fi
    
    success "$binary_name installed to $target_binary"
    
    # Add to PATH if needed
    setup_path "$install_dir"
}

setup_path() {
    local install_dir="$1"
    
    # Check if install_dir is in PATH
    if [[ ":$PATH:" != *":$install_dir:"* ]]; then
        warning "$install_dir is not in your PATH"
        
        # Add to shell configuration files
        local shell_configs=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile")
        local path_line="export PATH=\"$install_dir:\$PATH\""
        
        for config_file in "${shell_configs[@]}"; do
            if [ -f "$config_file" ]; then
                if ! grep -q "$install_dir" "$config_file" 2>/dev/null; then
                    echo "" >> "$config_file"
                    echo "# Added by JFVM installer" >> "$config_file"
                    echo "$path_line" >> "$config_file"
                    success "Added $install_dir to PATH in $config_file"
                fi
            fi
        done
        
        # Update current session
        export PATH="$install_dir:$PATH"
    fi
}

setup_jfvm_shell_integration() {
    info "Setting up JFVM shell integration..."
    
    local shell_configs=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile")
    local jfvm_config="
# JFVM Configuration - JFrog CLI Version Manager
export PATH=\"\$HOME/.jfvm/shim:\$PATH\"

# JFVM shell function for enhanced priority (similar to nvm approach)
jf() {
    # Check if jfvm shim exists and is executable
    if [ -x \"\$HOME/.jfvm/shim/jf\" ]; then
        # Execute jfvm-managed jf with highest priority
        \"\$HOME/.jfvm/shim/jf\" \"\$@\"
    else
        # Fallback to system jf if jfvm shim not available
        command jf \"\$@\"
    fi
}"
    
    for config_file in "${shell_configs[@]}"; do
        if [ -f "$config_file" ]; then
            # Check if JFVM configuration already exists
            if ! grep -q "JFVM Configuration" "$config_file" 2>/dev/null; then
                echo "$jfvm_config" >> "$config_file"
                success "Added JFVM configuration to $config_file"
            fi
        fi
    done
    
    warning "Please restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"
}

verify_installation() {
    info "Verifying installations..."
    
    # Verify JFVM
    if command -v jfvm >/dev/null 2>&1; then
        local jfvm_version=$(jfvm --version 2>/dev/null | head -1 || echo "unknown")
        success "JFVM installed: $jfvm_version"
    else
        warning "jfvm command not found in current session"
    fi
    
    # Verify JFrog CLI if installed
    if [ "$INSTALL_JFROG_CLI" = "true" ]; then
        if command -v jf >/dev/null 2>&1; then
            local jf_version=$(jf --version 2>/dev/null | head -1 || echo "unknown")
            success "JFrog CLI installed: $jf_version"
        else
            warning "jf command not found in current session"
        fi
    fi
}

show_completion_message() {
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    Installation Complete!                        ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    success "✅ JFVM installed successfully"
    
    if [ "$INSTALL_JFROG_CLI" = "true" ]; then
        success "✅ JFrog CLI installed successfully"
        echo ""
        info "Both JFVM and JFrog CLI are now available!"
    fi
    
    echo ""
    info "Next steps:"
    
    if [ "$INSTALL_JFROG_CLI" = "true" ]; then
        echo "  jf config add          # Configure JFrog platform connection"
        echo "  jfvm use latest        # Ensure latest JFrog CLI version"
    else
        echo "  jfvm install latest    # Install latest JFrog CLI"
        echo "  jfvm use latest        # Switch to latest version"
    fi
    
    echo "  jfvm list              # List installed versions"
    echo "  jfvm --help            # Show all JFVM commands"
    
    if [ "$INSTALL_JFROG_CLI" = "true" ]; then
        echo "  jf --help              # Show all JFrog CLI commands"
    fi
    
    echo ""
    info "Environment variables for automation:"
    echo "  JFVM_INSTALL_JFROG_CLI=true    # Auto-install JFrog CLI"
    echo "  JFVM_SILENT_INSTALL=true       # Skip prompts"
    echo "  JFVM_VERSION=v1.0.0            # Install specific version"
    echo ""
    
    info "📖 Documentation:"
    echo "  JFVM: https://github.com/jfrog/jfrog-cli-vm/blob/main/README.md"
    if [ "$INSTALL_JFROG_CLI" = "true" ]; then
        echo "  JFrog CLI: https://docs.jfrog-applications.jfrog.io/jfrog-applications/jfrog-cli"
    fi
    echo ""
}

show_help() {
    cat << EOF
JFVM Installation Script with JFrog CLI Integration

This script installs JFVM (JFrog CLI Version Manager) and optionally JFrog CLI.

Usage:
  curl -fL https://install.jfrog.io/jfvm | sh
  wget -qO- https://install.jfrog.io/jfvm | sh

Environment Variables:
  JFVM_VERSION              JFVM version to install (default: latest)
  JFROG_CLI_VERSION         JFrog CLI version to install (default: latest)
  JFVM_INSTALL_DIR          Installation directory (default: /usr/local/bin)
  JFVM_HOME                 JFVM home directory (default: ~/.jfvm)
  JFVM_INSTALL_JFROG_CLI    Install JFrog CLI (true/false, prompts if unset)
  JFVM_SILENT_INSTALL       Skip prompts (default: false)

Examples:
  # Install with JFrog CLI automatically
  JFVM_INSTALL_JFROG_CLI=true curl -fL https://install.jfrog.io/jfvm | sh
  
  # Silent installation without JFrog CLI
  JFVM_INSTALL_JFROG_CLI=false JFVM_SILENT_INSTALL=true curl -fL https://install.jfrog.io/jfvm | sh
  
  # Install specific versions
  JFVM_VERSION=v1.0.0 JFROG_CLI_VERSION=v2.50.0 curl -fL https://install.jfrog.io/jfvm | sh

For more information:
  JFVM: https://github.com/jfrog/jfrog-cli-vm
  JFrog CLI: https://docs.jfrog-applications.jfrog.io/jfrog-applications/jfrog-cli
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --with-jfrog-cli)
            INSTALL_JFROG_CLI="true"
            shift
            ;;
        --without-jfrog-cli)
            INSTALL_JFROG_CLI="false"
            shift
            ;;
        --silent)
            SILENT_INSTALL="true"
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
    
    prompt_jfrog_cli_installation
    
    install_jfvm
    
    install_jfrog_cli
    
    verify_installation
    
    show_completion_message
}

# Run main function if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi



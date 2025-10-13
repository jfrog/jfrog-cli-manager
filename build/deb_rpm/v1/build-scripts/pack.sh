#!/bin/bash
set -e

# Package creation script for Debian and RPM packages
# This script builds both DEB and RPM packages for JFVM

JFVM_VERSION=""
BINARY_PATH=""
PACKAGE_TYPE=""
ARCHITECTURE=""
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            JFVM_VERSION="$2"
            shift 2
            ;;
        -b|--binary)
            BINARY_PATH="$2"
            shift 2
            ;;
        -t|--type)
            PACKAGE_TYPE="$2"
            shift 2
            ;;
        -a|--arch)
            ARCHITECTURE="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 -v <version> -b <binary_path> -t <deb|rpm> -a <architecture> [--verbose]"
            echo ""
            echo "Create Debian or RPM packages for JFVM"
            echo ""
            echo "Options:"
            echo "  -v, --version     JFVM version"
            echo "  -b, --binary      Path to JFVM binary"
            echo "  -t, --type        Package type (deb or rpm)"
            echo "  -a, --arch        Target architecture"
            echo "  --verbose         Enable verbose output"
            echo "  -h, --help        Show this help message"
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
if [ -z "$JFVM_VERSION" ] || [ -z "$BINARY_PATH" ] || [ -z "$PACKAGE_TYPE" ] || [ -z "$ARCHITECTURE" ]; then
    echo "Error: All required arguments must be provided"
    echo "Usage: $0 -v <version> -b <binary_path> -t <deb|rpm> -a <architecture>"
    exit 1
fi

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary file '$BINARY_PATH' does not exist"
    exit 1
fi

# Validate package type
if [ "$PACKAGE_TYPE" != "deb" ] && [ "$PACKAGE_TYPE" != "rpm" ]; then
    echo "Error: Package type must be 'deb' or 'rpm'"
    exit 1
fi

# Verbose output function
log() {
    if [ "$VERBOSE" = true ]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
    fi
}

# Clean version (remove 'v' prefix if present)
CLEAN_VERSION=$(echo "$JFVM_VERSION" | sed 's/^v//')

# Create build directory
BUILD_DIR="build-$PACKAGE_TYPE-$ARCHITECTURE"
mkdir -p "$BUILD_DIR"

log "Starting $PACKAGE_TYPE package creation..."
log "Version: $CLEAN_VERSION"
log "Binary: $BINARY_PATH"
log "Architecture: $ARCHITECTURE"
log "Build directory: $BUILD_DIR"

if [ "$PACKAGE_TYPE" = "deb" ]; then
    create_deb_package
elif [ "$PACKAGE_TYPE" = "rpm" ]; then
    create_rpm_package
fi

log "$PACKAGE_TYPE package creation completed"

create_deb_package() {
    log "Creating Debian package..."
    
    # Map architecture names for Debian
    case "$ARCHITECTURE" in
        "amd64"|"x86_64") DEB_ARCH="amd64" ;;
        "386"|"i386") DEB_ARCH="i386" ;;
        "arm64"|"aarch64") DEB_ARCH="arm64" ;;
        "arm") DEB_ARCH="armhf" ;;
        *) DEB_ARCH="$ARCHITECTURE" ;;
    esac
    
    PACKAGE_DIR="$BUILD_DIR/jfvm_${CLEAN_VERSION}_${DEB_ARCH}"
    
    # Create package structure
    mkdir -p "$PACKAGE_DIR"/{DEBIAN,usr/bin,usr/share/doc/jfvm,usr/share/man/man1}
    
    # Copy binary
    cp "$BINARY_PATH" "$PACKAGE_DIR/usr/bin/jfvm"
    chmod 755 "$PACKAGE_DIR/usr/bin/jfvm"
    
    # Create control file
    cat > "$PACKAGE_DIR/DEBIAN/control" << EOF
Package: jfvm
Version: $CLEAN_VERSION
Section: utils
Priority: optional
Architecture: $DEB_ARCH
Depends: libc6
Maintainer: JFrog Ltd. <support@jfrog.com>
Description: JFrog CLI Version Manager
 JFVM (JFrog CLI Version Manager) is a powerful tool that helps you manage
 multiple versions of JFrog CLI on your system. Features include:
 .
 * Install and manage multiple JFrog CLI versions
 * Switch between versions easily
 * Set project-specific JFrog CLI versions
 * Compare performance between versions
 * Track usage analytics
 * Automatic version detection from .jfrog-version files
Homepage: https://github.com/jfrog/jfrog-cli-vm
Bugs: https://github.com/jfrog/jfrog-cli-vm/issues
EOF

    # Create postinst script
    cat > "$PACKAGE_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
set -e

echo ""
echo "✅ JFVM installed successfully!"
echo ""
echo "💡 Optional: Install JFrog CLI for full JFrog platform integration:"
echo "   curl -fL https://install-cli.jfrog.io | sh"
echo "   # or"
echo "   wget -qO- https://install-cli.jfrog.io | sh"
echo ""
echo "Next steps:"
echo "  jfvm install latest    # Install latest JFrog CLI"
echo "  jfvm use latest        # Switch to latest version"
echo "  jfvm --help            # Show all commands"
echo ""
echo "📖 Documentation: https://github.com/jfrog/jfrog-cli-vm/blob/main/README.md"
echo ""
EOF
    chmod 755 "$PACKAGE_DIR/DEBIAN/postinst"
    
    # Create prerm script
    cat > "$PACKAGE_DIR/DEBIAN/prerm" << 'EOF'
#!/bin/bash
set -e

echo "Removing JFVM..."

# Clean up JFVM data if user confirms
if [ "$1" = "remove" ] || [ "$1" = "purge" ]; then
    echo ""
    echo "Note: JFVM configuration and downloaded CLI versions will remain in ~/.jfvm"
    echo "To completely remove all JFVM data, run: rm -rf ~/.jfvm"
fi
EOF
    chmod 755 "$PACKAGE_DIR/DEBIAN/prerm"
    
    # Create copyright file
    cat > "$PACKAGE_DIR/usr/share/doc/jfvm/copyright" << EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: jfvm
Source: https://github.com/jfrog/jfrog-cli-vm

Files: *
Copyright: 2024 JFrog Ltd.
License: MIT

License: MIT
 Permission is hereby granted, free of charge, to any person obtaining a
 copy of this software and associated documentation files (the "Software"),
 to deal in the Software without restriction, including without limitation
 the rights to use, copy, modify, merge, publish, distribute, sublicense,
 and/or sell copies of the Software, and to permit persons to whom the
 Software is furnished to do so, subject to the following conditions:
 .
 The above copyright notice and this permission notice shall be included
 in all copies or substantial portions of the Software.
 .
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
 OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
 THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
 DEALINGS IN THE SOFTWARE.
EOF

    # Create changelog
    cat > "$PACKAGE_DIR/usr/share/doc/jfvm/changelog.Debian" << EOF
jfvm ($CLEAN_VERSION-1) stable; urgency=medium

  * Release $JFVM_VERSION

 -- JFrog Release Team <support@jfrog.com>  $(date -R)
EOF
    gzip -9 "$PACKAGE_DIR/usr/share/doc/jfvm/changelog.Debian"
    
    # Create man page
    cat > "$PACKAGE_DIR/usr/share/man/man1/jfvm.1" << 'EOF'
.TH JFVM 1 "2024" "JFVM" "User Commands"
.SH NAME
jfvm \- JFrog CLI Version Manager
.SH SYNOPSIS
.B jfvm
[\fIOPTION\fR]... [\fICOMMAND\fR]...
.SH DESCRIPTION
JFVM (JFrog CLI Version Manager) is a powerful tool that helps you manage multiple versions of JFrog CLI on your system.
.SH COMMANDS
.TP
.B install VERSION
Install a specific version of JFrog CLI
.TP
.B use VERSION
Switch to a specific version of JFrog CLI
.TP
.B list
Show all installed versions
.TP
.B remove VERSION
Remove a specific version
.TP
.B clear
Remove all installed versions
.TP
.B alias NAME VERSION
Create an alias for a version
.TP
.B --help
Show help information
.SH EXAMPLES
.TP
jfvm install 2.50.0
Install JFrog CLI version 2.50.0
.TP
jfvm use latest
Switch to the latest version
.TP
jfvm list
List all installed versions
.SH SEE ALSO
Full documentation at: https://github.com/jfrog/jfrog-cli-vm
.SH AUTHOR
JFrog Ltd. <support@jfrog.com>
EOF
    gzip -9 "$PACKAGE_DIR/usr/share/man/man1/jfvm.1"
    
    # Build the package
    log "Building Debian package..."
    dpkg-deb --build "$PACKAGE_DIR"
    
    # Move to final location
    FINAL_DEB="jfvm_${CLEAN_VERSION}_${DEB_ARCH}.deb"
    mv "${PACKAGE_DIR}.deb" "$FINAL_DEB"
    
    log "Debian package created: $FINAL_DEB"
    
    # Verify package
    if command -v dpkg >/dev/null 2>&1; then
        log "Package information:"
        dpkg --info "$FINAL_DEB"
    fi
}

create_rpm_package() {
    log "Creating RPM package..."
    
    # Map architecture names for RPM
    case "$ARCHITECTURE" in
        "amd64"|"x86_64") RPM_ARCH="x86_64" ;;
        "386"|"i386") RPM_ARCH="i386" ;;
        "arm64"|"aarch64") RPM_ARCH="aarch64" ;;
        "arm") RPM_ARCH="armv7hl" ;;
        *) RPM_ARCH="$ARCHITECTURE" ;;
    esac
    
    # Setup RPM build environment
    RPM_BUILD_ROOT="$BUILD_DIR/rpmbuild"
    mkdir -p "$RPM_BUILD_ROOT"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
    
    # Copy binary to SOURCES
    cp "$BINARY_PATH" "$RPM_BUILD_ROOT/SOURCES/jfvm"
    
    # Create spec file
    cat > "$RPM_BUILD_ROOT/SPECS/jfvm.spec" << EOF
Name:           jfvm
Version:        $CLEAN_VERSION
Release:        1%{?dist}
Summary:        JFrog CLI Version Manager
License:        MIT
URL:            https://github.com/jfrog/jfrog-cli-vm
Source0:        jfvm
BuildArch:      $RPM_ARCH
Requires:       glibc

%description
JFVM (JFrog CLI Version Manager) is a powerful tool that helps you manage
multiple versions of JFrog CLI on your system. Features include:

* Install and manage multiple JFrog CLI versions
* Switch between versions easily  
* Set project-specific JFrog CLI versions
* Compare performance between versions
* Track usage analytics
* Automatic version detection from .jfrog-version files

%prep
# No prep needed for binary package

%build
# No build needed for binary package

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_mandir}/man1
mkdir -p %{buildroot}%{_docdir}/%{name}

# Install binary
install -m 755 %{SOURCE0} %{buildroot}%{_bindir}/jfvm

# Create man page
cat > %{buildroot}%{_mandir}/man1/jfvm.1 << 'MANEOF'
.TH JFVM 1 "2024" "JFVM" "User Commands"
.SH NAME
jfvm \\- JFrog CLI Version Manager
.SH SYNOPSIS
.B jfvm
[\\fIOPTION\\fR]... [\\fICOMMAND\\fR]...
.SH DESCRIPTION
JFVM (JFrog CLI Version Manager) is a powerful tool that helps you manage multiple versions of JFrog CLI on your system.
.SH COMMANDS
.TP
.B install VERSION
Install a specific version of JFrog CLI
.TP
.B use VERSION
Switch to a specific version of JFrog CLI
.TP
.B list
Show all installed versions
.TP
.B remove VERSION
Remove a specific version
.TP
.B clear
Remove all installed versions
.TP
.B alias NAME VERSION
Create an alias for a version
.TP
.B --help
Show help information
.SH EXAMPLES
.TP
jfvm install 2.50.0
Install JFrog CLI version 2.50.0
.TP
jfvm use latest
Switch to the latest version
.TP
jfvm list
List all installed versions
.SH SEE ALSO
Full documentation at: https://github.com/jfrog/jfrog-cli-vm
.SH AUTHOR
JFrog Ltd. <support@jfrog.com>
MANEOF

# Create README
cat > %{buildroot}%{_docdir}/%{name}/README << 'READMEEOF'
JFVM - JFrog CLI Version Manager

JFVM helps you manage multiple versions of JFrog CLI on your system.

Quick Start:
  jfvm install latest    # Install latest JFrog CLI
  jfvm use latest        # Switch to latest version
  jfvm list              # List installed versions
  jfvm --help            # Show all commands

For full documentation, visit:
https://github.com/jfrog/jfrog-cli-vm/blob/main/README.md

Optional: Install JFrog CLI for full JFrog platform integration:
  curl -fL https://install-cli.jfrog.io | sh
  # or
  wget -qO- https://install-cli.jfrog.io | sh
READMEEOF

%files
%{_bindir}/jfvm
%{_mandir}/man1/jfvm.1*
%{_docdir}/%{name}/README

%post
echo ""
echo "✅ JFVM installed successfully!"
echo ""
echo "💡 Optional: Install JFrog CLI for full JFrog platform integration:"
echo "   curl -fL https://install-cli.jfrog.io | sh"
echo "   # or"
echo "   dnf install jfrog-cli-v2-jf"
echo ""
echo "Next steps:"
echo "  jfvm install latest    # Install latest JFrog CLI"
echo "  jfvm use latest        # Switch to latest version"
echo "  jfvm --help            # Show all commands"
echo ""
echo "📖 Documentation: https://github.com/jfrog/jfrog-cli-vm/blob/main/README.md"
echo ""

%preun
if [ "\$1" = "0" ]; then
    echo "Removing JFVM..."
    echo ""
    echo "Note: JFVM configuration and downloaded CLI versions will remain in ~/.jfvm"
    echo "To completely remove all JFVM data, run: rm -rf ~/.jfvm"
fi

%changelog
* $(date +'%a %b %d %Y') JFrog Release Team <support@jfrog.com> - $CLEAN_VERSION-1
- Release $JFVM_VERSION
EOF

    # Build RPM
    log "Building RPM package..."
    
    rpmbuild --define "_topdir $PWD/$RPM_BUILD_ROOT" \
             --define "_rpmdir $PWD" \
             --define "_build_name_fmt %%{NAME}-%%{VERSION}-%%{RELEASE}.%%{ARCH}.rpm" \
             -bb "$RPM_BUILD_ROOT/SPECS/jfvm.spec"
    
    FINAL_RPM="jfvm-${CLEAN_VERSION}-1.${RPM_ARCH}.rpm"
    
    log "RPM package created: $FINAL_RPM"
    
    # Verify package
    if command -v rpm >/dev/null 2>&1; then
        log "Package information:"
        rpm -qip "$FINAL_RPM"
    fi
}

# Cleanup function
cleanup() {
    if [ -d "$BUILD_DIR" ]; then
        rm -rf "$BUILD_DIR"
        log "Cleaned up build directory"
    fi
}

# Set trap for cleanup on exit
trap cleanup EXIT

echo "Package creation completed successfully!"
echo "Created: $(ls -1 *.deb *.rpm 2>/dev/null | head -1 || echo 'Package file')"



# JFVM Multi-Platform Build System

This directory contains the complete multi-platform build and distribution system for JFVM, following the same patterns and infrastructure as JFrog CLI.

## Directory Structure

```
build/
├── build.sh                 # Main Unix build script
├── build.bat                # Windows build script
├── sign/                    # Windows signing infrastructure
│   ├── Dockerfile           # Docker container for Windows signing
│   └── sign-windows.sh      # Windows signing script
├── apple_release/           # macOS signing and notarization
│   └── scripts/
│       └── darwin-sign-and-notarize.sh
├── npm/                     # NPM package configurations
│   └── v1/                  # NPM v1 package template
├── chocolatey/              # Chocolatey package configurations
│   └── v1/                  # Chocolatey v1 package template
├── deb_rpm/                 # Debian and RPM package creation
│   └── v1/
│       └── build-scripts/
│           ├── pack.sh      # Universal package creation script
│           └── rpm-sign.sh  # RPM signing script
├── docker/                  # Docker image creation
│   ├── slim/                # Slim JFVM-only images
│   └── full/                # Full images with JFrog CLI
├── getcli/                  # Standalone download scripts
│   └── jfvm.sh              # Get JFVM script (like get.jfrog.io/jf)
├── installcli/              # Installation scripts with CLI integration
│   └── jfvm.sh              # Install script with JFrog CLI option
└── setupcli/                # Setup and configuration scripts
```

## Build Process

### 1. Jenkins Pipeline

The main build process is orchestrated by the `Jenkinsfile` in the repository root. This pipeline:

- Builds binaries for all supported platforms
- Signs Windows and macOS binaries using JFrog's certificates
- Creates packages for all distribution channels
- Uploads to Artifactory
- Publishes to package repositories (NPM, Chocolatey, etc.)

### 2. Supported Platforms

- **Windows**: amd64
- **Linux**: 386, amd64, arm64, arm, s390x, ppc64, ppc64le
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)

### 3. Build Scripts

#### Unix Build (`build.sh`)
```bash
./build.sh [executable_name] [version]
```

#### Windows Build (`build.bat`)
```batch
build.bat [executable_name] [version]
```

## Signing Infrastructure

### Windows Signing
Uses Docker container with `osslsigncode` and JFrog's DigiCert certificates:

```bash
cd build/sign
docker build -t jfvm-sign-tool .
docker run -v $(pwd):/workspace jfvm-sign-tool -in unsigned.exe -out signed.exe
```

### macOS Signing
Uses Apple's codesign and notarization tools:

```bash
cd build/apple_release/scripts
./darwin-sign-and-notarize.sh -i input_binary -o output_binary
```

### Linux Package Signing
RPM packages are signed using GPG:

```bash
cd build/deb_rpm/v1/build-scripts
./rpm-sign.sh -f package.rpm -k gpg_key.asc
```

## Package Creation

### Debian/RPM Packages
```bash
cd build/deb_rpm/v1/build-scripts
./pack.sh -v 1.0.0 -b /path/to/jfvm -t deb -a amd64
./pack.sh -v 1.0.0 -b /path/to/jfvm -t rpm -a x86_64
```

### NPM Package
The NPM package is created during the Jenkins pipeline and includes:
- Cross-platform binary download logic
- Interactive JFrog CLI installation prompt
- Proper npm lifecycle management

### Chocolatey Package
Windows package manager integration with:
- Parameter support for JFrog CLI installation
- Proper PATH management
- Windows-specific post-install scripts

### Docker Images
Two variants are created:
- **Slim**: JFVM only
- **Full**: JFVM + JFrog CLI

## Distribution Endpoints

### Artifactory Paths
```
https://releases.jfrog.io/artifactory/
├── jfvm/v1/${version}/
│   ├── jfvm-windows-amd64/
│   ├── jfvm-linux-amd64/
│   ├── jfvm-mac-amd64/
│   └── jfvm-mac-arm64/
├── jfvm-debs/
├── jfvm-rpms/
├── jfvm-npm/
└── jfvm-docker/
```

### Package Repositories
- **NPM**: `@jfrog/jfvm`
- **Chocolatey**: `jfvm`
- **Docker Hub**: `jfrog/jfvm`
- **APT**: Custom JFrog repository
- **YUM**: Custom JFrog repository

## Installation Methods

### One-liner Install Scripts

#### Unix/Linux/macOS
```bash
# Download only
curl -fL https://get.jfrog.io/jfvm | sh

# Install with JFrog CLI
curl -fL https://install.jfrog.io/jfvm | sh
```

#### Windows PowerShell
```powershell
# Download only
iwr -useb https://get.jfrog.io/jfvm.ps1 | iex

# Install with JFrog CLI
iwr -useb https://install.jfrog.io/jfvm.ps1 | iex
```

### Package Managers

#### NPM
```bash
npm install -g @jfrog/jfvm
# With JFrog CLI option during install
npm install -g @jfrog/jfvm --install-with-jfrog-cli
```

#### Chocolatey
```batch
choco install jfvm
# With JFrog CLI option
choco install jfvm --params '/InstallJfrogCli'
```

#### Homebrew
```bash
brew tap jfrog/tap
brew install jfvm
```

#### Debian/Ubuntu
```bash
curl -fsSL https://releases.jfrog.io/artifactory/jfvm-deb/jfvm.gpg | sudo apt-key add -
echo "deb https://releases.jfrog.io/artifactory/jfvm-deb stable main" | sudo tee /etc/apt/sources.list.d/jfvm.list
sudo apt-get update
sudo apt-get install jfvm
```

#### RHEL/CentOS/Fedora
```bash
curl -fsSL https://releases.jfrog.io/artifactory/jfvm-rpm/jfvm.gpg | sudo rpm --import -
sudo tee /etc/yum.repos.d/jfvm.repo << EOF
[jfvm]
name=JFVM Repository
baseurl=https://releases.jfrog.io/artifactory/jfvm-rpm
enabled=1
gpgcheck=1
gpgkey=https://releases.jfrog.io/artifactory/jfvm-rpm/jfvm.gpg
EOF
sudo yum install jfvm
```

## JFrog CLI Integration Strategy

### 1. Optional Installation
All installation methods include an **unchecked by default** option to install JFrog CLI alongside JFVM:

- **Interactive prompts** with clear documentation links
- **Environment variables** for automation
- **Command-line flags** for scripted installations
- **Package parameters** for package managers

### 2. Cross-Promotion
- JFVM installations promote JFrog CLI as optional
- JFrog CLI installations will promote JFVM as optional (future enhancement)
- Documentation cross-references both tools

### 3. Environment Variables

#### For Silent Automation
```bash
# Install both JFVM and JFrog CLI without prompts
JFVM_INSTALL_JFROG_CLI=true JFVM_SILENT_INSTALL=true curl -fL https://install.jfrog.io/jfvm | sh

# Install only JFVM
JFVM_INSTALL_JFROG_CLI=false JFVM_SILENT_INSTALL=true curl -fL https://install.jfrog.io/jfvm | sh
```

## Certificate Requirements

To use this build system in production, you need access to:

1. **Windows Signing Certificate** (DigiCert)
2. **Apple Developer Certificates** for macOS signing and notarization
3. **GPG Keys** for Linux package signing
4. **Artifactory Credentials** for publishing
5. **Package Repository Credentials** (NPM, Chocolatey, etc.)

## Jenkins Configuration

The pipeline requires these credential IDs in Jenkins:
- `repo21` - Artifactory credentials
- `windows-signing-cert` - Windows signing certificate
- `apple-team-id` - Apple Developer Team ID
- `apple-account-id` - Apple ID for notarization
- `apple-app-password` - App-specific password
- `rpm-gpg-key3` - RPM signing GPG key
- `choco-api-key` - Chocolatey API key
- `npm-token` - NPM publishing token
- `docker-hub` - Docker Hub credentials

## Testing

Each package type includes verification steps:
- Binary execution tests
- Package installation tests
- Integration tests with JFrog CLI
- Cross-platform compatibility tests

## Monitoring and Analytics

- Track installation method preferences
- Monitor JFVM adoption rates through different channels
- Collect feedback on installation experience
- A/B test different prompt strategies for JFrog CLI integration

This build system provides a comprehensive, production-ready infrastructure for distributing JFVM across all major platforms while strategically promoting JFrog CLI adoption through optional, user-friendly integration points.



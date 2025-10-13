#!/bin/bash
# install.sh - Complete JFVM Jenkins setup installer
set -euo pipefail

echo "🔧 JFVM Jenkins Development Environment Installer"
echo "================================================="

# Function to check command availability
check_command() {
    if command -v "$1" &> /dev/null; then
        echo "✅ $1 is available"
        return 0
    else
        echo "❌ $1 is not installed"
        return 1
    fi
}

# Function to install Docker on macOS
install_docker_macos() {
    echo "Installing Docker Desktop for macOS..."
    if command -v brew &> /dev/null; then
        brew install --cask docker
        echo "✅ Docker Desktop installed via Homebrew"
        echo "⚠️  Please start Docker Desktop manually and then re-run this script"
    else
        echo "Please install Docker Desktop manually from: https://docs.docker.com/desktop/mac/install/"
    fi
    exit 1
}

# Function to install Docker on Linux
install_docker_linux() {
    echo "Installing Docker on Linux..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
    echo "✅ Docker installed"
    echo "⚠️  Please log out and back in, then re-run this script"
    exit 1
}

# Check prerequisites
echo "1. Checking prerequisites..."

# Check Docker
if ! check_command docker; then
    echo "Docker is required but not installed."
    echo "Would you like to install it? [y/N]"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            install_docker_macos
        elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
            install_docker_linux
        else
            echo "Please install Docker manually for your platform"
            exit 1
        fi
    else
        echo "Docker is required. Please install it and re-run this script."
        exit 1
    fi
fi

# Check Docker Compose
DOCKER_COMPOSE="docker-compose"
if ! check_command docker-compose; then
    if check_command "docker compose"; then
        DOCKER_COMPOSE="docker compose"
        echo "✅ Using 'docker compose' (newer syntax)"
    else
        echo "Installing Docker Compose..."
        if [[ "$OSTYPE" == "darwin"* ]] && command -v brew &> /dev/null; then
            brew install docker-compose
        else
            sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
            sudo chmod +x /usr/local/bin/docker-compose
        fi
        echo "✅ Docker Compose installed"
    fi
fi

# Check if Docker is running
echo "Checking if Docker daemon is running..."
if ! docker info &> /dev/null; then
    echo "❌ Docker daemon is not running"
    echo "Please start Docker and re-run this script"
    exit 1
fi
echo "✅ Docker daemon is running"

# Navigate to setup directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo
echo "2. Environment Configuration"

# Check for existing Artifactory
echo "Checking for existing Artifactory instance..."
if curl -f -s http://localhost:8082/artifactory/api/system/ping > /dev/null 2>&1; then
    echo "✅ Artifactory detected at http://localhost:8082"
    echo "The setup will connect to your existing Artifactory instance."
    USE_EXISTING_ARTIFACTORY=true
else
    echo "ℹ️  No Artifactory detected at http://localhost:8082"
    echo "The setup will start a new Artifactory container."
    USE_EXISTING_ARTIFACTORY=false
fi

echo
echo "3. Starting Jenkins Environment..."

# Make scripts executable
chmod +x *.sh

# Start the environment
if ./start.sh; then
    echo
    echo "🎉 Installation completed successfully!"
    echo
    echo "📋 What's Available:"
    echo "  🔧 Jenkins Web UI:  http://localhost:8080"
    echo "  📦 Artifactory UI:  http://localhost:8082"
    echo "  🔑 Jenkins Login:   admin / admin123"
    echo "  🔑 Artifactory:     admin / password"
    echo
    echo "🚀 Quick Start:"
    echo "  1. Open Jenkins: http://localhost:8080"
    echo "  2. Create New Item → Pipeline"
    echo "  3. Pipeline Script from SCM:"
    echo "     - Repository URL: file://${PWD}/.."
    echo "     - Script Path: Jenkinsfile.local"
    echo "  4. Build with Parameters to customize your build"
    echo
    echo "📖 Repository Structure:"
    echo "  Artifacts will be published to:"
    echo "  jfvm-binaries/jfvm/v1/{version}/jfvm-{os}-{arch}/"
    echo
    echo "🛑 To stop environment: cd jenkins-setup && ./stop.sh"
else
    echo "❌ Installation failed. Check the logs above for details."
    exit 1
fi

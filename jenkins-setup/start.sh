#!/bin/bash
# start.sh - Start Jenkins and Artifactory with proper networking
set -euo pipefail

echo "🚀 Starting JFVM Jenkins Development Environment..."

# Check prerequisites
echo "Checking prerequisites..."
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "✅ Prerequisites satisfied"

# Determine Docker Compose command
DOCKER_COMPOSE="docker-compose"
if command -v docker compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
fi

# Navigate to jenkins-setup directory
cd "$(dirname "$0")"

echo "📁 Working directory: $(pwd)"

# Check existing Artifactory
echo "🔍 Checking existing Artifactory at http://localhost:8082..."
if curl -f -s http://localhost:8082/artifactory/api/system/ping > /dev/null 2>&1; then
    echo "✅ Found existing Artifactory at http://localhost:8082"
    echo "Jenkins will connect to your existing Artifactory instance"
else
    echo "❌ No Artifactory found at http://localhost:8082"
    echo "Please ensure your Artifactory is running and accessible"
    exit 1
fi

# Stop any existing Jenkins containers
echo "🛑 Stopping any existing Jenkins containers..."
$DOCKER_COMPOSE down 2>/dev/null || true

# Build Jenkins image
echo "🔧 Building Jenkins image..."
$DOCKER_COMPOSE build --no-cache jenkins

# Start Jenkins
echo "🚀 Starting Jenkins..."
$DOCKER_COMPOSE up -d jenkins

# Wait for services to be healthy
echo "⏳ Waiting for services to start..."

# Wait for Artifactory (existing or containerized)
echo "Waiting for Artifactory to be ready..."
timeout=300
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if curl -f -s http://localhost:8082/artifactory/api/system/ping > /dev/null 2>&1; then
        echo "✅ Artifactory is ready at http://localhost:8082"
        break
    fi
    sleep 5
    elapsed=$((elapsed + 5))
    echo "  Waiting... (${elapsed}s/${timeout}s)"
done

if [ $elapsed -ge $timeout ]; then
    echo "❌ Artifactory not accessible within ${timeout} seconds"
    echo "Please ensure your existing Artifactory is running and accessible at http://localhost:8082"
    exit 1
fi

# Wait for Jenkins
echo "Waiting for Jenkins to be ready..."
timeout=300
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if curl -f -s http://localhost:8080/login > /dev/null 2>&1; then
        echo "✅ Jenkins is ready"
        break
    fi
    sleep 5
    elapsed=$((elapsed + 5))
    echo "  Waiting... (${elapsed}s/${timeout}s)"
done

if [ $elapsed -ge $timeout ]; then
    echo "❌ Jenkins failed to start within ${timeout} seconds"
    echo "Checking Jenkins logs:"
    $DOCKER_COMPOSE logs jenkins | tail -20
    exit 1
fi

# Create repositories
echo "🏗️ Creating Artifactory repositories..."
if ./create-repositories.sh http://localhost:8082 admin password; then
    echo "✅ Repositories created successfully"
else
    echo "⚠️ Repository creation had issues"
    echo "You may need to:"
    echo "  1. Check Artifactory credentials (default: admin/password)"
    echo "  2. Create repositories manually via Artifactory UI"
    echo "  3. Ensure Artifactory is fully initialized"
fi

echo
echo "🎉 JFVM Jenkins Development Environment is ready!"
echo
echo "📋 Access Information:"
echo "  🔧 Jenkins:     http://localhost:8080"
echo "  📦 Artifactory: http://localhost:8082"
echo
echo "🔑 Default Credentials:"
echo "  Jenkins:     admin / password"
echo "  Artifactory: admin / password"
echo
echo "🚀 Next Steps:"
echo "  1. Open Jenkins at http://localhost:8080"
echo "  2. Create a new Pipeline job"
echo "  3. Pipeline Script from SCM:"
echo "     - Repository URL: file://$(realpath ..)"
echo "     - Script Path: Jenkinsfile.local"
echo "  4. Build with Parameters to customize your build!"
echo
echo "📦 Artifacts will be published to:"
echo "  http://localhost:8082/ui/repos/tree/General/jfvm-binaries"
echo
echo "🛑 To stop: cd $(pwd) && ./stop.sh"

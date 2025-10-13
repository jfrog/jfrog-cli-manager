#!/bin/bash
# cleanup.sh - Complete cleanup of Jenkins development environment
set -euo pipefail

echo "🗑️ JFVM Jenkins Environment Cleanup"
echo "==================================="
echo
echo "⚠️  WARNING: This will permanently delete:"
echo "  • All Jenkins jobs and configuration"
echo "  • All Artifactory repositories and artifacts"
echo "  • All Docker containers and volumes"
echo
echo "This action cannot be undone!"
echo
read -p "Are you sure you want to continue? [y/N]: " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

cd "$(dirname "$0")"

# Determine Docker Compose command
DOCKER_COMPOSE="docker-compose"
if command -v docker compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
fi

echo "🛑 Stopping and removing containers..."
$DOCKER_COMPOSE down --volumes --remove-orphans

echo "🗑️ Removing Docker volumes..."
docker volume rm jenkins-setup_jenkins_home 2>/dev/null || echo "Jenkins volume already removed"
docker volume rm jenkins-setup_artifactory_data 2>/dev/null || echo "Artifactory volume already removed"

echo "🧹 Cleaning up Docker networks..."
docker network rm jenkins-setup_jfvm-network 2>/dev/null || echo "Network already removed"

echo "🧹 Removing unused Docker resources..."
docker system prune -f 2>/dev/null || true

echo "🗑️ Cleaning up temporary files..."
rm -rf ../dist/ 2>/dev/null || true
rm -rf ../test-results/ 2>/dev/null || true

echo
echo "✅ Cleanup completed successfully!"
echo
echo "All Jenkins jobs, Artifactory data, and Docker resources have been removed."
echo "To start fresh: ./install.sh"

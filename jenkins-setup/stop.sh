#!/bin/bash
# stop.sh - Stop Jenkins development environment
set -euo pipefail

echo "🛑 Stopping JFVM Jenkins Development Environment..."

cd "$(dirname "$0")"

# Determine Docker Compose command
DOCKER_COMPOSE="docker-compose"
if command -v docker compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
fi

# Stop and remove containers
echo "Stopping containers..."
$DOCKER_COMPOSE down

echo "Removing unused networks..."
docker network prune -f 2>/dev/null || true

echo "✅ Environment stopped successfully"
echo
echo "💾 Data preserved in Docker volumes:"
echo "  • jenkins_home (Jenkins configuration and jobs)"
echo "  • artifactory_data (Artifactory repositories and artifacts)"
echo
echo "🔄 To restart: ./start.sh"
echo "🗑️  To completely remove (including data): ./cleanup.sh"

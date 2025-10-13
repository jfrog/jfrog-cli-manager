#!/bin/bash
# create-repositories.sh - Create Artifactory repositories for JFVM
set -euo pipefail

ARTIFACTORY_URL="${1:-http://localhost:8082}"
ARTIFACTORY_USER="${2:-admin}"
ARTIFACTORY_PASSWORD="${3:-password}"

echo "🏗️ Creating Artifactory repositories for JFVM..."
echo "Artifactory URL: $ARTIFACTORY_URL"

# Test connectivity
echo "Testing Artifactory connectivity..."
if ! curl -f -s -u "$ARTIFACTORY_USER:$ARTIFACTORY_PASSWORD" "$ARTIFACTORY_URL/artifactory/api/system/ping" > /dev/null; then
    echo "❌ Cannot connect to Artifactory at $ARTIFACTORY_URL"
    exit 1
fi
echo "✅ Connected to Artifactory"

# Function to create repository
create_repository() {
    local repo_key="$1"
    local repo_type="$2"
    local description="$3"
    local layout="$4"
    
    echo "Creating repository: $repo_key ($repo_type)"
    
    # Check if repository exists
    if curl -s -u "$ARTIFACTORY_USER:$ARTIFACTORY_PASSWORD" \
        "$ARTIFACTORY_URL/artifactory/api/repositories/$repo_key" \
        -w "%{http_code}" -o /dev/null | grep -q "200"; then
        echo "✅ Repository $repo_key already exists"
        return
    fi
    
    # Create repository configuration
    local config=""
    case "$repo_type" in
        "generic")
            config='{
                "key": "'$repo_key'",
                "rclass": "local",
                "packageType": "generic",
                "description": "'$description'",
                "repoLayoutRef": "'$layout'",
                "checksumPolicyType": "client-checksums",
                "handleReleases": true,
                "handleSnapshots": true,
                "maxUniqueSnapshots": 0,
                "suppressPomConsistencyChecks": false,
                "blackedOut": false,
                "archiveBrowsingEnabled": true,
                "calculateYumMetadata": false,
                "yumRootDepth": 0
            }'
            ;;
        "docker")
            config='{
                "key": "'$repo_key'",
                "rclass": "local",
                "packageType": "docker",
                "description": "'$description'",
                "repoLayoutRef": "'$layout'",
                "dockerApiVersion": "V2",
                "maxUniqueTags": 0,
                "blockPushingSchema1": true,
                "checksumPolicyType": "client-checksums"
            }'
            ;;
        "npm")
            config='{
                "key": "'$repo_key'",
                "rclass": "local",
                "packageType": "npm",
                "description": "'$description'",
                "repoLayoutRef": "'$layout'",
                "checksumPolicyType": "client-checksums"
            }'
            ;;
    esac
    
    # Create repository
    if curl -X PUT \
        -H "Content-Type: application/json" \
        -u "$ARTIFACTORY_USER:$ARTIFACTORY_PASSWORD" \
        "$ARTIFACTORY_URL/artifactory/api/repositories/$repo_key" \
        -d "$config" \
        --fail --silent; then
        echo "✅ Created repository: $repo_key"
    else
        echo "❌ Failed to create repository: $repo_key"
        return 1
    fi
}

# Create repositories following JFrog CLI structure
echo
echo "Creating repositories..."

create_repository "jfvm-binaries" "generic" "JFVM binary artifacts organized by version and platform" "simple-default"
create_repository "jfvm-docker" "docker" "JFVM Docker test images" "docker-default"
create_repository "jfvm-npm" "npm" "JFVM NPM packages for future use" "npm-default"

echo
echo "🎉 Repository creation completed!"
echo
echo "📋 Created repositories:"
echo "  • jfvm-binaries - Binary artifacts (http://localhost:8082/ui/repos/tree/General/jfvm-binaries)"
echo "  • jfvm-docker - Docker images (http://localhost:8082/ui/repos/tree/General/jfvm-docker)"
echo "  • jfvm-npm - NPM packages (http://localhost:8082/ui/repos/tree/General/jfvm-npm)"
echo
echo "🔗 Repository structure will follow:"
echo "  jfvm-binaries/jfvm/v1/{version}/jfvm-{os}-{arch}/jfvm[.exe]"
echo
echo "✅ Ready for JFVM artifact publishing!"

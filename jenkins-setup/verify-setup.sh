#!/bin/bash
# verify-setup.sh - Verify Jenkins and Artifactory setup
set -euo pipefail

echo "🔍 Verifying JFVM Jenkins Development Environment..."

# Check Jenkins
echo "1. Checking Jenkins..."
if curl -f -s http://localhost:8080/login > /dev/null 2>&1; then
    echo "✅ Jenkins is accessible at http://localhost:8080"
else
    echo "❌ Jenkins is not accessible"
    exit 1
fi

# Check Artifactory
echo "2. Checking Artifactory..."
if curl -f -s http://localhost:8082/artifactory/api/system/ping > /dev/null 2>&1; then
    echo "✅ Artifactory is accessible at http://localhost:8082"
else
    echo "❌ Artifactory is not accessible"
    exit 1
fi

# Check repositories
echo "3. Checking JFVM repositories..."
if curl -f -s -u admin:password http://localhost:8082/artifactory/api/repositories/jfvm-binaries > /dev/null 2>&1; then
    echo "✅ jfvm-binaries repository exists"
else
    echo "⚠️ jfvm-binaries repository not found (will be created during first build)"
fi

# Test Jenkins authentication
echo "4. Testing Jenkins authentication..."
if curl -f -s -u admin:password http://localhost:8080/api/json > /dev/null 2>&1; then
    echo "✅ Jenkins authentication works (admin/password)"
else
    echo "❌ Jenkins authentication failed"
fi

echo
echo "🎉 Setup verification completed!"
echo
echo "📋 Ready to use:"
echo "  🔧 Jenkins:     http://localhost:8080 (admin/password)"
echo "  📦 Artifactory: http://localhost:8082 (admin/password)"
echo
echo "🚀 Create your first JFVM pipeline:"
echo "  1. Open http://localhost:8080"
echo "  2. New Item → Pipeline"
echo "  3. Name: 'JFVM-Build'"
echo "  4. Pipeline Script from SCM:"
echo "     - SCM: Git"
echo "     - Repository URL: file://$(realpath ..)"
echo "     - Script Path: Jenkinsfile.local"
echo "  5. Save and Build with Parameters!"

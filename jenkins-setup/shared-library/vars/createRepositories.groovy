def call() {
    echo "🏗️ Setting up Artifactory repositories..."
    
    def artifactoryManager = new org.jfrog.jfvm.ArtifactoryManager(this)
    
    // Define repositories to create
    def repositories = [
        [
            key: 'jfvm-binaries',
            type: 'generic',
            description: 'JFVM binary artifacts',
            layout: 'simple-default'
        ],
        [
            key: 'jfvm-docker',
            type: 'docker',
            description: 'JFVM Docker test images',
            layout: 'docker-default'
        ],
        [
            key: 'jfvm-npm',
            type: 'npm',
            description: 'JFVM NPM packages',
            layout: 'npm-default'
        ]
    ]
    
    repositories.each { repo ->
        try {
            if (artifactoryManager.repositoryExists(repo.key)) {
                echo "✅ Repository ${repo.key} already exists"
            } else {
                artifactoryManager.createRepository(repo)
                echo "✅ Created repository: ${repo.key}"
            }
        } catch (Exception e) {
            echo "⚠️ Warning: Failed to create repository ${repo.key}: ${e.getMessage()}"
            echo "Repository may need to be created manually"
        }
    }
    
    echo "✅ Repository setup completed"
}

def createGenericRepository(key, description) {
    sh """
        curl -X PUT \\
            -H "Content-Type: application/json" \\
            -u admin:password \\
            "${env.ARTIFACTORY_URL}/artifactory/api/repositories/${key}" \\
            -d '{
                "key": "${key}",
                "rclass": "local",
                "packageType": "generic",
                "description": "${description}",
                "repoLayoutRef": "simple-default",
                "checksumPolicyType": "client-checksums",
                "handleReleases": true,
                "handleSnapshots": true,
                "maxUniqueSnapshots": 0,
                "suppressPomConsistencyChecks": false,
                "blackedOut": false,
                "propertySets": [],
                "archiveBrowsingEnabled": false
            }' || echo "Repository creation may have failed"
    """
}

def createDockerRepository(key, description) {
    sh """
        curl -X PUT \\
            -H "Content-Type: application/json" \\
            -u admin:password \\
            "${env.ARTIFACTORY_URL}/artifactory/api/repositories/${key}" \\
            -d '{
                "key": "${key}",
                "rclass": "local",
                "packageType": "docker",
                "description": "${description}",
                "repoLayoutRef": "docker-default",
                "dockerApiVersion": "V2",
                "maxUniqueTags": 0,
                "blockPushingSchema1": true,
                "checksumPolicyType": "client-checksums"
            }' || echo "Repository creation may have failed"
    """
}

def createNpmRepository(key, description) {
    sh """
        curl -X PUT \\
            -H "Content-Type: application/json" \\
            -u admin:password \\
            "${env.ARTIFACTORY_URL}/artifactory/api/repositories/${key}" \\
            -d '{
                "key": "${key}",
                "rclass": "local",
                "packageType": "npm",
                "description": "${description}",
                "repoLayoutRef": "npm-default",
                "checksumPolicyType": "client-checksums"
            }' || echo "Repository creation may have failed"
    """
}

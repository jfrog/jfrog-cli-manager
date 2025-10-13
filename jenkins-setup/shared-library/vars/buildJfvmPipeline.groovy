def call(Map config) {
    pipeline {
        agent any
        
        parameters {
            choice(
                name: 'BUILD_TYPE',
                choices: ['dev', 'release', 'pr'],
                description: 'Type of build to execute'
            )
            string(
                name: 'VERSION',
                defaultValue: '',
                description: 'Version to build (auto-detected if empty)'
            )
            booleanParam(
                name: 'SKIP_TESTS',
                defaultValue: false,
                description: 'Skip cross-platform testing'
            )
            choice(
                name: 'ARCHITECTURES',
                choices: ['all', 'linux-only', 'darwin-only', 'windows-only'],
                description: 'Architectures to build'
            )
        }
        
        environment {
            ARTIFACTORY_URL = "${env.ARTIFACTORY_URL ?: 'http://artifactory:8082'}"
            ARTIFACTORY_FALLBACK_URL = "${env.ARTIFACTORY_FALLBACK_URL ?: 'http://172.20.0.10:8082'}"
            GO_VERSION = "1.21.5"
            BUILD_NAME = "jfvm-multi-platform"
            BUILD_NUMBER = "${env.BUILD_NUMBER}"
        }
        
        stages {
            stage('Environment Setup') {
                steps {
                    script {
                        // Initialize build configuration
                        def buildConfig = new org.jfrog.jfvm.BuildConfig(this)
                        env.JFVM_VERSION = buildConfig.determineVersion(params.VERSION)
                        env.PUBLISH_TO_PROD = buildConfig.shouldPublishToProd(env.BRANCH_NAME, env.TAG_NAME)
                        
                        // Setup build environment
                        setupBuildEnvironment()
                        
                        // Health check Artifactory
                        sh '''
                            echo "Checking Artifactory connectivity..."
                            if curl -f -s "${ARTIFACTORY_URL}/artifactory/api/system/ping" > /dev/null; then
                                echo "✅ Artifactory accessible at ${ARTIFACTORY_URL}"
                                export ARTIFACTORY_ACTIVE_URL="${ARTIFACTORY_URL}/artifactory"
                            elif curl -f -s "${ARTIFACTORY_FALLBACK_URL}/artifactory/api/system/ping" > /dev/null; then
                                echo "✅ Artifactory accessible at ${ARTIFACTORY_FALLBACK_URL}"
                                export ARTIFACTORY_ACTIVE_URL="${ARTIFACTORY_FALLBACK_URL}/artifactory"
                            else
                                echo "❌ Artifactory not accessible"
                                exit 1
                            fi
                        '''
                    }
                }
            }
            
            stage('Repository Setup') {
                steps {
                    script {
                        createRepositories()
                    }
                }
            }
            
            stage('Build Binaries') {
                steps {
                    script {
                        def architectures = getArchitectures(params.ARCHITECTURES)
                        buildBinaries(architectures, env.JFVM_VERSION)
                    }
                }
            }
            
            stage('Publish Artifacts') {
                steps {
                    script {
                        publishArtifacts(env.JFVM_VERSION, env.BUILD_NAME, env.BUILD_NUMBER)
                    }
                }
            }
            
            stage('Cross-Platform Testing') {
                when {
                    not { params.SKIP_TESTS }
                }
                steps {
                    script {
                        testBinaries(env.JFVM_VERSION)
                    }
                }
                post {
                    always {
                        publishTestResults(
                            testResultsPattern: 'test-results/*.xml',
                            allowEmptyResults: true
                        )
                    }
                }
            }
        }
        
        post {
            always {
                script {
                    // Publish build info
                    publishBuildInfo(env.BUILD_NAME, env.BUILD_NUMBER)
                }
                
                // Archive artifacts
                archiveArtifacts(
                    artifacts: 'dist/**/*',
                    allowEmptyArchive: true,
                    fingerprint: true
                )
                
                // Clean workspace
                cleanWs()
            }
            success {
                echo "✅ JFVM build completed successfully!"
            }
            failure {
                echo "❌ JFVM build failed!"
            }
        }
    }
}

def setupBuildEnvironment() {
    sh '''
        # Verify Go installation
        go version
        
        # Create directory structure
        mkdir -p dist/{binaries,packages,signed}
        mkdir -p test-results
        
        # Download dependencies
        go mod download
        go mod verify
        
        echo "✅ Build environment ready"
    '''
}

def getArchitectures(selection) {
    def allArchitectures = [
        [pkg: 'jfvm-linux-amd64', goos: 'linux', goarch: 'amd64', fileExtension: ''],
        [pkg: 'jfvm-linux-arm64', goos: 'linux', goarch: 'arm64', fileExtension: ''],
        [pkg: 'jfvm-linux-386', goos: 'linux', goarch: '386', fileExtension: ''],
        [pkg: 'jfvm-linux-arm', goos: 'linux', goarch: 'arm', fileExtension: ''],
        [pkg: 'jfvm-linux-s390x', goos: 'linux', goarch: 's390x', fileExtension: ''],
        [pkg: 'jfvm-linux-ppc64le', goos: 'linux', goarch: 'ppc64le', fileExtension: ''],
        [pkg: 'jfvm-darwin-amd64', goos: 'darwin', goarch: 'amd64', fileExtension: ''],
        [pkg: 'jfvm-darwin-arm64', goos: 'darwin', goarch: 'arm64', fileExtension: ''],
        [pkg: 'jfvm-windows-amd64', goos: 'windows', goarch: 'amd64', fileExtension: '.exe'],
        [pkg: 'jfvm-freebsd-amd64', goos: 'freebsd', goarch: 'amd64', fileExtension: '']
    ]
    
    switch(selection) {
        case 'linux-only':
            return allArchitectures.findAll { it.goos == 'linux' }
        case 'darwin-only':
            return allArchitectures.findAll { it.goos == 'darwin' }
        case 'windows-only':
            return allArchitectures.findAll { it.goos == 'windows' }
        default:
            return allArchitectures
    }
}

def publishBuildInfo(buildName, buildNumber) {
    try {
        sh """
            echo "Publishing build info for ${buildName} #${buildNumber}"
            # Build info publishing logic would go here
            # This is a placeholder for Artifactory build info API calls
        """
    } catch (Exception e) {
        echo "Warning: Failed to publish build info: ${e.getMessage()}"
    }
}

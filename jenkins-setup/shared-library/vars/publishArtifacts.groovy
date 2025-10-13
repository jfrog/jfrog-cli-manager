def call(version, buildName, buildNumber) {
    echo "📤 Publishing artifacts to Artifactory for version ${version}"
    
    def artifactoryManager = new org.jfrog.jfvm.ArtifactoryManager(this)
    
    // Get all built binaries
    def binaries = findBinaries()
    
    if (binaries.isEmpty()) {
        error "No binaries found to publish"
    }
    
    echo "Found ${binaries.size()} binaries to publish"
    
    // Create parallel upload steps
    def uploadSteps = [:]
    
    binaries.each { binary ->
        uploadSteps["Upload ${binary.pkg}"] = {
            uploadBinary(binary, version, artifactoryManager)
        }
    }
    
    // Upload all binaries in parallel
    parallel uploadSteps
    
    // Set build properties
    setBuildProperties(version, buildName, buildNumber)
    
    echo "✅ All artifacts published successfully"
}

def findBinaries() {
    def binaries = []
    
    // Find all binary files
    def binaryDirs = sh(
        script: 'find dist/binaries -type d -name "jfvm-*" | sort',
        returnStdout: true
    ).trim().split('\n')
    
    binaryDirs.each { dir ->
        if (dir?.trim()) {
            def pkg = dir.split('/').last()
            def files = sh(
                script: "find ${dir} -type f -name 'jfvm*' ! -name '*.sha256' | head -1",
                returnStdout: true
            ).trim()
            
            if (files) {
                def fileName = files.split('/').last()
                def fileExtension = fileName.endsWith('.exe') ? '.exe' : ''
                def goos = pkg.contains('windows') ? 'windows' : 
                          pkg.contains('darwin') ? 'darwin' : 
                          pkg.contains('linux') ? 'linux' : 
                          pkg.contains('freebsd') ? 'freebsd' : 'unknown'
                def goarch = pkg.contains('amd64') ? 'amd64' :
                            pkg.contains('arm64') ? 'arm64' :
                            pkg.contains('386') ? '386' :
                            pkg.contains('arm') ? 'arm' :
                            pkg.contains('s390x') ? 's390x' :
                            pkg.contains('ppc64le') ? 'ppc64le' : 'unknown'
                
                binaries.add([
                    pkg: pkg,
                    goos: goos,
                    goarch: goarch,
                    fileName: fileName,
                    fileExtension: fileExtension,
                    filePath: files,
                    checksumPath: "${files}.sha256"
                ])
            }
        }
    }
    
    return binaries
}

def uploadBinary(binary, version, artifactoryManager) {
    def uploadPath = "jfvm/v1/${version}/${binary.pkg}/${binary.fileName}"
    def checksumPath = "jfvm/v1/${version}/${binary.pkg}/${binary.fileName}.sha256"
    
    echo "Uploading ${binary.pkg}/${binary.fileName} to ${uploadPath}"
    
    try {
        // Upload binary
        artifactoryManager.uploadFile(
            binary.filePath,
            'jfvm-binaries',
            uploadPath
        )
        
        // Upload checksum
        if (fileExists(binary.checksumPath)) {
            artifactoryManager.uploadFile(
                binary.checksumPath,
                'jfvm-binaries',
                checksumPath
            )
        }
        
        echo "✅ Uploaded ${binary.pkg}/${binary.fileName}"
        
    } catch (Exception e) {
        echo "❌ Failed to upload ${binary.pkg}: ${e.getMessage()}"
        throw e
    }
}

def setBuildProperties(version, buildName, buildNumber) {
    sh """
        # Set build properties for traceability
        echo "Setting build properties..."
        echo "Version: ${version}"
        echo "Build Name: ${buildName}"
        echo "Build Number: ${buildNumber}"
        echo "Git Commit: \$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
        echo "Build Date: \$(date -u '+%Y-%m-%d_%H:%M:%S')"
    """
}

def retryUpload(closure, maxRetries = 3) {
    def attempt = 1
    while (attempt <= maxRetries) {
        try {
            closure()
            return
        } catch (Exception e) {
            if (attempt == maxRetries) {
                throw e
            }
            echo "Upload attempt ${attempt} failed, retrying... (${e.getMessage()})"
            sleep(time: attempt * 5, unit: 'SECONDS')
            attempt++
        }
    }
}

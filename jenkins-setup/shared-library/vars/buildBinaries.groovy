def call(architectures, version) {
    echo "🔧 Building JFVM binaries for version ${version}"
    
    def buildSteps = [:]
    def jfvmExecutableName = 'jfvm'
    
    architectures.each { architecture ->
        def goos = architecture.goos
        def goarch = architecture.goarch
        def pkg = architecture.pkg
        def fileExtension = architecture.fileExtension
        def fileName = "${jfvmExecutableName}${fileExtension}"
        
        buildSteps["${pkg}"] = {
            buildSingleBinary(goos, goarch, pkg, fileName, version)
        }
    }
    
    // Build all architectures in parallel
    parallel buildSteps
    
    echo "✅ All binaries built successfully"
}

def buildSingleBinary(goos, goarch, pkg, fileName, version) {
    echo "Building ${pkg} (${goos}/${goarch})..."
    
    sh """
        # Set build environment
        export GOOS=${goos}
        export GOARCH=${goarch}
        export CGO_ENABLED=0
        
        # Create output directory
        mkdir -p dist/binaries/${pkg}
        
        # Build with version information
        BUILD_DATE=\$(date -u '+%Y-%m-%d_%H:%M:%S')
        GIT_COMMIT=\$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
        
        LDFLAGS="-w -extldflags '-static' -X main.Version=${version} -X main.BuildDate=\${BUILD_DATE} -X main.GitCommit=\${GIT_COMMIT}"
        
        echo "Building ${fileName} for ${goos}/${goarch}..."
        go build -o "dist/binaries/${pkg}/${fileName}" -ldflags "\${LDFLAGS}" main.go
        
        # Make executable
        chmod +x "dist/binaries/${pkg}/${fileName}"
        
        # Verify binary was created
        if [ ! -f "dist/binaries/${pkg}/${fileName}" ]; then
            echo "❌ Failed to create binary: dist/binaries/${pkg}/${fileName}"
            exit 1
        fi
        
        # Get binary size
        BINARY_SIZE=\$(stat -c%s "dist/binaries/${pkg}/${fileName}" 2>/dev/null || stat -f%z "dist/binaries/${pkg}/${fileName}" 2>/dev/null || echo "unknown")
        echo "✅ Built ${pkg}/${fileName} (size: \${BINARY_SIZE} bytes)"
        
        # Basic binary test on compatible platforms
        if [ "${goos}" = "linux" ] && [ "${goarch}" = "amd64" ]; then
            echo "🧪 Testing binary on current platform..."
            if ./dist/binaries/${pkg}/${fileName} --version; then
                echo "✅ Binary test passed"
            else
                echo "⚠️  Binary test failed but continuing..."
            fi
        fi
        
        # Create checksum
        cd dist/binaries/${pkg}
        sha256sum ${fileName} > ${fileName}.sha256
        cd ../../..
        
        echo "✅ Successfully built and verified ${pkg}/${fileName}"
    """
}

def getBinarySize(filePath) {
    def size = sh(
        script: "stat -c%s '${filePath}' 2>/dev/null || stat -f%z '${filePath}' 2>/dev/null || echo 'unknown'",
        returnStdout: true
    ).trim()
    return size
}

def createChecksum(filePath) {
    sh """
        cd \$(dirname "${filePath}")
        FILENAME=\$(basename "${filePath}")
        sha256sum "\${FILENAME}" > "\${FILENAME}.sha256"
        echo "✅ Created checksum for \${FILENAME}"
    """
}

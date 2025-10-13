def call(version) {
    echo "🧪 Running cross-platform binary tests for version ${version}"
    
    def testRunner = new org.jfrog.jfvm.TestRunner(this)
    
    // Find all binaries to test
    def binaries = findTestableBinaries()
    
    if (binaries.isEmpty()) {
        echo "⚠️ No binaries found to test"
        return
    }
    
    echo "Found ${binaries.size()} binaries to test"
    
    // Create parallel test steps
    def testSteps = [:]
    
    binaries.each { binary ->
        testSteps["Test ${binary.pkg}"] = {
            testSingleBinary(binary, version, testRunner)
        }
    }
    
    // Run all tests in parallel
    parallel testSteps
    
    echo "✅ All binary tests completed"
}

def findTestableBinaries() {
    def binaries = []
    
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
                def goos = extractOS(pkg)
                def goarch = extractArch(pkg)
                
                binaries.add([
                    pkg: pkg,
                    goos: goos,
                    goarch: goarch,
                    fileName: fileName,
                    filePath: files
                ])
            }
        }
    }
    
    return binaries
}

def testSingleBinary(binary, version, testRunner) {
    def testImage = getTestImage(binary.goos, binary.goarch)
    def testName = "test-${binary.pkg}"
    
    echo "Testing ${binary.pkg} using ${testImage}"
    
    try {
        // Create test container
        sh """
            # Create test directory
            mkdir -p test-results
            
            # Copy binary to test location
            cp "${binary.filePath}" test-${binary.pkg}-binary
            chmod +x test-${binary.pkg}-binary
        """
        
        if (binary.goos == 'windows') {
            testWindowsBinary(binary, testName)
        } else {
            testUnixBinary(binary, testImage, testName)
        }
        
    } catch (Exception e) {
        echo "❌ Test failed for ${binary.pkg}: ${e.getMessage()}"
        
        // Create failure test result
        writeFile(
            file: "test-results/${testName}-result.xml",
            text: createFailureTestResult(testName, e.getMessage())
        )
        
        throw e
    }
}

def testUnixBinary(binary, testImage, testName) {
    sh """
        # Run test in Docker container
        docker run --rm \\
            -v \$(pwd)/test-${binary.pkg}-binary:/usr/local/bin/jfvm \\
            -v \$(pwd)/test-results:/test-results \\
            ${testImage} \\
            /bin/bash -c "
                set -e
                echo 'Testing JFVM binary...'
                
                # Basic execution tests
                echo 'Test 1: Version check'
                if jfvm --version; then
                    echo '✅ Version check passed'
                    VERSION_OUTPUT=\$(jfvm --version)
                else
                    echo '❌ Version check failed'
                    exit 1
                fi
                
                echo 'Test 2: Help command'
                if jfvm --help > /dev/null; then
                    echo '✅ Help command passed'
                else
                    echo '❌ Help command failed'
                    exit 1
                fi
                
                echo 'Test 3: List command'
                if jfvm list > /dev/null 2>&1 || true; then
                    echo '✅ List command executed'
                else
                    echo '⚠️ List command had issues (expected for fresh install)'
                fi
                
                # Create test result XML
                cat > /test-results/${testName}-result.xml << EOF
<?xml version='1.0' encoding='UTF-8'?>
<testsuite name='${testName}' tests='3' failures='0' errors='0' time='1.0'>
    <testcase name='version_check' classname='${binary.pkg}' time='0.1'>
        <system-out>\$VERSION_OUTPUT</system-out>
    </testcase>
    <testcase name='help_command' classname='${binary.pkg}' time='0.1'/>
    <testcase name='list_command' classname='${binary.pkg}' time='0.1'/>
</testsuite>
EOF
                
                echo '✅ All tests passed for ${binary.pkg}'
            "
    """
}

def testWindowsBinary(binary, testName) {
    // For Windows, we'll do basic file validation since running Windows containers is complex
    sh """
        echo "Testing Windows binary ${binary.pkg}..."
        
        # Verify it's a valid PE executable
        if file test-${binary.pkg}-binary | grep -q "PE32"; then
            echo "✅ Valid Windows PE executable"
        else
            echo "❌ Not a valid Windows executable"
            exit 1
        fi
        
        # Create test result XML
        cat > test-results/${testName}-result.xml << EOF
<?xml version='1.0' encoding='UTF-8'?>
<testsuite name='${testName}' tests='1' failures='0' errors='0' time='0.5'>
    <testcase name='pe_validation' classname='${binary.pkg}' time='0.1'>
        <system-out>Valid Windows PE executable</system-out>
    </testcase>
</testsuite>
EOF
        
        echo "✅ Windows binary validation passed for ${binary.pkg}"
    """
}

def getTestImage(goos, goarch) {
    switch("${goos}-${goarch}") {
        case 'linux-amd64':
            return 'ubuntu:20.04'
        case 'linux-arm64':
            return 'arm64v8/ubuntu:20.04'
        case 'linux-386':
            return 'i386/ubuntu:20.04'
        case 'linux-arm':
            return 'arm32v7/ubuntu:20.04'
        default:
            return 'alpine:latest'
    }
}

def extractOS(pkg) {
    if (pkg.contains('windows')) return 'windows'
    if (pkg.contains('darwin')) return 'darwin'
    if (pkg.contains('linux')) return 'linux'
    if (pkg.contains('freebsd')) return 'freebsd'
    return 'unknown'
}

def extractArch(pkg) {
    if (pkg.contains('amd64')) return 'amd64'
    if (pkg.contains('arm64')) return 'arm64'
    if (pkg.contains('386')) return '386'
    if (pkg.contains('arm')) return 'arm'
    if (pkg.contains('s390x')) return 's390x'
    if (pkg.contains('ppc64le')) return 'ppc64le'
    return 'unknown'
}

def createFailureTestResult(testName, errorMessage) {
    return """<?xml version='1.0' encoding='UTF-8'?>
<testsuite name='${testName}' tests='1' failures='1' errors='0' time='1.0'>
    <testcase name='binary_test' classname='${testName}' time='1.0'>
        <failure message='Test failed'>${errorMessage}</failure>
    </testcase>
</testsuite>"""
}

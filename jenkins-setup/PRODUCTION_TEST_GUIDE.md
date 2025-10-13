# Production Jenkins Testing Guide

Complete guide to test your production-ready Jenkinsfile locally with SCM integration.

## 🎯 Overview

This guide helps you test the **exact production release process** locally using:
- Your local Jenkins (Docker)
- Your local Artifactory 
- Production Jenkinsfile with local adaptations
- Complete architecture matrix (10+ platforms)
- Full packaging simulation

## 🚀 Step-by-Step Setup

### **Step 1: Ensure Environment is Running**

```bash
cd jenkins-setup
./verify-setup.sh
```

**Expected output:**
- ✅ Jenkins accessible at http://localhost:8080
- ✅ Artifactory accessible at http://localhost:8082
- ✅ Authentication works (admin/password)

### **Step 2: Create Production Test Pipeline**

1. **Open Jenkins**: http://localhost:8080
2. **Login**: `admin` / `password`
3. **New Item**:
   - **Name**: `JFVM-Production-Test`
   - **Type**: `Pipeline`
   - **Click "OK"**

### **Step 3: Configure SCM Pipeline**

In the **Pipeline** section:

#### **Definition Settings:**
- **Definition**: `Pipeline script from SCM`
- **SCM**: `Git`

#### **Repository Configuration:**
- **Repository URL**: `file:///Users/bhanur/codebase/jfrog-cli-vm`
- **Credentials**: `None` (local file system)

#### **Branches to Build:**
- **Branch Specifier**: `*/main`
- **Or use**: `*/${GIT_BRANCH}` for parameter-based branch selection

#### **Repository Browser:**
- **Type**: `Auto` (or leave empty)

#### **Script Path:**
- **Script Path**: `Jenkinsfile.local`

#### **Additional Behaviors (Optional):**
- **Check**: `Lightweight checkout` for faster checkouts
- **Advanced**: `Clean before checkout` to ensure clean builds

### **Step 4: Configure Build Parameters**

Click **"This project is parameterized"** and add:

#### **Git Parameters:**
1. **String Parameter**:
   - **Name**: `GIT_BRANCH`
   - **Default**: `main`
   - **Description**: `Git branch to checkout and build`

2. **String Parameter**:
   - **Name**: `RELEASE_VERSION`
   - **Default**: `v0.0.11-test`
   - **Description**: `Test release version (use v0.0.11-test for testing)`

#### **Build Control:**
3. **Boolean Parameter**:
   - **Name**: `PRODUCTION_MODE`
   - **Default**: `true`
   - **Description**: `Enable production mode (full packaging, signing simulation)`

4. **Boolean Parameter**:
   - **Name**: `SKIP_PACKAGING`
   - **Default**: `false`
   - **Description**: `Skip package creation (NPM, Chocolatey, etc.)`

5. **Boolean Parameter**:
   - **Name**: `SKIP_TESTS`
   - **Default**: `false`
   - **Description**: `Skip cross-platform testing`

**Click "Save"**

### **Step 5: Run Production Test Build**

1. **Click "Build with Parameters"**

2. **Set Test Parameters**:
   - **GIT_BRANCH**: `main`
   - **RELEASE_VERSION**: `v0.0.11-test`
   - **PRODUCTION_MODE**: ✅ `true`
   - **SKIP_PACKAGING**: ❌ `false` (test full packaging)
   - **SKIP_TESTS**: ❌ `false` (run all tests)

3. **Click "Build"**

### **Step 6: Monitor Build Progress**

**Build will take 5-10 minutes and execute these stages:**

#### **Stage 1: Checkout and Setup** (30s)
- ✅ Checks out your JFVM code from specified branch
- ✅ Determines version from Git tags or uses provided version
- ✅ Sets up Go environment and downloads dependencies

#### **Stage 2: Build All Architectures** (2-3 minutes)
- ✅ Builds 10 architectures in parallel:
  - `jfvm-windows-amd64`
  - `jfvm-linux-386`
  - `jfvm-linux-amd64`
  - `jfvm-linux-arm64`
  - `jfvm-linux-arm`
  - `jfvm-mac-amd64`
  - `jfvm-mac-arm64`
  - `jfvm-linux-s390x`
  - `jfvm-linux-ppc64le`
  - `jfvm-freebsd-amd64`

#### **Stage 3: Sign Binaries** (30s)
- ✅ Simulates production signing process
- ✅ Copies binaries to `dist/signed/` directory

#### **Stage 4: Create Packages** (1-2 minutes)
- ✅ **NPM Package**: Creates `package.json` and tarball
- ✅ **Chocolatey Package**: Creates `.nuspec` and install scripts
- ✅ **Debian Packages**: Creates `.deb` files for amd64/arm64
- ✅ **RPM Packages**: Creates `.rpm` files for x86_64/aarch64

#### **Stage 5: Upload to Artifactory** (1 minute)
- ✅ Uploads all binaries to `jfvm-binaries/jfvm/v1/{version}/`
- ✅ Uploads packages to respective repositories
- ✅ Uploads checksums for integrity verification

#### **Stage 6: Cross-Platform Testing** (1 minute)
- ✅ Tests binary execution where possible
- ✅ Validates binary formats
- ✅ Generates JUnit test reports

#### **Stage 7: Build Summary**
- ✅ Shows complete build statistics
- ✅ Provides links to artifacts in Artifactory

## 📦 Expected Results

### **Artifactory Repository Structure:**
```
jfvm-binaries/jfvm/v1/v0.0.11-test/
├── jfvm-windows-amd64/
│   ├── jfvm.exe
│   └── jfvm.exe.sha256
├── jfvm-linux-amd64/
│   ├── jfvm
│   └── jfvm.sha256
├── jfvm-mac-amd64/
│   ├── jfvm
│   └── jfvm.sha256
├── jfvm-mac-arm64/
│   ├── jfvm
│   └── jfvm.sha256
└── ... (all 10 platforms)

jfvm-npm/v1/
└── jfvm-{version}.tgz

jfvm-debs/
├── jfvm_{version}_amd64.deb
└── jfvm_{version}_arm64.deb

jfvm-rpms/
├── jfvm-{version}.x86_64.rpm
└── jfvm-{version}.aarch64.rpm
```

### **Jenkins Artifacts:**
- All binaries archived in Jenkins job
- Test reports available in Jenkins
- Build logs with detailed progress

## 🔍 Monitoring and Verification

### **During Build:**
- **Console Output**: http://localhost:8080/job/JFVM-Production-Test/{buildNumber}/console
- **Pipeline View**: http://localhost:8080/job/JFVM-Production-Test/{buildNumber}/pipeline-graph/
- **Real-time Progress**: Watch each stage complete

### **After Build:**
- **Jenkins Artifacts**: http://localhost:8080/job/JFVM-Production-Test/{buildNumber}/artifact/
- **Artifactory Binaries**: http://localhost:8082/ui/repos/tree/General/jfvm-binaries
- **Test Reports**: http://localhost:8080/job/JFVM-Production-Test/{buildNumber}/testReport/

## 🛠️ Troubleshooting

### **Common Issues:**

#### **SCM Checkout Fails**
```bash
# Check repository access
ls -la /Users/bhanur/codebase/jfrog-cli-vm/.git

# Verify Jenkins can access the path
docker exec jenkins ls -la /Users/bhanur/codebase/jfrog-cli-vm
```

#### **Artifactory Connection Issues**
```bash
# Test from Jenkins container
docker exec jenkins curl -f http://host.docker.internal:8082/artifactory/api/system/ping
```

#### **Build Failures**
- Check Go version in container: `docker exec jenkins go version`
- Verify dependencies: Check console output for `go mod download` errors
- Architecture issues: Some exotic architectures may fail (expected)

### **Success Indicators:**
- ✅ All or most architectures build successfully
- ✅ Artifacts appear in Artifactory
- ✅ Test reports generated
- ✅ No critical errors in console output

## 🎉 Production Readiness Validation

After successful local testing:

### **✅ Validated:**
- Architecture build matrix works
- Artifactory upload process functions
- Repository structure is correct
- Packaging workflows operate
- Cross-platform testing executes

### **🚀 Ready for Production:**
- Switch to real `Jenkinsfile` on production Jenkins
- Update Artifactory URL to production instance
- Enable real code signing
- Activate real package publishing

**Your production release process is now thoroughly tested!** 🎯
